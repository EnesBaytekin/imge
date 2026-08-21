package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core"
)

// testTrigger returns a named sensor of the given size.
func testTrigger(name string, w, h float64) *Trigger {
	t := &Trigger{Width: w, Height: h}
	t.SetName(name)
	return t
}

// triggerRecorder records trigger_enter/trigger_exit events so tests can assert
// the trigger's overlap tracking without reading the unexported overlaps map.
type triggerRecorder struct {
	core.BaseComponent
	enters []*core.Object
	exits  []*core.Object
}

func (r *triggerRecorder) Initialize() {
	r.On("trigger_enter", func(data any) {
		if obj, ok := data.(*core.Object); ok {
			r.enters = append(r.enters, obj)
		}
	})
	r.On("trigger_exit", func(data any) {
		if obj, ok := data.(*core.Object); ok {
			r.exits = append(r.exits, obj)
		}
	})
}

func TestTriggerOverlapEnterExit(t *testing.T) {
	scene := core.NewScene("main")

	recorder := &triggerRecorder{}
	recorder.SetName("recorder")

	sensor := testObject("sensor", 0, 0, testTrigger("hitbox", 32, 32), recorder)
	player := testObject("player", 0, 0, testCollider("body", 32, 32, 0))
	mustAdd(scene, sensor)
	mustAdd(scene, player)

	trigger := core.GetFrom[*Trigger](sensor)

	// First update: trigger detects the player and emits trigger_enter (delivered
	// at the end of the same frame).
	scene.Update(&core.Context{})

	if overlaps := trigger.GetOverlaps(); len(overlaps) != 1 || overlaps[0] != player {
		t.Fatalf("expected player in overlaps, got %v", overlaps)
	}
	if len(recorder.enters) != 1 || recorder.enters[0] != player {
		t.Fatalf("expected one trigger_enter for player, got %v", recorder.enters)
	}

	// Move the player away: trigger emits trigger_exit.
	player.SetPosition(100, 0)
	scene.Update(&core.Context{})

	if overlaps := trigger.GetOverlaps(); len(overlaps) != 0 {
		t.Fatalf("expected no overlaps after moving away, got %v", overlaps)
	}
	if len(recorder.exits) != 1 || recorder.exits[0] != player {
		t.Fatalf("expected one trigger_exit for player, got %v", recorder.exits)
	}
}

func TestTriggerDoesNotBlockMovement(t *testing.T) {
	scene := core.NewScene("main")

	sensor := testObject("sensor", 0, 0, testTrigger("hitbox", 32, 32))
	player := testObject("player", 0, 0,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
	)
	mustAdd(scene, sensor)
	mustAdd(scene, player)

	// A trigger is a sensor: it detects overlap but never participates in movement
	// resolution, so the mover should pass straight through it.
	res := getMover(player).Move(10, 0)
	if !res.X {
		t.Fatalf("trigger should not block movement")
	}
	if player.Transform.Position.X != 10 {
		t.Fatalf("expected player at x=10, got %v", player.Transform.Position.X)
	}
}

func TestTriggerSelfExclusion(t *testing.T) {
	scene := core.NewScene("main")

	recorder := &triggerRecorder{}
	recorder.SetName("recorder")

	// An object carrying both a collider and a trigger: the trigger must not detect
	// its own collider.
	obj := core.NewObject("self")
	if err := obj.AddComponent(testCollider("body", 32, 32, 0)); err != nil {
		t.Fatal(err)
	}
	if err := obj.AddComponent(testTrigger("hitbox", 32, 32)); err != nil {
		t.Fatal(err)
	}
	if err := obj.AddComponent(recorder); err != nil {
		t.Fatal(err)
	}
	obj.SetPosition(0, 0)
	mustAdd(scene, obj)

	trigger := core.GetFrom[*Trigger](obj)

	scene.Update(&core.Context{})

	if overlaps := trigger.GetOverlaps(); len(overlaps) != 0 {
		t.Fatalf("trigger should not detect its own collider, got %v", overlaps)
	}
	if len(recorder.enters) != 0 {
		t.Fatalf("expected no trigger_enter, got %v", recorder.enters)
	}
}

func TestDamageTargetsNamedTrigger(t *testing.T) {
	scene := core.NewScene("main")

	hitbox := testTrigger("hitbox", 32, 32)
	other := testTrigger("other", 32, 32)
	damage := &Damage{Amount: 1, TargetTags: []string{"player"}, Trigger: "hitbox"}
	damage.SetName("damage")

	hazard := core.NewObject("hazard")
	if err := hazard.AddComponent(hitbox); err != nil {
		t.Fatal(err)
	}
	if err := hazard.AddComponent(other); err != nil {
		t.Fatal(err)
	}
	if err := hazard.AddComponent(damage); err != nil {
		t.Fatal(err)
	}
	hazard.SetPosition(0, 0)
	mustAdd(scene, hazard)

	health := &Health{Max: 100}
	health.SetName("health")

	player := core.NewObject("player")
	player.AddTag("player")
	if err := player.AddComponent(testCollider("body", 32, 32, 0)); err != nil {
		t.Fatal(err)
	}
	if err := player.AddComponent(health); err != nil {
		t.Fatal(err)
	}
	player.SetPosition(0, 0)
	mustAdd(scene, player)

	// Manual lifecycle in a deterministic order. Scene.Update interleaves
	// Initialize and Update per object over a map, so the player's Health might not
	// be initialized before the hazard's Damage reads it; drive it by hand instead.
	ctx := &core.Context{}
	hitbox.Initialize()
	other.Initialize()
	damage.Initialize()
	health.Initialize()

	hitbox.Update(ctx) // populate the hitbox's overlaps with the player
	damage.Update(ctx) // read the named trigger and apply damage

	if got := health.Current(); got != 99 {
		t.Fatalf("expected 99 hp after one damage tick, got %d", got)
	}
}
