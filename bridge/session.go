package bridge

import (
	"context"
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
)

// Translator is the explicit ownership boundary between the Bedrock and Java
// protocols. It is intentionally typed and session-scoped: packet bytes must
// never be copied between editions without version-aware translation.
type Translator interface {
	Bootstrap(context.Context, *minecraft.Conn, *javaprotocol.Client) error
	Run(context.Context, *minecraft.Conn, *javaprotocol.Client) error
}

type Session struct {
	Bedrock    *minecraft.Conn
	Java       *javaprotocol.Client
	JavaConfig javaprotocol.Profile
	Translator Translator
}

func (s *Session) Run(ctx context.Context, javaAddress, username string, playerUUID *[16]byte) error {
	if s.Translator == nil {
		return ErrTranslatorUnavailable
	}
	if s.Bedrock == nil {
		return fmt.Errorf("bridge: session has no Bedrock connection")
	}
	javaClient, err := javaprotocol.DialAndLogin(ctx, javaAddress, s.JavaConfig, username, playerUUID)
	if err != nil {
		return err
	}
	s.Java = javaClient
	defer func() { _ = javaClient.Close() }()
	if err := s.Translator.Bootstrap(ctx, s.Bedrock, javaClient); err != nil {
		return err
	}
	return s.Translator.Run(ctx, s.Bedrock, javaClient)
}
