package translate

import (
	"fmt"
	"math"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
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

const (
	javaGameEventNoRespawnBlock = iota
	javaGameEventStartRaining
	javaGameEventStopRaining
	javaGameEventChangeGameMode
	javaGameEventWinGame
	javaGameEventDemoEvent
	javaGameEventArrowHitPlayer
	javaGameEventRainLevelChange
	javaGameEventThunderLevelChange
	javaGameEventPufferFishSting
	javaGameEventGuardianElderEffect
	javaGameEventImmediateRespawn
)

// JavaDifficulty is the wire shape of ClientboundChangeDifficultyPacket.
type JavaDifficulty struct {
	Difficulty uint8
	Locked     bool
}

func DecodeJavaDifficulty(data []byte) (JavaDifficulty, error) {
	r := javaprotocol.NewReader(data)
	difficulty, err := r.Byte()
	if err != nil {
		return JavaDifficulty{}, fmt.Errorf("translate: difficulty: %w", err)
	}
	locked, err := r.Bool()
	if err != nil {
		return JavaDifficulty{}, fmt.Errorf("translate: difficulty lock: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaDifficulty{}, fmt.Errorf("translate: difficulty has %d trailing bytes", r.Remaining())
	}
	return JavaDifficulty{Difficulty: difficulty, Locked: locked}, nil
}

// JavaGameStateChange is the wire shape of ClientboundGameEventPacket. The
// meaning of Value is selected by Reason; keeping it as a float preserves the
// protocol shape while allowing the translator to apply semantic bounds.
type JavaGameStateChange struct {
	Reason byte
	Value  float32
}

func DecodeJavaGameStateChange(data []byte) (JavaGameStateChange, error) {
	r := javaprotocol.NewReader(data)
	reason, err := r.Byte()
	if err != nil {
		return JavaGameStateChange{}, fmt.Errorf("translate: game state reason: %w", err)
	}
	value, err := r.Float32()
	if err != nil {
		return JavaGameStateChange{}, fmt.Errorf("translate: game state value: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaGameStateChange{}, fmt.Errorf("translate: game state has %d trailing bytes", r.Remaining())
	}
	return JavaGameStateChange{Reason: reason, Value: value}, nil
}

func DecodeJavaClearTitles(data []byte) (bool, error) {
	r := javaprotocol.NewReader(data)
	reset, err := r.Bool()
	if err != nil {
		return false, fmt.Errorf("translate: clear titles: %w", err)
	}
	if r.Remaining() != 0 {
		return false, fmt.Errorf("translate: clear titles has %d trailing bytes", r.Remaining())
	}
	return reset, nil
}

func DecodeJavaTitleText(data []byte) (any, error) {
	r := javaprotocol.NewReader(data)
	text, err := decodeJavaNBTValue(r)
	if err != nil {
		return nil, fmt.Errorf("translate: title text: %w", err)
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("translate: title text has %d trailing bytes", r.Remaining())
	}
	return text, nil
}

type JavaTitleTimes struct {
	FadeIn  int32
	Stay    int32
	FadeOut int32
}

func DecodeJavaTitleTimes(data []byte) (JavaTitleTimes, error) {
	r := javaprotocol.NewReader(data)
	fadeIn, err := r.Int32()
	if err != nil {
		return JavaTitleTimes{}, fmt.Errorf("translate: title fade-in: %w", err)
	}
	stay, err := r.Int32()
	if err != nil {
		return JavaTitleTimes{}, fmt.Errorf("translate: title stay: %w", err)
	}
	fadeOut, err := r.Int32()
	if err != nil {
		return JavaTitleTimes{}, fmt.Errorf("translate: title fade-out: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaTitleTimes{}, fmt.Errorf("translate: title times has %d trailing bytes", r.Remaining())
	}
	return JavaTitleTimes{FadeIn: fadeIn, Stay: stay, FadeOut: fadeOut}, nil
}

type JavaWorldEvent struct {
	EffectID int32
	Position gtprotocol.BlockPos
	Data     int32
	Global   bool
}

func DecodeJavaWorldEvent(data []byte) (JavaWorldEvent, error) {
	r := javaprotocol.NewReader(data)
	effectID, err := r.Int32()
	if err != nil {
		return JavaWorldEvent{}, fmt.Errorf("translate: world event effect: %w", err)
	}
	packed, err := r.Int64()
	if err != nil {
		return JavaWorldEvent{}, fmt.Errorf("translate: world event position: %w", err)
	}
	eventData, err := r.Int32()
	if err != nil {
		return JavaWorldEvent{}, fmt.Errorf("translate: world event data: %w", err)
	}
	global, err := r.Bool()
	if err != nil {
		return JavaWorldEvent{}, fmt.Errorf("translate: world event global flag: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaWorldEvent{}, fmt.Errorf("translate: world event has %d trailing bytes", r.Remaining())
	}
	return JavaWorldEvent{EffectID: effectID, Position: decodeJavaPosition(packed), Data: eventData, Global: global}, nil
}

type JavaSpawnPosition struct {
	Position gtprotocol.BlockPos
	Angle    float32
}

func DecodeJavaSpawnPosition(data []byte) (JavaSpawnPosition, error) {
	r := javaprotocol.NewReader(data)
	packed, err := r.Int64()
	if err != nil {
		return JavaSpawnPosition{}, fmt.Errorf("translate: spawn position: %w", err)
	}
	angle, err := r.Float32()
	if err != nil {
		return JavaSpawnPosition{}, fmt.Errorf("translate: spawn angle: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSpawnPosition{}, fmt.Errorf("translate: spawn position has %d trailing bytes", r.Remaining())
	}
	return JavaSpawnPosition{Position: decodeJavaPosition(packed), Angle: angle}, nil
}

func (b *Basic) translateJavaDifficulty(bedrock *minecraft.Conn, data []byte) error {
	difficulty, err := DecodeJavaDifficulty(data)
	if err != nil {
		return err
	}
	value := uint32(difficulty.Difficulty)
	if value > 3 {
		b.logSemanticAnomaly("skipping unknown Java difficulty", "difficulty", value)
		return nil
	}
	// Bedrock has no peaceful client difficulty equivalent. Geyser exposes
	// peaceful worlds as easy so food and client-side UI remain usable while
	// the Java server remains authoritative.
	if value == 0 {
		value = 1
	}
	return bedrock.WritePacket(&packet.SetDifficulty{Difficulty: value})
}

func (b *Basic) translateJavaGameStateChange(bedrock *minecraft.Conn, data []byte) error {
	change, err := DecodeJavaGameStateChange(data)
	if err != nil {
		return err
	}
	if !finiteFloat32(change.Value) {
		b.logSemanticAnomaly("skipping non-finite Java game-state value", "reason", change.Reason)
		return nil
	}
	switch change.Reason {
	case javaGameEventStartRaining:
		return b.writeLevelEvent(bedrock, packet.LevelEventStartRaining, 0)
	case javaGameEventStopRaining:
		return b.writeLevelEvent(bedrock, packet.LevelEventStopRaining, 0)
	case javaGameEventChangeGameMode:
		mode := int32(math.Round(float64(change.Value)))
		if mode < 0 || mode > 3 {
			b.logSemanticAnomaly("skipping unknown Java game mode", "mode", mode)
			return nil
		}
		bedrockMode := mode
		if mode == 3 {
			bedrockMode = packet.GameTypeSpectator
		}
		b.mu.Lock()
		b.gameData.PlayerGameMode = mode
		b.gameData.WorldGameMode = mode
		b.mu.Unlock()
		return bedrock.WritePacket(&packet.SetPlayerGameType{GameType: bedrockMode})
	case javaGameEventWinGame:
		status := int32(packet.ShowCreditsStatusStart)
		if change.Value != 0 {
			status = packet.ShowCreditsStatusEnd
		}
		return bedrock.WritePacket(&packet.ShowCredits{
			PlayerRuntimeID: b.gameData.EntityRuntimeID,
			StatusType:      status,
		})
	case javaGameEventArrowHitPlayer:
		return b.writePlayerSound(bedrock, "random.orb", 0.5, 0.5)
	case javaGameEventGuardianElderEffect:
		return bedrock.WritePacket(&packet.ActorEvent{
			EntityRuntimeID: b.gameData.EntityRuntimeID,
			EventType:       packet.ActorEventGuardianMiningFatigue,
		})
	case javaGameEventImmediateRespawn:
		return bedrock.WritePacket(&packet.GameRulesChanged{GameRules: []gtprotocol.GameRule{{
			Name:  "doimmediaterespawn",
			Value: change.Value != 0,
		}}})
	case javaGameEventRainLevelChange, javaGameEventThunderLevelChange:
		b.logSemanticAnomaly("Java weather strength has no direct Bedrock packet in this slice", "reason", change.Reason, "value", change.Value)
		return nil
	default:
		b.logSemanticAnomaly("skipping unsupported Java game-state reason", "reason", change.Reason)
		return nil
	}
}

func (b *Basic) writeLevelEvent(bedrock *minecraft.Conn, eventType, data int32) error {
	b.mu.Lock()
	position := mgl32.Vec3{float32(b.position.x), float32(b.position.y), float32(b.position.z)}
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.LevelEvent{EventType: eventType, Position: position, EventData: data})
}

func (b *Basic) writePlayerSound(bedrock *minecraft.Conn, name string, volume, pitch float32) error {
	b.mu.Lock()
	position := mgl32.Vec3{float32(b.position.x), float32(b.position.y), float32(b.position.z)}
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.PlaySound{SoundName: name, Position: position, Volume: volume, Pitch: pitch})
}

func (b *Basic) translateJavaClearTitles(bedrock *minecraft.Conn, data []byte) error {
	reset, err := DecodeJavaClearTitles(data)
	if err != nil {
		return err
	}
	action := int32(packet.TitleActionClear)
	if reset {
		action = packet.TitleActionReset
	}
	return bedrock.WritePacket(&packet.SetTitle{ActionType: action})
}

func (b *Basic) translateJavaTitleText(bedrock *minecraft.Conn, data []byte, action int32) error {
	value, err := DecodeJavaTitleText(data)
	if err != nil {
		return err
	}
	text := JavaTextComponentText(value)
	if text == "" {
		text = " "
	}
	return bedrock.WritePacket(&packet.SetTitle{ActionType: action, Text: text})
}

func (b *Basic) translateJavaTitleTimes(bedrock *minecraft.Conn, data []byte) error {
	times, err := DecodeJavaTitleTimes(data)
	if err != nil {
		return err
	}
	if times.FadeIn < 0 || times.Stay < 0 || times.FadeOut < 0 {
		b.logSemanticAnomaly("skipping negative Java title duration")
		return nil
	}
	return bedrock.WritePacket(&packet.SetTitle{
		ActionType:      packet.TitleActionSetDurations,
		FadeInDuration:  times.FadeIn,
		RemainDuration:  times.Stay,
		FadeOutDuration: times.FadeOut,
	})
}

func (b *Basic) translateJavaSpawnPosition(bedrock *minecraft.Conn, data []byte) error {
	spawn, err := DecodeJavaSpawnPosition(data)
	if err != nil {
		return err
	}
	if !finiteFloat32(spawn.Angle) {
		b.logSemanticAnomaly("skipping Java spawn position with non-finite angle")
		return nil
	}
	b.mu.Lock()
	dimension := b.gameData.Dimension
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.SetSpawnPosition{
		SpawnType:     packet.SpawnTypeWorld,
		Position:      spawn.Position,
		Dimension:     dimension,
		SpawnPosition: spawn.Position,
	})
}
