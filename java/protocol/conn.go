package protocol

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

type State uint8

const (
	StateHandshaking State = iota
	StateStatus
	StateLogin
	StateConfiguration
	StatePlay
)

type Packet struct {
	ID   int32
	Data []byte
}

type ConnConfig struct {
	MaxPacketSize       int
	MaxDecompressedSize int
	MaxStringSize       int
}

type Conn struct {
	rwc net.Conn

	readMu  sync.Mutex
	writeMu sync.Mutex
	stateMu sync.RWMutex
	state   State

	reader io.Reader
	writer io.Writer

	maxPacketSize       int
	maxDecompressedSize int
	maxStringSize       int
	compression         int
}

func NewConn(conn net.Conn, cfg ConnConfig) *Conn {
	maxPacketSize := cfg.MaxPacketSize
	if maxPacketSize <= 0 {
		maxPacketSize = DefaultMaxPacketSize
	}
	maxDecompressedSize := cfg.MaxDecompressedSize
	if maxDecompressedSize <= 0 {
		maxDecompressedSize = maxPacketSize
	}
	maxStringSize := cfg.MaxStringSize
	if maxStringSize <= 0 {
		maxStringSize = DefaultMaxStringSize
	}
	return &Conn{
		rwc:                 conn,
		reader:              conn,
		writer:              conn,
		state:               StateHandshaking,
		maxPacketSize:       maxPacketSize,
		maxDecompressedSize: maxDecompressedSize,
		maxStringSize:       maxStringSize,
		compression:         -1,
	}
}

func Dial(ctx context.Context, address string, cfg ConnConfig) (*Conn, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return NewConn(conn, cfg), nil
}

func (c *Conn) NetConn() net.Conn { return c.rwc }

func (c *Conn) State() State {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *Conn) SetState(state State) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}

func (c *Conn) SetCompression(threshold int) error {
	if threshold < 0 {
		c.compression = -1
		return nil
	}
	if threshold > c.maxDecompressedSize {
		return fmt.Errorf("java protocol: compression threshold %d exceeds packet limit %d", threshold, c.maxDecompressedSize)
	}
	c.compression = threshold
	return nil
}

func (c *Conn) CompressionThreshold() int { return c.compression }

func (c *Conn) EnableEncryption(key []byte) error {
	enc, err := newCFB8(key, false)
	if err != nil {
		return err
	}
	dec, err := newCFB8(key, true)
	if err != nil {
		return err
	}
	c.reader = &streamReader{r: c.reader, stream: dec}
	c.writer = &streamWriter{w: c.writer, stream: enc}
	return nil
}

func (c *Conn) ReadPacket() (Packet, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	frameLength, err := ReadVarInt(c.reader)
	if err != nil {
		return Packet{}, err
	}
	length, err := requireNonNegativeLength(frameLength, c.maxPacketSize, "frame")
	if err != nil {
		if frameLength > 0 {
			return Packet{}, ErrFrameTooLarge
		}
		return Packet{}, err
	}
	frame := make([]byte, length)
	if _, err := io.ReadFull(c.reader, frame); err != nil {
		return Packet{}, err
	}
	payload, err := c.decodeFrame(frame)
	if err != nil {
		return Packet{}, err
	}
	reader := NewReader(payload)
	reader.SetMaxStringSize(c.maxStringSize)
	id, err := reader.VarInt()
	if err != nil {
		return Packet{}, fmt.Errorf("java protocol: read packet id: %w", err)
	}
	data, err := reader.RawRemaining()
	if err != nil {
		return Packet{}, fmt.Errorf("java protocol: read packet payload: %w", err)
	}
	return Packet{ID: id, Data: data}, nil
}

func (c *Conn) decodeFrame(frame []byte) ([]byte, error) {
	if c.compression < 0 {
		return frame, nil
	}
	reader := NewReader(frame)
	declared, err := reader.VarInt()
	if err != nil {
		return nil, fmt.Errorf("java protocol: read uncompressed length: %w", err)
	}
	data, err := reader.RawRemaining()
	if err != nil {
		return nil, err
	}
	if declared == 0 {
		return data, nil
	}
	length, err := requireNonNegativeLength(declared, c.maxDecompressedSize, "uncompressed")
	if err != nil {
		return nil, ErrDecompressedTooLarge
	}
	zipReader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("java protocol: open zlib payload: %w", err)
	}
	decoded, readErr := io.ReadAll(io.LimitReader(zipReader, int64(c.maxDecompressedSize)+1))
	closeErr := zipReader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("java protocol: decompress payload: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("java protocol: close zlib payload: %w", closeErr)
	}
	if len(decoded) > c.maxDecompressedSize || len(decoded) != length {
		return nil, ErrDecompressedTooLarge
	}
	return decoded, nil
}

func (c *Conn) WritePacket(id int32, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	var body bytes.Buffer
	if err := WriteVarInt(&body, id); err != nil {
		return err
	}
	if _, err := body.Write(data); err != nil {
		return err
	}
	payload, err := c.encodeFrame(body.Bytes())
	if err != nil {
		return err
	}
	var frame bytes.Buffer
	if err := WriteVarInt(&frame, int32(len(payload))); err != nil {
		return err
	}
	if _, err := frame.Write(payload); err != nil {
		return err
	}
	return writeFull(c.writer, frame.Bytes())
}

func (c *Conn) encodeFrame(body []byte) ([]byte, error) {
	if c.compression < 0 {
		return body, nil
	}
	var payload bytes.Buffer
	if len(body) < c.compression {
		if err := WriteVarInt(&payload, 0); err != nil {
			return nil, err
		}
		_, _ = payload.Write(body)
		return payload.Bytes(), nil
	}
	if err := WriteVarInt(&payload, int32(len(body))); err != nil {
		return nil, err
	}
	zipWriter := zlib.NewWriter(&payload)
	if _, err := zipWriter.Write(body); err != nil {
		_ = zipWriter.Close()
		return nil, err
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return payload.Bytes(), nil
}

func (c *Conn) Close() error { return c.rwc.Close() }

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
