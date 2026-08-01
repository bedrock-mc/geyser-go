package translate

import (
	"fmt"
	"math"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	javaEntityPositionScale = 4096.0
	javaEntityVelocityScale = 8000.0
)

// JavaBlockChange is the wire shape of ClientboundBlockUpdatePacket. StateID
// is retained as a Java registry ID until the versioned data table is applied.
type JavaBlockChange struct {
	Position gtprotocol.BlockPos
	StateID  int32
}

func DecodeBlockChange(payload []byte) (JavaBlockChange, error) {
	r := javaprotocol.NewReader(payload)
	packed, err := r.Int64()
	if err != nil {
		return JavaBlockChange{}, fmt.Errorf("translate: block change position: %w", err)
	}
	stateID, err := r.VarInt()
	if err != nil {
		return JavaBlockChange{}, fmt.Errorf("translate: block change state: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaBlockChange{}, fmt.Errorf("translate: block change has %d trailing bytes", r.Remaining())
	}
	return JavaBlockChange{Position: decodeJavaPosition(packed), StateID: stateID}, nil
}

func decodeJavaPosition(value int64) gtprotocol.BlockPos {
	return gtprotocol.BlockPos{
		signExtend(uint32(value>>38)&0x03ffffff, 26),
		signExtend(uint32(value)&0x00000fff, 12),
		signExtend(uint32(value>>12)&0x03ffffff, 26),
	}
}

func encodeJavaPosition(position gtprotocol.BlockPos) int64 {
	return (int64(uint64(uint32(position[0])&0x03ffffff)) << 38) |
		(int64(uint64(uint32(position[2])&0x03ffffff)) << 12) |
		int64(uint64(uint32(position[1])&0x00000fff))
}

func signExtend(value uint32, bits uint) int32 {
	shift := 32 - bits
	return int32(value<<shift) >> shift
}

// JavaBlockRuntimeID maps a Java state to a Bedrock runtime ID. An unknown
// registry ID is a semantically odd but well-formed server value; callers get
// air plus false and can keep the session alive while recording the anomaly.
func JavaBlockRuntimeID(stateID int32) (uint32, bool) {
	if stateID < 0 || int64(stateID) >= int64(len(data.Java1214ToBedrock)) {
		return data.Java1214ToBedrock[0], false
	}
	return data.Java1214ToBedrock[stateID], true
}

type JavaUnloadChunk struct {
	X int32
	Z int32
}

func DecodeUnloadChunk(payload []byte) (JavaUnloadChunk, error) {
	r := javaprotocol.NewReader(payload)
	z, err := r.Int32()
	if err != nil {
		return JavaUnloadChunk{}, fmt.Errorf("translate: unload chunk z: %w", err)
	}
	x, err := r.Int32()
	if err != nil {
		return JavaUnloadChunk{}, fmt.Errorf("translate: unload chunk x: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaUnloadChunk{}, fmt.Errorf("translate: unload chunk has %d trailing bytes", r.Remaining())
	}
	return JavaUnloadChunk{X: x, Z: z}, nil
}

type JavaTimeUpdate struct {
	Age         int64
	Time        int64
	TickDayTime bool
}

func DecodeTimeUpdate(payload []byte) (JavaTimeUpdate, error) {
	r := javaprotocol.NewReader(payload)
	age, err := r.Int64()
	if err != nil {
		return JavaTimeUpdate{}, fmt.Errorf("translate: time age: %w", err)
	}
	timeValue, err := r.Int64()
	if err != nil {
		return JavaTimeUpdate{}, fmt.Errorf("translate: time value: %w", err)
	}
	ticking, err := r.Bool()
	if err != nil {
		return JavaTimeUpdate{}, fmt.Errorf("translate: time ticking flag: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaTimeUpdate{}, fmt.Errorf("translate: time update has %d trailing bytes", r.Remaining())
	}
	return JavaTimeUpdate{Age: age, Time: timeValue, TickDayTime: ticking}, nil
}

type JavaSpawnEntity struct {
	EntityID   int32
	UUID       [16]byte
	Type       int32
	Position   mgl32.Vec3
	Pitch      float32
	Yaw        float32
	HeadYaw    float32
	ObjectData int32
	Velocity   mgl32.Vec3
}

func DecodeSpawnEntity(payload []byte) (JavaSpawnEntity, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity ID: %w", err)
	}
	var entityUUID [16]byte
	data, err := r.Bytes(len(entityUUID))
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity UUID: %w", err)
	}
	copy(entityUUID[:], data)
	typeID, err := r.VarInt()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity type: %w", err)
	}
	x, err := r.Float64()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity x: %w", err)
	}
	y, err := r.Float64()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity y: %w", err)
	}
	z, err := r.Float64()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity z: %w", err)
	}
	pitch, err := r.Int8()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity pitch: %w", err)
	}
	yaw, err := r.Int8()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity yaw: %w", err)
	}
	headYaw, err := r.Int8()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity head yaw: %w", err)
	}
	objectData, err := r.VarInt()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity object data: %w", err)
	}
	velocityX, err := r.Int16()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity velocity x: %w", err)
	}
	velocityY, err := r.Int16()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity velocity y: %w", err)
	}
	velocityZ, err := r.Int16()
	if err != nil {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity velocity z: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSpawnEntity{}, fmt.Errorf("translate: spawn entity has %d trailing bytes", r.Remaining())
	}
	return JavaSpawnEntity{
		EntityID: entityID, UUID: entityUUID, Type: typeID,
		Position: mgl32.Vec3{float32(x), float32(y), float32(z)},
		Pitch:    angleByte(pitch), Yaw: angleByte(yaw), HeadYaw: angleByte(headYaw),
		ObjectData: objectData,
		Velocity: mgl32.Vec3{
			float32(float64(velocityX) / javaEntityVelocityScale),
			float32(float64(velocityY) / javaEntityVelocityScale),
			float32(float64(velocityZ) / javaEntityVelocityScale),
		},
	}, nil
}

type JavaEntityTeleport struct {
	EntityID int32
	Position mgl32.Vec3
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func DecodeEntityTeleport(payload []byte) (JavaEntityTeleport, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityTeleport{}, fmt.Errorf("translate: entity teleport ID: %w", err)
	}
	position, err := readJavaPosition(r, "entity teleport")
	if err != nil {
		return JavaEntityTeleport{}, err
	}
	yaw, err := r.Int8()
	if err != nil {
		return JavaEntityTeleport{}, fmt.Errorf("translate: entity teleport yaw: %w", err)
	}
	pitch, err := r.Int8()
	if err != nil {
		return JavaEntityTeleport{}, fmt.Errorf("translate: entity teleport pitch: %w", err)
	}
	onGround, err := r.Bool()
	if err != nil {
		return JavaEntityTeleport{}, fmt.Errorf("translate: entity teleport ground flag: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntityTeleport{}, fmt.Errorf("translate: entity teleport has %d trailing bytes", r.Remaining())
	}
	return JavaEntityTeleport{EntityID: entityID, Position: position, Yaw: angleByte(yaw), Pitch: angleByte(pitch), OnGround: onGround}, nil
}

type JavaEntityRelativeMove struct {
	EntityID    int32
	Delta       mgl32.Vec3
	Yaw         float32
	Pitch       float32
	HasRotation bool
	OnGround    bool
}

func DecodeEntityRelativeMove(payload []byte, hasRotation bool) (JavaEntityRelativeMove, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move ID: %w", err)
	}
	dx, err := r.Int16()
	if err != nil {
		return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move x: %w", err)
	}
	dy, err := r.Int16()
	if err != nil {
		return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move y: %w", err)
	}
	dz, err := r.Int16()
	if err != nil {
		return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move z: %w", err)
	}
	result := JavaEntityRelativeMove{
		EntityID: entityID,
		Delta: mgl32.Vec3{
			float32(float64(dx) / javaEntityPositionScale),
			float32(float64(dy) / javaEntityPositionScale),
			float32(float64(dz) / javaEntityPositionScale),
		},
		HasRotation: hasRotation,
	}
	if hasRotation {
		yaw, readErr := r.Int8()
		if readErr != nil {
			return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move yaw: %w", readErr)
		}
		pitch, readErr := r.Int8()
		if readErr != nil {
			return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move pitch: %w", readErr)
		}
		result.Yaw, result.Pitch = angleByte(yaw), angleByte(pitch)
	}
	onGround, err := r.Bool()
	if err != nil {
		return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move ground flag: %w", err)
	}
	result.OnGround = onGround
	if r.Remaining() != 0 {
		return JavaEntityRelativeMove{}, fmt.Errorf("translate: entity relative move has %d trailing bytes", r.Remaining())
	}
	return result, nil
}

type JavaEntityLook struct {
	EntityID int32
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func DecodeEntityLook(payload []byte) (JavaEntityLook, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityLook{}, fmt.Errorf("translate: entity look ID: %w", err)
	}
	yaw, err := r.Int8()
	if err != nil {
		return JavaEntityLook{}, fmt.Errorf("translate: entity look yaw: %w", err)
	}
	pitch, err := r.Int8()
	if err != nil {
		return JavaEntityLook{}, fmt.Errorf("translate: entity look pitch: %w", err)
	}
	onGround, err := r.Bool()
	if err != nil {
		return JavaEntityLook{}, fmt.Errorf("translate: entity look ground flag: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntityLook{}, fmt.Errorf("translate: entity look has %d trailing bytes", r.Remaining())
	}
	return JavaEntityLook{EntityID: entityID, Yaw: angleByte(yaw), Pitch: angleByte(pitch), OnGround: onGround}, nil
}

type JavaEntityHeadRotation struct {
	EntityID int32
	HeadYaw  float32
}

func DecodeEntityHeadRotation(payload []byte) (JavaEntityHeadRotation, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityHeadRotation{}, fmt.Errorf("translate: entity head rotation ID: %w", err)
	}
	headYaw, err := r.Int8()
	if err != nil {
		return JavaEntityHeadRotation{}, fmt.Errorf("translate: entity head rotation: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntityHeadRotation{}, fmt.Errorf("translate: entity head rotation has %d trailing bytes", r.Remaining())
	}
	return JavaEntityHeadRotation{EntityID: entityID, HeadYaw: angleByte(headYaw)}, nil
}

func DecodeEntityDestroy(payload []byte) ([]int32, error) {
	r := javaprotocol.NewReader(payload)
	count, err := r.VarInt()
	if err != nil {
		return nil, fmt.Errorf("translate: entity destroy count: %w", err)
	}
	if count < 0 || count > maxJavaCollectionSize {
		return nil, fmt.Errorf("translate: invalid entity destroy count %d", count)
	}
	ids := make([]int32, count)
	for i := range ids {
		ids[i], err = r.VarInt()
		if err != nil {
			return nil, fmt.Errorf("translate: entity destroy ID %d: %w", i, err)
		}
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("translate: entity destroy has %d trailing bytes", r.Remaining())
	}
	return ids, nil
}

func readJavaPosition(r *javaprotocol.Reader, prefix string) (mgl32.Vec3, error) {
	x, err := r.Float64()
	if err != nil {
		return mgl32.Vec3{}, fmt.Errorf("translate: %s x: %w", prefix, err)
	}
	y, err := r.Float64()
	if err != nil {
		return mgl32.Vec3{}, fmt.Errorf("translate: %s y: %w", prefix, err)
	}
	z, err := r.Float64()
	if err != nil {
		return mgl32.Vec3{}, fmt.Errorf("translate: %s z: %w", prefix, err)
	}
	// Non-finite coordinates are semantically odd but still valid wire data.
	// Keep decoding them so the session can apply the lenient remote-data
	// policy; the translator decides whether forwarding that value is safe for
	// the Bedrock client.
	return mgl32.Vec3{float32(x), float32(y), float32(z)}, nil
}

func finiteVec3(value mgl32.Vec3) bool {
	for _, component := range value {
		if math.IsNaN(float64(component)) || math.IsInf(float64(component), 0) {
			return false
		}
	}
	return true
}

func angleByte(value int8) float32 {
	return float32(float64(value) * 360.0 / 256.0)
}
