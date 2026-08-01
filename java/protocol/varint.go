package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

func ReadVarInt(r io.Reader) (int32, error) {
	var value uint32
	for shift := uint(0); shift < 35; shift += 7 {
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		value |= uint32(b[0]&0x7f) << shift
		if b[0]&0x80 == 0 {
			if shift == 28 && b[0] > 0x0f {
				return 0, ErrMalformedVarInt
			}
			return int32(value), nil
		}
	}
	return 0, ErrMalformedVarInt
}

func WriteVarInt(w io.Writer, value int32) error {
	v := uint32(value)
	for v&^0x7f != 0 {
		if err := writeByte(w, byte(v&0x7f)|0x80); err != nil {
			return err
		}
		v >>= 7
	}
	return writeByte(w, byte(v))
}

func ReadVarLong(r io.Reader) (int64, error) {
	var value uint64
	for shift := uint(0); shift < 70; shift += 7 {
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		value |= uint64(b[0]&0x7f) << shift
		if b[0]&0x80 == 0 {
			if shift == 63 && b[0] > 0x01 {
				return 0, ErrMalformedVarLong
			}
			return int64(value), nil
		}
	}
	return 0, ErrMalformedVarLong
}

func WriteVarLong(w io.Writer, value int64) error {
	v := uint64(value)
	for v&^0x7f != 0 {
		if err := writeByte(w, byte(v&0x7f)|0x80); err != nil {
			return err
		}
		v >>= 7
	}
	return writeByte(w, byte(v))
}

func writeByte(w io.Writer, b byte) error {
	var one [1]byte
	one[0] = b
	_, err := w.Write(one[:])
	return err
}

func varIntBytes(value int32) []byte {
	var buf [binary.MaxVarintLen32]byte
	n := 0
	v := uint32(value)
	for v&^0x7f != 0 {
		buf[n] = byte(v&0x7f) | 0x80
		n++
		v >>= 7
	}
	buf[n] = byte(v)
	return buf[:n+1]
}

func requireNonNegativeLength(value int32, limit int, what string) (int, error) {
	if value < 0 {
		return 0, fmt.Errorf("java protocol: negative %s length %d", what, value)
	}
	if int64(value) > int64(limit) {
		return 0, fmt.Errorf("java protocol: %s length %d exceeds limit %d", what, value, limit)
	}
	return int(value), nil
}
