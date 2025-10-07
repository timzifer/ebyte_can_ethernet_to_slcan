package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/ebyte"
	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/slcan"
)

type Bridge struct {
	cfg Config

	mu      sync.RWMutex
	clients map[*client]struct{}

	logger Logger
}

type client struct {
	conn net.Conn
	send chan string

	mu     sync.Mutex
	open   bool
	closed bool
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
	}, nil
}

func (b *Bridge) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- b.runAdapterLoop(ctx)
	}()

	listener, err := net.Listen("tcp", b.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", b.cfg.ListenAddress, err)
	}
	defer listener.Close()
	b.logger.Infof("SLCAN server listening on %s", listener.Addr())

	go func() {
		errCh <- b.acceptClients(ctx, listener)
	}()

	select {
	case <-ctx.Done():
		b.logger.Infof("context cancelled")
		return nil
	case err := <-errCh:
		return err
	}
}

func (b *Bridge) acceptClients(ctx context.Context, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				b.logger.Warnf("temporary accept error: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}

		b.logger.Infof("client connected: %s", conn.RemoteAddr())
		c := &client{
			conn: conn,
			send: make(chan string, 16),
		}

		b.mu.Lock()
		b.clients[c] = struct{}{}
		b.mu.Unlock()

		go b.runClient(ctx, c)
	}
}

func (b *Bridge) runClient(ctx context.Context, c *client) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		b.handleClientRead(ctx, c)
	}()

	go func() {
		defer wg.Done()
		b.handleClientWrite(ctx, c)
	}()

	wg.Wait()

	b.mu.Lock()
	delete(b.clients, c)
	b.mu.Unlock()

	_ = c.conn.Close()
	b.logger.Infof("client disconnected: %s", c.conn.RemoteAddr())
}

func (b *Bridge) handleClientRead(ctx context.Context, c *client) {
	defer func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.send)
	}()

	scanner := bufio.NewScanner(c.conn)
	scanner.Split(splitSLCAN)

	for scanner.Scan() {
		line := scanner.Text()
		cmd := slcan.ParseCommand(line)

		switch cmd.Type {
		case slcan.CommandOpen:
			c.mu.Lock()
			c.open = true
			c.mu.Unlock()
			b.logger.Debugf("client %s requested bus open", c.conn.RemoteAddr())
			c.send <- "\r"
		case slcan.CommandClose:
			c.mu.Lock()
			c.open = false
			c.mu.Unlock()
			b.logger.Debugf("client %s requested bus close", c.conn.RemoteAddr())
			c.send <- "\r"
		default:
			b.logger.Debugf("client %s sent unsupported command %q", c.conn.RemoteAddr(), line)
			c.send <- "\a"
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		b.logger.Warnf("client read error: %v", err)
	}
}

func (b *Bridge) handleClientWrite(ctx context.Context, c *client) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if _, err := io.WriteString(c.conn, msg); err != nil {
				b.logger.Warnf("client write error: %v", err)
				return
			}
		}
	}
}

func splitSLCAN(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == '\r' || b == '\n' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func (b *Bridge) broadcastFrame(frame ebyte.Frame) {
	msg := slcan.EncodeFrame(frame)

	b.mu.RLock()
	defer b.mu.RUnlock()

	for c := range b.clients {
		c.mu.Lock()
		open := c.open
		closed := c.closed
		c.mu.Unlock()
		if !open || closed {
			continue
		}

		select {
		case c.send <- msg:
		default:
			b.logger.Warnf("dropping frame for client %s due to slow consumer", c.conn.RemoteAddr())
		}
	}
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
