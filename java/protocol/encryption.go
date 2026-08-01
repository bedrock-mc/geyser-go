package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

type cfb8 struct {
	block     cipher.Block
	state     [aes.BlockSize]byte
	keystream [aes.BlockSize]byte
	decrypt   bool
}

func newCFB8(key []byte, decrypt bool) (*cfb8, error) {
	if len(key) != 16 {
		return nil, ErrEncryptionKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := &cfb8{block: block, decrypt: decrypt}
	copy(stream.state[:], key)
	return stream, nil
}

func (s *cfb8) XORKeyStream(dst, src []byte) {
	for i, input := range src {
		s.block.Encrypt(s.keystream[:], s.state[:])
		output := input ^ s.keystream[0]
		copy(s.state[:aes.BlockSize-1], s.state[1:])
		if s.decrypt {
			s.state[aes.BlockSize-1] = input
		} else {
			s.state[aes.BlockSize-1] = output
		}
		dst[i] = output
	}
}

type streamReader struct {
	r      io.Reader
	stream *cfb8
}

func (r *streamReader) Read(dst []byte) (int, error) {
	n, err := r.r.Read(dst)
	if n > 0 {
		r.stream.XORKeyStream(dst[:n], dst[:n])
	}
	return n, err
}

type streamWriter struct {
	w      io.Writer
	stream *cfb8
}

func (w *streamWriter) Write(src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	encoded := make([]byte, len(src))
	w.stream.XORKeyStream(encoded, src)
	n, err := w.w.Write(encoded)
	if n < len(encoded) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}
