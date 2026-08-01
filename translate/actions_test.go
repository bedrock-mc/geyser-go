package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
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
