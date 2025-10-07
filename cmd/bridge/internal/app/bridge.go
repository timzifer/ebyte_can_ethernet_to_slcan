package app

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
)

type Bridge struct {
	cfg Config

	mu      sync.RWMutex
	clients map[*client]struct{}

	logger Logger

	listener net.Listener
	start    time.Time
}

type client struct {
	conn      net.Conn
	sendCh    chan []byte
	done      chan struct{}
	closeOnce sync.Once
	remote    string
}

type gvretParserState int

const (
	gvretStateIdle gvretParserState = iota
	gvretStateExpectCommand
	gvretStateClassicFrame
	gvretStateFdFrame
	gvretStateSkip
)

type gvretClientState struct {
	binary    bool
	e7Count   int
	state     gvretParserState
	step      int
	remaining int
	fdLength  int
}

func New(cfg Config) (*Bridge, error) {
	logger, err := NewLogger(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	return &Bridge{
		cfg:     cfg,
		clients: make(map[*client]struct{}),
		logger:  logger,
		start:   time.Now(),
	}, nil
}

func (b *Bridge) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", b.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", b.cfg.ListenAddress, err)
	}
	defer listener.Close()
	b.listener = listener
	b.logger.Infof("GVRET TCP server listening on %s", listener.Addr())

	errCh := make(chan error, 2)

	go func() {
		errCh <- b.runAdapterLoop(ctx)
	}()

	go func() {
		errCh <- b.acceptClients(ctx, listener)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (b *Bridge) acceptClients(ctx context.Context, listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return fmt.Errorf("accept client: %w", err)
		}

		b.logger.Infof("client connected: %s", conn.RemoteAddr())
		go b.handleClient(ctx, conn)
	}
}

func (b *Bridge) handleClient(ctx context.Context, conn net.Conn) {
	c := newClient(conn)
	b.addClient(c)
	defer func() {
		b.removeClient(c)
		b.logger.Infof("client disconnected: %s", c.remote)
	}()

	clientCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.writer(clientCtx, b.logger)

	state := gvretClientState{}
	buf := make([]byte, 1024)

	for {
		if ctx.Err() != nil {
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if errors.Is(err, io.EOF) {
				return
			}
			if ctx.Err() != nil {
				return
			}
			b.logger.Warnf("client %s read error: %v", c.remote, err)
			return
		}

		for i := 0; i < n; i++ {
			b.processGVRETByte(c, &state, buf[i])
		}
	}
}

func newClient(conn net.Conn) *client {
	return &client{
		conn:   conn,
		sendCh: make(chan []byte, 128),
		done:   make(chan struct{}),
		remote: conn.RemoteAddr().String(),
	}
}

func (c *client) writer(ctx context.Context, logger Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case data := <-c.sendCh:
			if len(data) == 0 {
				continue
			}
			if _, err := c.conn.Write(data); err != nil {
				if logger != nil {
					logger.Debugf("write to %s failed: %v", c.remote, err)
				}
				c.close()
				return
			}
		}
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *client) enqueue(payload []byte) {
	data := append([]byte(nil), payload...)
	select {
	case <-c.done:
		return
	default:
	}

	select {
	case c.sendCh <- data:
	case <-c.done:
	default:
	}
}

func (c *client) enqueuePriority(payload []byte) {
	data := append([]byte(nil), payload...)
	select {
	case <-c.done:
		return
	default:
	}

	select {
	case c.sendCh <- data:
	case <-c.done:
	}
}

func (b *Bridge) addClient(c *client) {
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
}

func (b *Bridge) removeClient(c *client) {
	b.mu.Lock()
	if _, ok := b.clients[c]; ok {
		delete(b.clients, c)
	}
	b.mu.Unlock()
	c.close()
}

func (b *Bridge) processGVRETByte(c *client, state *gvretClientState, by byte) {
	if !state.binary {
		if by == 0xE7 {
			state.e7Count++
			if state.e7Count >= 2 {
				state.binary = true
				state.state = gvretStateIdle
				state.e7Count = 0
				b.logger.Debugf("client %s switched to GVRET binary mode", c.remote)
			}
		} else {
			state.e7Count = 0
		}
		return
	}

	switch state.state {
	case gvretStateIdle:
		if by == 0xF1 {
			state.state = gvretStateExpectCommand
		}
	case gvretStateExpectCommand:
		state.state = gvretStateIdle
		state.step = 0
		b.handleGVRETCommandByte(c, state, by)
	case gvretStateClassicFrame:
		state.step++
		switch state.step {
		case 1, 2, 3, 4, 5, 6, 7, 8:
			// header bytes: timestamp and identifier
		case 9:
			length := int(by & 0x0F)
			state.remaining = length + 1 // payload + terminator
			if state.remaining <= 0 {
				state.state = gvretStateIdle
				state.step = 0
			} else {
				state.state = gvretStateSkip
			}
		default:
			state.state = gvretStateIdle
			state.step = 0
		}
	case gvretStateFdFrame:
		state.step++
		switch state.step {
		case 1, 2, 3, 4, 5, 6, 7, 8:
			// timestamp and identifier bytes
		case 9:
			state.fdLength = int(by & 0x3F)
		case 10:
			state.remaining = state.fdLength + 1 // data + trailing byte
			if state.remaining <= 0 {
				state.state = gvretStateIdle
				state.step = 0
			} else {
				state.state = gvretStateSkip
			}
		default:
			state.state = gvretStateIdle
			state.step = 0
		}
	case gvretStateSkip:
		state.remaining--
		if state.remaining <= 0 {
			state.state = gvretStateIdle
			state.step = 0
		}
	}
}

func (b *Bridge) handleGVRETCommandByte(c *client, state *gvretClientState, cmd byte) {
	switch cmd {
	case 0x00:
		state.state = gvretStateClassicFrame
		state.step = 0
	case 0x01:
		b.sendGVRETTimeSync(c)
	case 0x06:
		b.sendGVRETBusParams(c)
	case 0x07:
		b.sendGVRETDeviceInfo(c)
	case 0x09:
		b.sendGVRETValidationAck(c)
	case 0x0C:
		b.sendGVRETNumBuses(c)
	case 0x0D:
		b.sendGVRETExtendedBusInfo(c)
	case 0x05:
		state.state = gvretStateSkip
		state.remaining = 9
	case 0x08:
		state.state = gvretStateSkip
		state.remaining = 2
	case 0x14:
		state.state = gvretStateFdFrame
		state.step = 0
		state.fdLength = 0
	default:
		state.state = gvretStateIdle
	}
}

func (b *Bridge) sendGVRETTimeSync(c *client) {
	ts := b.gvretTimestamp()
	payload := []byte{0xF1, 0x01, byte(ts), byte(ts >> 8), byte(ts >> 16), byte(ts >> 24)}
	c.enqueuePriority(payload)
}

func (b *Bridge) sendGVRETBusParams(c *client) {
	bitrate := b.cfg.BusBitrate
	payload := []byte{
		0xF1, 0x06,
		0x01,
		byte(bitrate), byte(bitrate >> 8), byte(bitrate >> 16), byte(bitrate >> 24),
		0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	c.enqueuePriority(payload)
}

func (b *Bridge) sendGVRETDeviceInfo(c *client) {
	payload := []byte{0xF1, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
	c.enqueuePriority(payload)
}

func (b *Bridge) sendGVRETNumBuses(c *client) {
	payload := []byte{0xF1, 0x0C, 0x01}
	c.enqueuePriority(payload)
}

func (b *Bridge) sendGVRETExtendedBusInfo(c *client) {
	payload := make([]byte, 17)
	payload[0] = 0xF1
	payload[1] = 0x0D
	c.enqueuePriority(payload)
}

func (b *Bridge) sendGVRETValidationAck(c *client) {
	payload := []byte{0xF1, 0x09}
	c.enqueuePriority(payload)
}

func (b *Bridge) gvretTimestamp() uint32 {
	elapsed := time.Since(b.start)
	return uint32(elapsed / time.Microsecond)
}

func (b *Bridge) broadcastFrame(frame ebyte.Frame) {
	data, err := encodeGVRETFrame(frame, b.gvretTimestamp(), 0)
	if err != nil {
		b.logger.Warnf("unable to encode GVRET frame: %v", err)
		return
	}

	b.mu.RLock()
	clients := make([]*client, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	for _, c := range clients {
		c.enqueue(data)
	}
}

func encodeGVRETFrame(frame ebyte.Frame, timestamp uint32, bus uint8) ([]byte, error) {
	if frame.DLC > 8 {
		return nil, fmt.Errorf("invalid DLC %d", frame.DLC)
	}

	id := frame.ID
	if frame.Extended || frame.ID > 0x7FF {
		id |= 1 << 31
	}
	if frame.Remote {
		id |= 1 << 30
	}

	buf := make([]byte, 0, 13+int(frame.DLC))
	buf = append(buf, 0xF1, 0x00)

	tsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tsBytes, timestamp)
	buf = append(buf, tsBytes...)

	idBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(idBytes, id)
	buf = append(buf, idBytes...)

	lengthByte := byte(frame.DLC & 0x0F)
	lengthByte |= (bus & 0x0F) << 4
	buf = append(buf, lengthByte)

	if frame.DLC > 0 {
		buf = append(buf, frame.Data[:frame.DLC]...)
	}

	buf = append(buf, 0x00)
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
