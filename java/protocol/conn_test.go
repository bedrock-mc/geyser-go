package protocol

import (
	"bytes"
	"crypto/aes"
	"io"
	"net"
	"testing"
)

func TestVarIntRoundTrip(t *testing.T) {
	values := []int32{0, 1, 127, 128, 255, 2097151, -1, -2147483648, 2147483647}
	for _, want := range values {
		var buf bytes.Buffer
		if err := WriteVarInt(&buf, want); err != nil {
			t.Fatal(err)
		}
		got, err := ReadVarInt(&buf)
		if err != nil {
			t.Fatalf("%d: %v", want, err)
		}
		if got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	}
}

func TestConnCompressionRoundTrip(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	writer := NewConn(left, ConnConfig{})
	reader := NewConn(right, ConnConfig{})
	if err := writer.SetCompression(1); err != nil {
		t.Fatal(err)
	}
	if err := reader.SetCompression(1); err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte("geyser-go"), 128)
	go func() { _ = writer.WritePacket(0x17, want) }()
	got, err := reader.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 0x17 || !bytes.Equal(got.Data, want) {
		t.Fatalf("packet mismatch: id=%#x data=%d", got.ID, len(got.Data))
	}
}

func TestCFB8MatchesKnownRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef")
	plain := []byte("Bedrock to Java")
	encrypt, err := newCFB8(key, false)
	if err != nil {
		t.Fatal(err)
	}
	encoded := make([]byte, len(plain))
	encrypt.XORKeyStream(encoded, plain)
	decrypt, err := newCFB8(key, true)
	if err != nil {
		t.Fatal(err)
	}
	decoded := make([]byte, len(encoded))
	decrypt.XORKeyStream(decoded, encoded)
	if !bytes.Equal(decoded, plain) {
		t.Fatalf("decoded %q, want %q", decoded, plain)
	}
	if aes.BlockSize != 16 {
		t.Fatal("unexpected AES block size")
	}
}

func TestDecodeFrameRejectsDeclaredSizeMismatch(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	reader := NewConn(right, ConnConfig{})
	if err := reader.SetCompression(1); err != nil {
		t.Fatal(err)
	}
	go func() {
		// Frame length 4, followed by a non-zero uncompressed length and an
		// invalid zlib stream.
		_, _ = left.Write([]byte{4, 3, 1, 2, 3})
		_ = left.Close()
	}()
	_, err := reader.ReadPacket()
	if err == nil {
		t.Fatal("expected malformed packet error")
	}
}

func TestReaderDoesNotReadPastPayload(t *testing.T) {
	r := NewReader([]byte{1})
	if _, err := r.Int64(); err != io.ErrUnexpectedEOF {
		t.Fatalf("got %v, want %v", err, io.ErrUnexpectedEOF)
	}
}
