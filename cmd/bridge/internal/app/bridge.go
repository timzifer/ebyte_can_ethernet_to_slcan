package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
)

type Bridge struct {
	cfg Config

	mu      sync.RWMutex
	clients map[string]*client

	logger Logger

	packetConn net.PacketConn
}

type client struct {
	addr     net.Addr
	version  int
	lastSeen time.Time
}

func New(cfg Config) (*Bridge, error) {
	logger, err := NewLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	return &Bridge{
		cfg:     cfg,
		clients: make(map[string]*client),
		logger:  logger,
	}, nil
}

func (b *Bridge) Run(ctx context.Context) error {
	packetConn, err := net.ListenPacket("udp", b.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", b.cfg.ListenAddress, err)
	}
	defer packetConn.Close()
	b.packetConn = packetConn
	b.logger.Infof("CANserver UDP server listening on %s", packetConn.LocalAddr())

	errCh := make(chan error, 2)

	go func() {
		errCh <- b.runAdapterLoop(ctx)
	}()

	go func() {
		errCh <- b.handleClientPackets(ctx, packetConn)
	}()

	cleanupTicker := time.NewTicker(5 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.logger.Infof("context cancelled")
			return nil
		case err := <-errCh:
			return err
		case <-cleanupTicker.C:
			b.cleanupClients()
		}
	}
}

func (b *Bridge) handleClientPackets(ctx context.Context, conn net.PacketConn) error {
	buf := make([]byte, 1024)
	for {
		if ctx.Err() != nil {
			return nil
		}

		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read packet: %w", err)
		}

		b.processClientDatagram(conn, addr, buf[:n])
	}
}

func (b *Bridge) processClientDatagram(conn net.PacketConn, addr net.Addr, data []byte) {
	if len(data) == 0 {
		return
	}

	switch {
	case bytes.EqualFold(data, []byte("hello")):
		b.registerClient(conn, addr, 1)
	case bytes.EqualFold(data, []byte("ehllo")):
		b.registerClient(conn, addr, 2)
	case bytes.EqualFold(data, []byte("bye")):
		b.unregisterClient(addr)
	default:
		b.touchClient(addr)
	}
}

func (b *Bridge) registerClient(conn net.PacketConn, addr net.Addr, version int) {
	key := addr.String()

	var isNew bool

	b.mu.Lock()
	c, ok := b.clients[key]
	if !ok {
		c = &client{addr: addr}
		b.clients[key] = c
		isNew = true
	}
	prevVersion := c.version
	c.version = version
	c.lastSeen = time.Now()
	b.mu.Unlock()

	if isNew {
		b.logger.Infof("client connected: %s (protocol v%d)", addr, version)
		if err := b.sendCANserverAck(conn, addr); err != nil {
			b.logger.Warnf("failed to send ack to %s: %v", addr, err)
		}
	} else if prevVersion != version {
		b.logger.Infof("client %s switched to protocol v%d", addr, version)
	}
}

func (b *Bridge) unregisterClient(addr net.Addr) {
	key := addr.String()
	b.mu.Lock()
	if _, ok := b.clients[key]; ok {
		delete(b.clients, key)
		b.logger.Infof("client disconnected: %s", addr)
	}
	b.mu.Unlock()
}

func (b *Bridge) touchClient(addr net.Addr) {
	key := addr.String()
	b.mu.Lock()
	if c, ok := b.clients[key]; ok {
		c.lastSeen = time.Now()
	}
	b.mu.Unlock()
}

func (b *Bridge) cleanupClients() {
	cutoff := time.Now().Add(-10 * time.Second)

	b.mu.Lock()
	for key, c := range b.clients {
		if c.lastSeen.Before(cutoff) {
			delete(b.clients, key)
			b.logger.Infof("client timed out: %s", c.addr)
		}
	}
	b.mu.Unlock()
}

func (b *Bridge) sendCANserverAck(conn net.PacketConn, addr net.Addr) error {
	ack := make([]byte, 16)
	binary.LittleEndian.PutUint32(ack[0:4], uint32(0x006<<21))
	binary.LittleEndian.PutUint32(ack[4:8], uint32(15<<4))
	_, err := conn.WriteTo(ack, addr)
	return err
}

func (b *Bridge) broadcastFrame(frame ebyte.Frame) {
	if b.packetConn == nil {
		return
	}

	data, err := encodeCANserverFrame(frame)
	if err != nil {
		b.logger.Warnf("unable to encode frame: %v", err)
		return
	}

	b.mu.RLock()
	clients := make([]*client, 0, len(b.clients))
	for _, c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	for _, c := range clients {
		if time.Since(c.lastSeen) > 10*time.Second {
			continue
		}
		if _, err := b.packetConn.WriteTo(data, c.addr); err != nil {
			b.logger.Warnf("failed to send frame to %s: %v", c.addr, err)
		}
	}
}

func encodeCANserverFrame(frame ebyte.Frame) ([]byte, error) {
	if frame.DLC > 8 {
		return nil, fmt.Errorf("invalid DLC %d", frame.DLC)
	}
	if frame.Extended {
		return nil, fmt.Errorf("extended frames are not supported")
	}

	header1 := frame.ID << 21
	header2 := uint32(frame.DLC & 0x0F)

	buf := make([]byte, 16)
	binary.LittleEndian.PutUint32(buf[0:4], header1)
	binary.LittleEndian.PutUint32(buf[4:8], header2)
	copy(buf[8:], frame.Data[:])
	return buf, nil
}

func (b *Bridge) runAdapterLoop(ctx context.Context) error {
	for {
		if err := b.connectAndServe(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			b.logger.Warnf("adapter loop error: %v", err)
			time.Sleep(b.cfg.ReconnectDelay)
			continue
		}
		return nil
	}
}

func (b *Bridge) connectAndServe(ctx context.Context) error {
	conn, err := net.Dial("tcp", b.cfg.EByteAddress)
	if err != nil {
		return fmt.Errorf("dial adapter: %w", err)
	}
	b.logger.Infof("connected to adapter at %s", conn.RemoteAddr())
	defer func() {
		_ = conn.Close()
		b.logger.Infof("disconnected from adapter")
	}()

	buf := make([]byte, 4096)
	frameBuf := make([]byte, 0, 4096)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return fmt.Errorf("adapter read: %w", err)
		}

		frameBuf = append(frameBuf, buf[:n]...)
		for len(frameBuf) >= ebyte.FrameSize {
			frameBytes := frameBuf[:ebyte.FrameSize]
			frameBuf = frameBuf[ebyte.FrameSize:]
			frame, err := ebyte.ParseFrame(frameBytes)
			if err != nil {
				b.logger.Warnf("discarding invalid frame: %v", err)
				continue
			}
			b.broadcastFrame(frame)
		}
	}
}
