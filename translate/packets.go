package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

type JoinGame struct {
	EntityID           int32
	Hardcore           bool
	WorldNames         []string
	ViewDistance       int32
	SimulationDistance int32
	World              SpawnInfo
}

type SpawnInfo struct {
	Dimension      int32
	Name           string
	HashedSeed     int64
	GameMode       int8
	PreviousMode   uint8
	Debug          bool
	Flat           bool
	PortalCooldown int32
	SeaLevel       int32
}

func DecodeJoinGame(data []byte) (JoinGame, error) {
	r := javaprotocol.NewReader(data)
	entityID, err := r.Int32()
	if err != nil {
		return JoinGame{}, err
	}
	hardcore, err := r.Bool()
	if err != nil {
		return JoinGame{}, err
	}
	worldCount, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	if worldCount < 0 || worldCount > maxJavaCollectionSize {
		return JoinGame{}, fmt.Errorf("translate: invalid world name count %d", worldCount)
	}
	worldNames := make([]string, 0, worldCount)
	for i := int32(0); i < worldCount; i++ {
		name, err := r.String()
		if err != nil {
			return JoinGame{}, err
		}
		worldNames = append(worldNames, name)
	}
	maxPlayers, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	_ = maxPlayers
	viewDistance, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	simulationDistance, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	for i := 0; i < 3; i++ {
		if _, err := r.Bool(); err != nil {
			return JoinGame{}, err
		}
	}
	dimension, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	dimensionName, err := r.String()
	if err != nil {
		return JoinGame{}, err
	}
	hashedSeed, err := r.Int64()
	if err != nil {
		return JoinGame{}, err
	}
	gameMode, err := r.Int8()
	if err != nil {
		return JoinGame{}, err
	}
	previousMode, err := r.Uint8()
	if err != nil {
		return JoinGame{}, err
	}
	debug, err := r.Bool()
	if err != nil {
		return JoinGame{}, err
	}
	flat, err := r.Bool()
	if err != nil {
		return JoinGame{}, err
	}
	deathLocation, err := r.Bool()
	if err != nil {
		return JoinGame{}, err
	}
	if deathLocation {
		if _, err := r.String(); err != nil {
			return JoinGame{}, err
		}
		if _, err := r.Int64(); err != nil {
			return JoinGame{}, err
		}
	}
	portalCooldown, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	seaLevel, err := r.VarInt()
	if err != nil {
		return JoinGame{}, err
	}
	if _, err := r.Bool(); err != nil { // enforces secure chat
		return JoinGame{}, err
	}
	if r.Remaining() != 0 {
		return JoinGame{}, fmt.Errorf("translate: Java Login has %d trailing bytes", r.Remaining())
	}
	return JoinGame{
		EntityID: entityID, Hardcore: hardcore, WorldNames: worldNames,
		ViewDistance: viewDistance, SimulationDistance: simulationDistance,
		World: SpawnInfo{
			Dimension: dimension, Name: dimensionName, HashedSeed: hashedSeed,
			GameMode: gameMode, PreviousMode: previousMode, Debug: debug, Flat: flat,
			PortalCooldown: portalCooldown, SeaLevel: seaLevel,
		},
	}, nil
}

func (j JoinGame) GameData(bedrock *minecraft.Conn, items []gtprotocol.ItemEntry) minecraft.GameData {
	worldName := "Java server"
	if len(j.WorldNames) > 0 && j.WorldNames[0] != "" {
		worldName = j.WorldNames[0]
	}
	mode := int32(j.World.GameMode)
	if mode < 0 || mode > 3 {
		mode = 0
	}
	var dimension int32
	switch j.World.Name {
	case "minecraft:the_nether":
		dimension = 1
	case "minecraft:the_end":
		dimension = 2
	}
	position := mgl32.Vec3{0.5, 80, 0.5}
	baseVersion := "1.26.30"
	if bedrock != nil && bedrock.Proto() != nil {
		baseVersion = bedrock.Proto().Ver()
	}
	return minecraft.GameData{
		WorldName: worldName, WorldSeed: j.World.HashedSeed,
		EntityUniqueID: int64(j.EntityID), EntityRuntimeID: uint64(uint32(j.EntityID)),
		PlayerGameMode: mode, PlayerPosition: position,
		Dimension: dimension, WorldSpawn: gtprotocol.BlockPos{0, 80, 0},
		WorldGameMode: mode, Difficulty: 2, ChunkRadius: defaultChunkRadius,
		Items: items, BaseGameVersion: baseVersion, Hardcore: j.Hardcore,
		ServerAuthoritativeInventory: true, PlayerPermissions: 1,
		PlayerMovementSettings: gtprotocol.PlayerMovementSettings{
			ServerAuthoritativeBlockBreaking: true,
		},
	}
}

type PositionUpdate struct {
	TeleportID             int32
	X, Y, Z                float64
	DeltaX, DeltaY, DeltaZ float64
	Yaw, Pitch             float32
	Flags                  uint32
}

type PlayerRotation struct {
	Yaw, Pitch float32
}

func DecodePlayerRotation(data []byte) (PlayerRotation, error) {
	r := javaprotocol.NewReader(data)
	yaw, err := r.Float32()
	if err != nil {
		return PlayerRotation{}, fmt.Errorf("translate: player rotation yaw: %w", err)
	}
	pitch, err := r.Float32()
	if err != nil {
		return PlayerRotation{}, fmt.Errorf("translate: player rotation pitch: %w", err)
	}
	if r.Remaining() != 0 {
		return PlayerRotation{}, fmt.Errorf("translate: player rotation has %d trailing bytes", r.Remaining())
	}
	return PlayerRotation{Yaw: yaw, Pitch: pitch}, nil
}

func DecodePositionUpdate(data []byte) (PositionUpdate, error) {
	r := javaprotocol.NewReader(data)
	teleportID, err := r.VarInt()
	if err != nil {
		return PositionUpdate{}, err
	}
	x, err := r.Float64()
	if err != nil {
		return PositionUpdate{}, err
	}
	y, err := r.Float64()
	if err != nil {
		return PositionUpdate{}, err
	}
	z, err := r.Float64()
	if err != nil {
		return PositionUpdate{}, err
	}
	deltaX, err := r.Float64()
	if err != nil {
		return PositionUpdate{}, err
	}
	deltaY, err := r.Float64()
	if err != nil {
		return PositionUpdate{}, err
	}
	deltaZ, err := r.Float64()
	if err != nil {
		return PositionUpdate{}, err
	}
	yaw, err := r.Float32()
	if err != nil {
		return PositionUpdate{}, err
	}
	pitch, err := r.Float32()
	if err != nil {
		return PositionUpdate{}, err
	}
	flags, err := r.Int32()
	if err != nil {
		return PositionUpdate{}, err
	}
	if r.Remaining() != 0 {
		return PositionUpdate{}, fmt.Errorf("translate: position update has %d trailing bytes", r.Remaining())
	}
	return PositionUpdate{
		TeleportID: teleportID,
		X:          x, Y: y, Z: z,
		DeltaX: deltaX, DeltaY: deltaY, DeltaZ: deltaZ,
		Yaw: yaw, Pitch: pitch, Flags: uint32(flags),
	}, nil
}

func DecodeHealthUpdate(data []byte) (float32, error) {
	r := javaprotocol.NewReader(data)
	health, err := r.Float32()
	if err != nil {
		return 0, err
	}
	if _, err := r.VarInt(); err != nil {
		return 0, err
	}
	if _, err := r.Float32(); err != nil {
		return 0, err
	}
	if r.Remaining() != 0 {
		return 0, fmt.Errorf("translate: health update has %d trailing bytes", r.Remaining())
	}
	return health, nil
}
