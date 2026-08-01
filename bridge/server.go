package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Server struct {
	config Config
	log    *slog.Logger
}

func New(config Config) (*Server, error) {
	config.setDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &Server{config: config, log: config.Logger}, nil
}

// ListenAndServe owns the Gophertunnel listener and blocks until ctx is
// cancelled or the listener fails. Gophertunnel performs the Bedrock login;
// this server receives only fully handshaken connections.
func (s *Server) ListenAndServe(ctx context.Context) error {
	compression := packet.DefaultCompression
	if s.config.Snappy {
		compression = packet.SnappyCompression
		s.log.Warn("Snappy compression enabled; validate the target Bedrock device matrix before production use")
	}
	listener, err := (minecraft.ListenConfig{
		ErrorLog:               s.log,
		AuthenticationDisabled: s.config.AuthenticationDisabled,
		AllowUnknownPackets:    true,
		AllowInvalidPackets:    false,
		Compression:            compression,
		CompressionThreshold:   256,
		MaxDecompressedLen:     s.config.MaxDecompressedLen,
		EnableBatchReading:     true,
		PacketBatchFunc:        s.config.BatchObserver,
		ConnHandler: func(conn *minecraft.Conn) error {
			return s.handleConnection(ctx, conn)
		},
	}).Listen("raknet", s.config.BedrockListen)
	if err != nil {
		return fmt.Errorf("bridge: listen on %s: %w", s.config.BedrockListen, err)
	}
	defer listener.Close()
	s.log.Info("Bedrock listener ready", "address", listener.Addr().String(), "java", s.config.JavaAddress)

	<-ctx.Done()
	return ctx.Err()
}

func (s *Server) handleConnection(ctx context.Context, bedrock *minecraft.Conn) error {
	username := playerName(bedrock.IdentityData(), bedrock.ClientData())
	s.log.Info("Bedrock player connected", "username", username, "remote", bedrock.RemoteAddr().String(), "protocol", bedrock.Proto().Ver())
	defer s.log.Info("Bedrock player disconnected", "username", username)

	session := &Session{
		Bedrock:    bedrock,
		JavaConfig: s.config.JavaProfile,
		Translator: s.config.Translator,
	}
	if err := session.Run(ctx, s.config.JavaAddress, username, nil); err != nil {
		if err == ErrTranslatorUnavailable {
			_ = bedrock.Disconnect("Geyser-Go is running without a translator build")
			return err
		}
		_ = bedrock.Disconnect("Unable to connect to the Java server")
		return err
	}
	return nil
}

func playerName(identity login.IdentityData, client login.ClientData) string {
	for _, value := range []string{identity.DisplayName, client.ThirdPartyName, "GeyserPlayer"} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return "GeyserPlayer"
}

// Profile returns the immutable Java protocol profile selected by the server.
func (s *Server) Profile() javaprotocol.Profile { return s.config.JavaProfile }
