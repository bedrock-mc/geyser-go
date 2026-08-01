package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"unicode/utf8"
)

const (
	DefaultMaxPacketSize = 2 << 20
	DefaultMaxStringSize = 32767
)

type Reader struct {
	r         io.Reader
	remaining int
	maxString int
}

func NewReader(data []byte) *Reader {
	return &Reader{r: bytes.NewReader(data), remaining: len(data), maxString: DefaultMaxStringSize}
}

func NewReaderFrom(r io.Reader, remaining int) *Reader {
	return &Reader{r: io.LimitReader(r, int64(remaining)), remaining: remaining, maxString: DefaultMaxStringSize}
}

func (r *Reader) SetMaxStringSize(limit int) {
	if limit > 0 {
		r.maxString = limit
	}
}

func (r *Reader) Remaining() int { return r.remaining }

// Read implements io.Reader while preserving the reader's byte budget.
func (r *Reader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if len(dst) > r.remaining {
		dst = dst[:r.remaining]
	}
	n, err := r.r.Read(dst)
	r.remaining -= n
	return n, err
}

func (r *Reader) readFull(dst []byte) error {
	if len(dst) > r.remaining {
		return io.ErrUnexpectedEOF
	}
	n, err := io.ReadFull(r.r, dst)
	r.remaining -= n
	return err
}

func (r *Reader) VarInt() (int32, error) { return ReadVarInt(r) }

func (r *Reader) VarLong() (int64, error) { return ReadVarLong(r) }

func (r *Reader) Bool() (bool, error) {
	b, err := r.Byte()
	return b != 0, err
}

func (r *Reader) Byte() (byte, error) {
	var b [1]byte
	if err := r.readFull(b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *Reader) Int8() (int8, error) {
	b, err := r.Byte()
	return int8(b), err
}

func (r *Reader) Uint8() (uint8, error) { return r.Byte() }

func (r *Reader) Int16() (int16, error) {
	var b [2]byte
	if err := r.readFull(b[:]); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(b[:])), nil
}

func (r *Reader) Uint16() (uint16, error) {
	v, err := r.Int16()
	return uint16(v), err
}

func (r *Reader) Int32() (int32, error) {
	var b [4]byte
	if err := r.readFull(b[:]); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(b[:])), nil
}

func (r *Reader) Int64() (int64, error) {
	var b [8]byte
	if err := r.readFull(b[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(b[:])), nil
}

func (r *Reader) Float32() (float32, error) {
	v, err := r.Int32()
	return math.Float32frombits(uint32(v)), err
}

func (r *Reader) Float64() (float64, error) {
	v, err := r.Int64()
	return math.Float64frombits(uint64(v)), err
}

func (r *Reader) Bytes(length int) ([]byte, error) {
	if length < 0 || length > r.remaining {
		return nil, io.ErrUnexpectedEOF
	}
	data := make([]byte, length)
	if err := r.readFull(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *Reader) String() (string, error) {
	length, err := r.VarInt()
	if err != nil {
		return "", err
	}
	lengthInt, err := requireNonNegativeLength(length, r.maxString, "string")
	if err != nil {
		return "", err
	}
	data, err := r.Bytes(lengthInt)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", ErrMalformedString
	}
	return string(data), nil
}

func (r *Reader) RawRemaining() ([]byte, error) {
	return r.Bytes(r.remaining)
}

type Writer struct{ buf bytes.Buffer }

func NewWriter() *Writer { return &Writer{} }

func (w *Writer) Bytes() []byte { return w.buf.Bytes() }

func (w *Writer) VarInt(value int32) error { return WriteVarInt(&w.buf, value) }

func (w *Writer) VarLong(value int64) error { return WriteVarLong(&w.buf, value) }

func (w *Writer) Bool(value bool) error {
	if value {
		return w.Byte(1)
	}
	return w.Byte(0)
}

func (w *Writer) Byte(value byte) error { return w.buf.WriteByte(value) }

func (w *Writer) Int16(value int16) error { return binary.Write(&w.buf, binary.BigEndian, value) }

func (w *Writer) Uint16(value uint16) error { return binary.Write(&w.buf, binary.BigEndian, value) }

func (w *Writer) Int32(value int32) error { return binary.Write(&w.buf, binary.BigEndian, value) }

func (w *Writer) Int64(value int64) error { return binary.Write(&w.buf, binary.BigEndian, value) }

func (w *Writer) Float32(value float32) error { return w.Int32(int32(math.Float32bits(value))) }

func (w *Writer) Float64(value float64) error { return w.Int64(int64(math.Float64bits(value))) }

func (w *Writer) BytesValue(value []byte) error {
	_, err := w.buf.Write(value)
	return err
}

func (w *Writer) String(value string) error {
	if !utf8.ValidString(value) {
		return ErrMalformedString
	}
	if len(value) > DefaultMaxStringSize {
		return fmt.Errorf("java protocol: string length %d exceeds limit %d", len(value), DefaultMaxStringSize)
	}
	if err := w.VarInt(int32(len(value))); err != nil {
		return err
	}
	return w.BytesValue([]byte(value))
}
