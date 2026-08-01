package translate

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type javaProjectileUpdate struct {
	runtimeID uint64
	position  mgl32.Vec3
	rotation  mgl32.Vec3
	old       mgl32.Vec3
}

func javaSimulatedProjectileEntity(entityType string) bool {
	switch entityType {
	case "minecraft:arrow", "minecraft:spectral_arrow", "minecraft:trident",
		"minecraft:egg", "minecraft:ender_pearl", "minecraft:experience_bottle",
		"minecraft:xp_bottle", "minecraft:lingering_potion", "minecraft:snowball",
		"minecraft:splash_potion", "minecraft:potion", "minecraft:fireworks_rocket",
		"minecraft:firework_rocket", "minecraft:fireball", "minecraft:small_fireball",
		"minecraft:dragon_fireball", "minecraft:shulker_bullet", "minecraft:llama_spit",
		"minecraft:fishing_hook", "minecraft:fishing_bobber", "minecraft:item":
		return true
	default:
		return false
	}
}

func javaProjectileGravity(entityType string) float32 {
	switch entityType {
	case "minecraft:lingering_potion", "minecraft:splash_potion", "minecraft:potion":
		return 0.05
	case "minecraft:experience_bottle", "minecraft:xp_bottle":
		return 0.07
	case "minecraft:egg", "minecraft:ender_pearl", "minecraft:snowball":
		return 0.03
	case "minecraft:llama_spit":
		return 0.06
	case "minecraft:fireball", "minecraft:small_fireball", "minecraft:dragon_fireball", "minecraft:shulker_bullet":
		return 0
	case "minecraft:arrow", "minecraft:spectral_arrow", "minecraft:trident":
		return 0.05
	default:
		return 0
	}
}

func javaProjectileDrag(entityType string) float32 {
	switch entityType {
	case "minecraft:lingering_potion", "minecraft:splash_potion", "minecraft:potion",
		"minecraft:experience_bottle", "minecraft:xp_bottle", "minecraft:egg",
		"minecraft:ender_pearl", "minecraft:snowball", "minecraft:llama_spit":
		return 0.99
	case "minecraft:fireball", "minecraft:small_fireball", "minecraft:dragon_fireball":
		return 0.95
	case "minecraft:shulker_bullet":
		return 1
	case "minecraft:arrow", "minecraft:spectral_arrow", "minecraft:trident":
		return 0.99
	case "minecraft:item":
		return 0.98
	default:
		return 1
	}
}

// javaProjectileStep mirrors Geyser's client-side projectile re-simulation.
// Java clients predict these entities locally, while Bedrock needs explicit
// positions each tick. Collision and water drag require a world manager and
// are intentionally kept out of this pure step; the server's authoritative
// movement packets still correct the simulated state.
func javaProjectileStep(entity *javaEntityState) (position, velocity, rotation mgl32.Vec3, ok bool) {
	if entity == nil || !javaSimulatedProjectileEntity(entity.entityType) || entity.projectileInGround || entity.fireworkAttachedToEntity {
		return mgl32.Vec3{}, mgl32.Vec3{}, mgl32.Vec3{}, false
	}
	position = entity.position
	velocity = entity.velocity
	rotation = entity.rotation
	if entity.entityType == "minecraft:fireworks_rocket" || entity.entityType == "minecraft:firework_rocket" {
		if !entity.fireworkShotAtAngle {
			velocity[0] *= 1.15
			velocity[2] *= 1.15
			velocity[1] += 0.04
		}
		position = position.Add(velocity)
		horizontal := float32(math.Hypot(float64(velocity[0]), float64(velocity[2])))
		rotation[0] = float32(math.Atan2(float64(velocity[1]), float64(horizontal)) * 180 / math.Pi)
		rotation[1] = float32(math.Atan2(float64(velocity[0]), float64(velocity[2])*1) * 180 / math.Pi)
		rotation[2] = rotation[1]
	} else {
		position = position.Add(velocity)
		drag := javaProjectileDrag(entity.entityType)
		velocity[0] *= drag
		gravity := float32(0)
		if !entity.projectileNoGravity {
			gravity = javaProjectileGravity(entity.entityType)
		}
		velocity[1] = velocity[1]*drag - gravity
		velocity[2] *= drag
	}
	if !finiteVec3(position) || !finiteVec3(velocity) || !finiteVec3(rotation) {
		return mgl32.Vec3{}, mgl32.Vec3{}, mgl32.Vec3{}, false
	}
	return position, velocity, rotation, true
}

func updateJavaProjectileMetadataLocked(entity *javaEntityState, entries []JavaEntityMetadataEntry) {
	if entity == nil {
		return
	}
	for _, entry := range entries {
		if entry.Index == 5 {
			if value, ok := entry.Value.(bool); ok {
				entity.projectileNoGravity = value
			}
		}
		if entry.Index == 10 {
			if value, ok := entry.Value.(bool); ok {
				switch entity.entityType {
				case "minecraft:arrow", "minecraft:spectral_arrow", "minecraft:trident":
					entity.projectileInGround = value
				case "minecraft:fireworks_rocket", "minecraft:firework_rocket":
					entity.fireworkShotAtAngle = value
				}
			}
		}
	}
}

func (b *Basic) pumpProjectiles(ctx context.Context, bedrock *minecraft.Conn) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := b.tickProjectiles(bedrock); err != nil {
				return err
			}
		}
	}
}

func (b *Basic) tickProjectiles(bedrock *minecraft.Conn) error {
	b.mu.Lock()
	updates := make([]javaProjectileUpdate, 0)
	for _, entity := range b.entities {
		position, velocity, rotation, ok := javaProjectileStep(entity)
		if !ok {
			continue
		}
		old := entity.position
		entity.position = position
		entity.velocity = velocity
		entity.rotation = rotation
		updates = append(updates, javaProjectileUpdate{runtimeID: entity.runtimeID, position: position, rotation: rotation, old: old})
	}
	b.mu.Unlock()

	for _, update := range updates {
		flags := uint16(0)
		if update.position[0] != update.old[0] {
			flags |= packet.MoveActorDeltaFlagHasX
		}
		if update.position[1] != update.old[1] {
			flags |= packet.MoveActorDeltaFlagHasY
		}
		if update.position[2] != update.old[2] {
			flags |= packet.MoveActorDeltaFlagHasZ
		}
		flags |= packet.MoveActorDeltaFlagHasRotX | packet.MoveActorDeltaFlagHasRotY | packet.MoveActorDeltaFlagHasRotZ
		if flags == 0 {
			continue
		}
		if err := bedrock.WritePacket(&packet.MoveActorDelta{
			EntityRuntimeID: update.runtimeID,
			Flags:           flags,
			Position:        update.position,
			Rotation:        update.rotation,
		}); err != nil {
			return fmt.Errorf("translate: send simulated projectile movement: %w", err)
		}
	}
	return nil
}
