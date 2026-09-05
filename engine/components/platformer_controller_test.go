package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core"
)

// testVelocity returns a named velocity.
func testVelocity(name string) *Velocity {
	v := &Velocity{}
	v.SetName(name)
	return v
}

// platformerPlayer builds a 32×32 player with the full platformer stack, resting
// flush on a platform placed below it at the given top Y.
func platformerPlayer(topY float64) (*core.Scene, *core.Object, *PlatformerController, *Velocity) {
	scene := core.NewScene("main")
	platform := testObject("platform", 0, topY, testCollider("body", 200, 20, 0))
	player := testObject("player", 0, topY-32,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
		testVelocity("velocity"),
	)
	mustAdd(scene, platform)
	mustAdd(scene, player)

	pc := &PlatformerController{}
	pc.SetName("controller")
	if err := player.AddComponent(pc); err != nil {
		panic(err)
	}
	pc.Initialize()

	return scene, player, pc, core.GetFrom[*Velocity](player)
}

func TestMoverIsGrounded(t *testing.T) {
	scene := core.NewScene("main")
	platform := testObject("platform", 0, 100, testCollider("body", 200, 20, 0))
	player := testObject("player", 0, 68,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
	)
	mustAdd(scene, platform)
	mustAdd(scene, player)

	mover := getMover(player)
	if !mover.IsGrounded() {
		t.Fatalf("player resting flush on the platform should be grounded")
	}

	// Lift the player into the air: no longer grounded.
	player.Transform.Position.Y = 60
	if mover.IsGrounded() {
		t.Fatalf("player 8px above the platform should not be grounded")
	}
}

func TestPlatformerControllerJump(t *testing.T) {
	_, _, pc, velocity := platformerPlayer(100)

	in := &stubInput{justPressed: justPressedKeys(core.KeySpace)}
	ctx := &core.Context{Input: in, Time: &stubTime{delta: 1.0 / 60.0}}
	pc.Update(ctx)

	vx, vy := velocity.Velocity()
	if vx != 0 {
		t.Fatalf("jump should not change vx, got %v", vx)
	}
	if vy >= 0 {
		t.Fatalf("expected an upward jump impulse, got vy=%v", vy)
	}
	if vy != -pc.JumpSpeed {
		t.Fatalf("expected vy=%v (=-jump_speed), got %v", -pc.JumpSpeed, vy)
	}
}

func TestPlatformerControllerNoJumpWhenAirborne(t *testing.T) {
	_, player, pc, velocity := platformerPlayer(100)

	// Lift the player off the ground, then try to jump.
	player.Transform.Position.Y = 60
	in := &stubInput{justPressed: justPressedKeys(core.KeySpace)}
	ctx := &core.Context{Input: in, Time: &stubTime{delta: 1.0 / 60.0}}
	pc.Update(ctx)

	if _, vy := velocity.Velocity(); vy != 0 {
		t.Fatalf("airborne player must not jump, got vy=%v", vy)
	}
}

func TestPlatformerControllerHorizontal(t *testing.T) {
	_, _, pc, velocity := platformerPlayer(100)

	in := &stubInput{held: heldKeys(core.KeyD)}
	ctx := &core.Context{Input: in, Time: &stubTime{delta: 1.0 / 60.0}}
	pc.Update(ctx)

	vx, vy := velocity.Velocity()
	if vx != pc.MoveSpeed {
		t.Fatalf("holding Right should set vx=%v, got %v", pc.MoveSpeed, vx)
	}
	if vy != 0 {
		t.Fatalf("horizontal movement should not change vy, got %v", vy)
	}
}

func TestPlatformerControllerVariableJump(t *testing.T) {
	_, _, pc, velocity := platformerPlayer(100)

	// Jump, then release the key this same frame while still rising.
	in := &stubInput{
		justPressed:  justPressedKeys(core.KeySpace),
		justReleased: heldKeys(core.KeySpace),
	}
	ctx := &core.Context{Input: in, Time: &stubTime{delta: 1.0 / 60.0}}
	pc.Update(ctx)

	_, vy := velocity.Velocity()
	if vy != -pc.JumpSpeed*0.5 {
		t.Fatalf("releasing during ascent should halve vy, got %v", vy)
	}
}
