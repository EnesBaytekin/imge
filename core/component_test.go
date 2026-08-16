package core

import "testing"

// testComponent records lifecycle calls so tests can assert ordering/count.
type testComponent struct {
	BaseComponent
	initialized int
	enabled     int
	updated     int
	gotEvents   []any
}

func (c *testComponent) Initialize()      { c.initialized++ }
func (c *testComponent) OnEnable()        { c.enabled++ }
func (c *testComponent) OnDisable()       {}
func (c *testComponent) Update(ctx *Context) { c.updated++ }

func TestComponentLifecycle(t *testing.T) {
	RegisterComponent("test/lifecycle", func() Component { return &testComponent{} })

	scene := NewScene("main")
	comp, err := CreateComponentFromJSON("test/lifecycle", "c", nil)
	if err != nil {
		t.Fatal(err)
	}
	obj := NewObject("o")
	if err := obj.AddComponent(comp); err != nil {
		t.Fatal(err)
	}
	if err := scene.AddObject(obj); err != nil {
		t.Fatal(err)
	}

	tc := comp.(*testComponent)
	if tc.initialized != 0 {
		t.Fatalf("Initialize ran before the first scene Update (%d)", tc.initialized)
	}

	scene.Update(&Context{})

	if tc.initialized != 1 {
		t.Fatalf("expected Initialize once after first Update, got %d", tc.initialized)
	}
	if tc.enabled != 1 {
		t.Fatalf("expected OnEnable once, got %d", tc.enabled)
	}
	if tc.updated != 1 {
		t.Fatalf("expected Update once, got %d", tc.updated)
	}

	scene.Update(&Context{})

	if tc.initialized != 1 {
		t.Fatalf("Initialize ran more than once: %d", tc.initialized)
	}
	if tc.updated != 2 {
		t.Fatalf("expected Update twice, got %d", tc.updated)
	}
}

func TestComponentEvents(t *testing.T) {
	RegisterComponent("test/emitter", func() Component { return &testComponent{} })
	RegisterComponent("test/listener", func() Component { return &testComponent{} })

	scene := NewScene("main")

	emitter, _ := CreateComponentFromJSON("test/emitter", "e", nil)
	listener, _ := CreateComponentFromJSON("test/listener", "l", nil)

	obj := NewObject("o")
	if err := obj.AddComponent(emitter); err != nil {
		t.Fatal(err)
	}
	if err := obj.AddComponent(listener); err != nil {
		t.Fatal(err)
	}
	if err := scene.AddObject(obj); err != nil {
		t.Fatal(err)
	}

	le := emitter.(*testComponent)
	li := listener.(*testComponent)

	// Register a handler (normally done in Initialize).
	li.On("boom", func(data any) {
		li.gotEvents = append(li.gotEvents, data)
	})

	// First Update initializes and syncs the subscription.
	scene.Update(&Context{})

	le.Emit("boom", 42)
	scene.Update(&Context{}) // delivers the queued event

	if len(li.gotEvents) != 1 || li.gotEvents[0] != 42 {
		t.Fatalf("expected one event 42, got %v", li.gotEvents)
	}
}

func TestComponentConfigInjection(t *testing.T) {
	type configured struct {
		BaseComponent
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}
	RegisterComponent("test/config", func() Component { return &configured{} })

	comp, err := CreateComponentFromJSON("test/config", "c", map[string]interface{}{
		"width":  10.0,
		"height": 20.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := comp.(*configured)
	if got.Width != 10 || got.Height != 20 {
		t.Fatalf("expected (10,20), got (%v,%v)", got.Width, got.Height)
	}
}
