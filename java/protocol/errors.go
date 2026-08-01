package protocol

import "errors"

var (
	ErrFrameTooLarge        = errors.New("java protocol: packet frame exceeds configured limit")
	ErrDecompressedTooLarge = errors.New("java protocol: decompressed packet exceeds configured limit")
	ErrMalformedVarInt      = errors.New("java protocol: malformed VarInt")
	ErrMalformedVarLong     = errors.New("java protocol: malformed VarLong")
	ErrMalformedString      = errors.New("java protocol: malformed string")
	ErrEncryptionKey        = errors.New("java protocol: AES key must be 16 bytes")
	ErrOnlineModeRequired   = errors.New("java protocol: server requires online-mode encryption")
	ErrUnsupportedProtocol  = errors.New("java protocol: unsupported protocol profile")
	ErrUnexpectedPacket     = errors.New("java protocol: unexpected packet")
)
