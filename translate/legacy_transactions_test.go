package translate

import (
	"net"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestTranslateLegacyEntityInteractAtOrdersHeldSlotAndJavaPackets(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	javaConn := javaprotocol.NewConn(left, javaprotocol.ConnConfig{})
	java := &javaprotocol.Client{Conn: javaConn, Profile: javaprotocol.Java1214}
	b := NewBasic(javaprotocol.Java1214, nil)
	b.sneaking = true

	transaction := &gtprotocol.UseItemOnEntityTransactionData{
		TargetEntityRuntimeID: 123,
		ActionType:            gtprotocol.UseItemOnEntityActionInteract,
		HotBarSlot:            3,
		ClickedPosition:       mgl32.Vec3{0.25, 0.5, 0.75},
	}
	errCh := make(chan error, 1)
	go func() { errCh <- b.translateLegacyUseEntity(java, transaction) }()

	reader := javaprotocol.NewConn(right, javaprotocol.ConnConfig{})
	packet, err := reader.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if packet.ID != javaprotocol.Java1214.PlayServerboundHeldItemSlotID {
		t.Fatalf("first packet ID=%#x, want held slot %#x", packet.ID, javaprotocol.Java1214.PlayServerboundHeldItemSlotID)
	}
	slotReader := javaprotocol.NewReader(packet.Data)
	slot, err := slotReader.Int16()
	if err != nil || slot != 3 || slotReader.Remaining() != 0 {
		t.Fatalf("held slot=%d err=%v remaining=%d", slot, err, slotReader.Remaining())
	}

	packet, err = reader.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if packet.ID != javaprotocol.Java1214.PlayServerboundUseEntityID {
		t.Fatalf("second packet ID=%#x, want use entity %#x", packet.ID, javaprotocol.Java1214.PlayServerboundUseEntityID)
	}
	entityReader := javaprotocol.NewReader(packet.Data)
	target, err := entityReader.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	action, err := entityReader.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	x, err := entityReader.Float32()
	if err != nil {
		t.Fatal(err)
	}
	y, err := entityReader.Float32()
	if err != nil {
		t.Fatal(err)
	}
	z, err := entityReader.Float32()
	if err != nil {
		t.Fatal(err)
	}
	hand, err := entityReader.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	sneaking, err := entityReader.Bool()
	if err != nil {
		t.Fatal(err)
	}
	if target != 123 || action != 2 || x != 0.25 || y != 0.5 || z != 0.75 || hand != 0 || !sneaking || entityReader.Remaining() != 0 {
		t.Fatalf("use entity = target=%d action=%d hit=(%v,%v,%v) hand=%d sneaking=%t remaining=%d", target, action, x, y, z, hand, sneaking, entityReader.Remaining())
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestTranslateLegacyEntityAttackEmitsJavaSwing(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	java := &javaprotocol.Client{Conn: javaprotocol.NewConn(left, javaprotocol.ConnConfig{}), Profile: javaprotocol.Java1214}
	b := NewBasic(javaprotocol.Java1214, nil)
	transaction := &gtprotocol.UseItemOnEntityTransactionData{
		TargetEntityRuntimeID: 456,
		ActionType:            gtprotocol.UseItemOnEntityActionAttack,
		HotBarSlot:            0,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- b.translateLegacyUseEntity(java, transaction) }()
	reader := javaprotocol.NewConn(right, javaprotocol.ConnConfig{})
	for i, wantID := range []int32{javaprotocol.Java1214.PlayServerboundUseEntityID, javaprotocol.Java1214.PlayServerboundArmAnimationID} {
		packet, err := reader.ReadPacket()
		if err != nil {
			t.Fatal(err)
		}
		if packet.ID != wantID {
			t.Fatalf("packet %d ID=%#x, want %#x", i, packet.ID, wantID)
		}
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestTranslateLegacyUseItemClickAirUsesCurrentJavaRotation(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	java := &javaprotocol.Client{Conn: javaprotocol.NewConn(left, javaprotocol.ConnConfig{}), Profile: javaprotocol.Java1214}
	b := NewBasic(javaprotocol.Java1214, nil)
	b.position.yaw = 70
	b.position.pitch = -15
	errCh := make(chan error, 1)
	go func() {
		errCh <- b.translateLegacyUseItem(java, &gtprotocol.UseItemTransactionData{
			ActionType: gtprotocol.UseItemActionClickAir,
			HotBarSlot: 0,
		})
	}()
	reader := javaprotocol.NewConn(right, javaprotocol.ConnConfig{})
	packet, err := reader.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if packet.ID != javaprotocol.Java1214.PlayServerboundUseItemID {
		t.Fatalf("packet ID=%#x, want %#x", packet.ID, javaprotocol.Java1214.PlayServerboundUseItemID)
	}
	r := javaprotocol.NewReader(packet.Data)
	if hand, _ := r.VarInt(); hand != 0 {
		t.Fatalf("hand=%d, want main hand", hand)
	}
	if _, err := r.VarInt(); err != nil {
		t.Fatal(err)
	}
	yaw, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	pitch, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	if yaw != 70 || pitch != -15 || r.Remaining() != 0 {
		t.Fatalf("rotation=(%v,%v) remaining=%d", yaw, pitch, r.Remaining())
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
