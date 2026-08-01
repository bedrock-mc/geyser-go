package bridge

import (
	"fmt"
	"log/slog"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/bedrock-mc/geyser-go/translate"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Config struct {
	BedrockListen string
	JavaAddress   string
	JavaProfile   javaprotocol.Profile

	AuthenticationDisabled bool
	Snappy                 bool
	MaxDecompressedLen     int
	BatchObserver          packet.BatchEncodeObserver
	Logger                 *slog.Logger
	Translator             Translator
}

func (c *Config) setDefaults() {
	if c.BedrockListen == "" {
		c.BedrockListen = "0.0.0.0:19132"
	}
	if c.JavaProfile.Name == "" {
		c.JavaProfile = javaprotocol.Java1214
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.MaxDecompressedLen == 0 {
		c.MaxDecompressedLen = 16 * 1024 * 1024
	}
	if c.Translator == nil {
		c.Translator = translate.NewBasic(c.JavaProfile, c.Logger)
	}
}

func (c Config) validate() error {
	if c.JavaAddress == "" {
		return fmt.Errorf("bridge: Java address is required")
	}
	if err := c.JavaProfile.Validate(); err != nil {
		return err
	}
	if c.MaxDecompressedLen < 0 {
		return fmt.Errorf("bridge: maximum decompressed length cannot be negative")
	}
	return nil
}
