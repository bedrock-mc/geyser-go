package translate

import (
	"math"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestJavaProjectileStepPotion(t *testing.T) {
	entity := &javaEntityState{
		entityType: "minecraft:splash_potion",
		position:   mgl32.Vec3{10, 20, 30},
		velocity:   mgl32.Vec3{1, 2, 0},
	}
	position, velocity, _, ok := javaProjectileStep(entity)
	if !ok {
		t.Fatal("potion projectile was not simulated")
	}
	if position != (mgl32.Vec3{11, 22, 30}) {
		t.Fatalf("potion position = %v", position)
	}
	wantVelocity := mgl32.Vec3{0.99, 1.93, 0}
	if math.Abs(float64(velocity[0]-wantVelocity[0])) > 0.00001 || math.Abs(float64(velocity[1]-wantVelocity[1])) > 0.00001 || velocity[2] != wantVelocity[2] {
		t.Fatalf("potion velocity = %v, want %v", velocity, wantVelocity)
	}
}

func TestJavaProjectileStepFirework(t *testing.T) {
	entity := &javaEntityState{
		entityType: "minecraft:fireworks_rocket",
		velocity:   mgl32.Vec3{1, 0, 0},
	}
	position, velocity, rotation, ok := javaProjectileStep(entity)
	if !ok {
		t.Fatal("firework projectile was not simulated")
	}
	if position != (mgl32.Vec3{1.15, 0.04, 0}) || velocity != (mgl32.Vec3{1.15, 0.04, 0}) {
		t.Fatalf("firework state = position %v velocity %v", position, velocity)
	}
	if rotation[0] <= 0 || rotation[1] != 90 || rotation[2] != 90 {
		t.Fatalf("firework rotation = %v", rotation)
	}

	entity.fireworkShotAtAngle = true
	position, velocity, _, ok = javaProjectileStep(entity)
	if !ok || position != (mgl32.Vec3{1, 0, 0}) || velocity != (mgl32.Vec3{1, 0, 0}) {
		t.Fatalf("shot-at-angle firework state = position %v velocity %v ok=%t", position, velocity, ok)
	}
}

func TestJavaProjectileStepSkipsAttachedAndGrounded(t *testing.T) {
	attached := &javaEntityState{entityType: "minecraft:fireworks_rocket", fireworkAttachedToEntity: true}
	if _, _, _, ok := javaProjectileStep(attached); ok {
		t.Fatal("attached firework was simulated")
	}
	grounded := &javaEntityState{entityType: "minecraft:arrow", projectileInGround: true}
	if _, _, _, ok := javaProjectileStep(grounded); ok {
		t.Fatal("grounded arrow was simulated")
	}
}

func TestUpdateJavaProjectileMetadata(t *testing.T) {
	entity := &javaEntityState{entityType: "minecraft:arrow"}
	updateJavaProjectileMetadataLocked(entity, []JavaEntityMetadataEntry{{Index: 10, Value: true}})
	if !entity.projectileInGround {
		t.Fatal("arrow in-ground metadata was not retained")
	}
	firework := &javaEntityState{entityType: "minecraft:fireworks_rocket"}
	updateJavaProjectileMetadataLocked(firework, []JavaEntityMetadataEntry{{Index: 10, Value: true}})
	if !firework.fireworkShotAtAngle {
		t.Fatal("firework shot-at-angle metadata was not retained")
	}
}
