package protocol

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestDialAndLoginNegotiatesConfiguration(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErrors := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErrors <- err
			return
		}
		defer conn.Close()
		server := NewConn(conn, ConnConfig{})
		packet, err := server.ReadPacket()
		if err != nil {
			serverErrors <- err
			return
		}
		handshake, err := DecodeHandshake(packet.Data)
		if err != nil {
			serverErrors <- err
			return
		}
		if packet.ID != 0 || handshake.NextState != 2 || handshake.ProtocolVersion != Java1214.ProtocolVersion {
			serverErrors <- fmt.Errorf("unexpected handshake: id=%d %#v", packet.ID, handshake)
			return
		}

		packet, err = server.ReadPacket()
		if err != nil {
			serverErrors <- err
			return
		}
		startReader := NewReader(packet.Data)
		username, err := startReader.String()
		if err != nil {
			serverErrors <- err
			return
		}
		if packet.ID != Java1214.LoginStartPacketID || username != "Tester" || startReader.Remaining() != 16 {
			serverErrors <- fmt.Errorf("unexpected login start: id=%d username=%q remaining=%d", packet.ID, username, startReader.Remaining())
			return
		}
		if _, err := startReader.Bytes(16); err != nil {
			serverErrors <- err
			return
		}

		compression, err := EncodeSetCompression(1)
		if err != nil {
			serverErrors <- err
			return
		}
		if err := server.WritePacket(Java1214.LoginSetCompressionPacketID, compression); err != nil {
			serverErrors <- err
			return
		}
		if err := server.SetCompression(1); err != nil {
			serverErrors <- err
			return
		}
		loginSuccess := NewWriter()
		var id [16]byte
		if err := loginSuccess.BytesValue(id[:]); err != nil {
			serverErrors <- err
			return
		}
		if err := loginSuccess.String("Tester"); err != nil {
			serverErrors <- err
			return
		}
		if err := loginSuccess.VarInt(0); err != nil {
			serverErrors <- err
			return
		}
		if err := server.WritePacket(Java1214.LoginSuccessPacketID, loginSuccess.Bytes()); err != nil {
			serverErrors <- err
			return
		}

		packet, err = server.ReadPacket()
		if err != nil {
			serverErrors <- err
			return
		}
		if packet.ID != Java1214.LoginAcknowledgedPacketID || len(packet.Data) != 0 {
			serverErrors <- fmt.Errorf("unexpected login acknowledged packet: %#v", packet)
			return
		}
		packet, err = server.ReadPacket()
		if err != nil {
			serverErrors <- err
			return
		}
		if packet.ID != Java1214.ConfigClientInformationPacketID || len(packet.Data) == 0 {
			serverErrors <- fmt.Errorf("unexpected client information packet: %#v", packet)
			return
		}
		registry := NewWriter()
		_ = registry.String("minecraft:test_registry")
		_ = registry.VarInt(1)
		_ = registry.String("minecraft:test_entry")
		_ = registry.Bool(false)
		if err := server.WritePacket(Java1214.ConfigRegistryDataPacketID, registry.Bytes()); err != nil {
			serverErrors <- err
			return
		}
		features := NewWriter()
		_ = features.VarInt(1)
		_ = features.String("minecraft:test_feature")
		if err := server.WritePacket(Java1214.ConfigFeatureFlagsPacketID, features.Bytes()); err != nil {
			serverErrors <- err
			return
		}
		tags := NewWriter()
		_ = tags.VarInt(1)
		_ = tags.String("minecraft:test_registry")
		_ = tags.VarInt(1)
		_ = tags.String("minecraft:test_tag")
		_ = tags.VarInt(1)
		_ = tags.VarInt(3)
		if err := server.WritePacket(Java1214.ConfigTagsPacketID, tags.Bytes()); err != nil {
			serverErrors <- err
			return
		}
		if err := server.WritePacket(Java1214.ConfigResetChatPacketID, nil); err != nil {
			serverErrors <- err
			return
		}

		if err := server.WritePacket(Java1214.ConfigSelectKnownPacksPacketID, []byte{0}); err != nil {
			serverErrors <- err
			return
		}
		packet, err = server.ReadPacket()
		if err != nil {
			serverErrors <- err
			return
		}
		if packet.ID != Java1214.ConfigServerboundSelectKnownPacksID || len(packet.Data) != 1 || packet.Data[0] != 0 {
			serverErrors <- fmt.Errorf("unexpected known packs response: %#v", packet)
			return
		}
		if err := server.WritePacket(Java1214.ConfigFinishPacketID, nil); err != nil {
			serverErrors <- err
			return
		}
		packet, err = server.ReadPacket()
		if err != nil {
			serverErrors <- err
			return
		}
		if packet.ID != Java1214.ConfigServerboundFinishPacketID || len(packet.Data) != 0 {
			serverErrors <- fmt.Errorf("unexpected finish response: %#v", packet)
			return
		}
		serverErrors <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var playerUUID [16]byte
	client, err := DialAndLogin(ctx, listener.Addr().String(), Java1214, "Tester", &playerUUID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.Conn.State() != StatePlay || client.Login.Username != "Tester" {
		t.Fatalf("unexpected negotiated client: state=%d login=%#v", client.Conn.State(), client.Login)
	}
	if len(client.Configuration.Registries) != 1 || client.Configuration.Registries[0].ID != "minecraft:test_registry" {
		t.Fatalf("configuration registries = %#v", client.Configuration.Registries)
	}
	if len(client.Configuration.FeatureFlags) != 1 || client.Configuration.FeatureFlags[0] != "minecraft:test_feature" {
		t.Fatalf("configuration features = %#v", client.Configuration.FeatureFlags)
	}
	if len(client.Configuration.Tags["minecraft:test_registry"]["minecraft:test_tag"]) != 1 || !client.Configuration.ResetChat {
		t.Fatalf("configuration tags/reset = %#v/%t", client.Configuration.Tags, client.Configuration.ResetChat)
	}
	if err := <-serverErrors; err != nil {
		t.Fatal(err)
	}
}
