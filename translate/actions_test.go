package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestEncodeJavaBlockDig(t *testing.T) {
	wantPosition := gtprotocol.BlockPos{-4, 63, 11}
	payload, err := encodeJavaBlockDig(2, wantPosition, 5, 17)
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	status, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	packed, err := r.Int64()
	if err != nil {
		t.Fatal(err)
	}
	face, err := r.Int8()
	if err != nil {
		t.Fatal(err)
	}
	sequence, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if status != 2 || decodeJavaPosition(packed) != wantPosition || face != 5 || sequence != 17 || r.Remaining() != 0 {
		t.Fatalf("Java block dig = status=%d position=%v face=%d sequence=%d remaining=%d", status, decodeJavaPosition(packed), face, sequence, r.Remaining())
	}
}

func TestEncodeJavaBlockPlaceAndUseItem(t *testing.T) {
	transaction := gtprotocol.UseItemTransactionData{
		BlockPosition:   gtprotocol.BlockPos{2, 64, -7},
		BlockFace:       1,
		ClickedPosition: [3]float32{0.25, 0.5, 0.75},
	}
	payload, err := encodeJavaBlockPlace(transaction, 8)
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	hand, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	position, err := r.Int64()
	if err != nil {
		t.Fatal(err)
	}
	direction, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	x, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	y, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	z, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	inside, err := r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	border, err := r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	sequence, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if hand != 0 || decodeJavaPosition(position) != transaction.BlockPosition || direction != 1 || x != 0.25 || y != 0.5 || z != 0.75 || inside || border || sequence != 8 || r.Remaining() != 0 {
		t.Fatalf("Java block place fields = hand=%d position=%v direction=%d cursor=(%v,%v,%v) inside=%v border=%v sequence=%d remaining=%d", hand, decodeJavaPosition(position), direction, x, y, z, inside, border, sequence, r.Remaining())
	}

	payload, err = encodeJavaUseItem(90, -10, 9)
	if err != nil {
		t.Fatal(err)
	}
	r = javaprotocol.NewReader(payload)
	hand, err = r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	sequence, err = r.VarInt()
	if err != nil {
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
	if hand != 0 || sequence != 9 || yaw != 90 || pitch != -10 || r.Remaining() != 0 {
		t.Fatalf("Java use item fields = hand=%d sequence=%d yaw=%v pitch=%v remaining=%d", hand, sequence, yaw, pitch, r.Remaining())
	}
	if _, err := encodeJavaUseItem(float32(math.NaN()), 0, 1); err == nil {
		t.Fatal("non-finite use-item rotation unexpectedly encoded")
	}
}

func TestEncodeJavaHeldArmEntityAndPlayerActions(t *testing.T) {
	payload, err := encodeJavaHeldItemSlot(7)
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	slot, err := r.Int16()
	if err != nil {
		t.Fatal(err)
	}
	if slot != 7 || r.Remaining() != 0 {
		t.Fatalf("held slot = %d remaining=%d", slot, r.Remaining())
	}

	payload, err = encodeJavaArmAnimation(0)
	if err != nil {
		t.Fatal(err)
	}
	r = javaprotocol.NewReader(payload)
	hand, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if hand != 0 || r.Remaining() != 0 {
		t.Fatalf("arm animation hand=%d remaining=%d", hand, r.Remaining())
	}

	payload, err = encodeJavaUseEntity(123, 0, gtprotocol.Optional[mgl32.Vec3]{})
	if err != nil {
		t.Fatal(err)
	}
	r = javaprotocol.NewReader(payload)
	target, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	mouse, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	hand, err = r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	sneaking, err := r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	if target != 123 || mouse != 0 || hand != 0 || sneaking || r.Remaining() != 0 {
		t.Fatalf("entity interact = target=%d mouse=%d hand=%d sneaking=%v remaining=%d", target, mouse, hand, sneaking, r.Remaining())
	}

	hit := gtprotocol.Option(mgl32.Vec3{0.1, 0.2, 0.3})
	payload, err = encodeJavaUseEntity(-4, 2, hit)
	if err != nil {
		t.Fatal(err)
	}
	r = javaprotocol.NewReader(payload)
	target, err = r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	mouse, err = r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	x, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	y, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	z, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	hand, err = r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	sneaking, err = r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	if target != -4 || mouse != 2 || x != 0.1 || y != 0.2 || z != 0.3 || hand != 0 || sneaking || r.Remaining() != 0 {
		t.Fatalf("entity interact-at = target=%d mouse=%d hit=(%v,%v,%v) hand=%d sneaking=%v remaining=%d", target, mouse, x, y, z, hand, sneaking, r.Remaining())
	}
	if _, err := encodeJavaUseEntity(1, 2, gtprotocol.Optional[mgl32.Vec3]{}); err == nil {
		t.Fatal("entity interact-at without a position unexpectedly encoded")
	}

	payload, err = encodeJavaEntityAction(-4, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	r = javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	javaAction, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	jumpBoost, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if entityID != -4 || javaAction != 3 || jumpBoost != 0 || r.Remaining() != 0 {
		t.Fatalf("entity action = entity=%d action=%d jump=%d remaining=%d", entityID, javaAction, jumpBoost, r.Remaining())
	}
}
