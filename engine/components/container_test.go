package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// newContainerFixture builds an object holding a container plus three named labels,
// returning the container so tests can tweak layout and assert offsets.
func newContainerFixture() (*ContainerComponent, *LabelComponent, *LabelComponent, *LabelComponent) {
	obj := core.NewObject("panel")

	cont := &ContainerComponent{}
	cont.SetName("layout")
	cont.Offset = math.NewVector2(10, 20)
	cont.Children = []string{"a", "b", "c"}
	cont.Gap = 5

	a := &LabelComponent{}
	a.SetName("a")
	a.Width = 40
	a.Height = 10

	b := &LabelComponent{}
	b.SetName("b")
	b.Width = 30
	b.Height = 15

	c := &LabelComponent{}
	c.SetName("c")
	c.Width = 20
	c.Height = 5

	_ = obj.AddComponent(cont)
	_ = obj.AddComponent(a)
	_ = obj.AddComponent(b)
	_ = obj.AddComponent(c)

	return cont, a, b, c
}

func TestContainerDefaultsToVertical(t *testing.T) {
	cont, _, _, _ := newContainerFixture()
	cont.Initialize()
	if cont.layout() != "vertical" {
		t.Fatalf("default layout = %q, want vertical", cont.layout())
	}
}

func TestContainerVerticalLayout(t *testing.T) {
	cont, a, b, c := newContainerFixture()
	cont.Initialize()
	cont.Layout = "vertical"

	cont.Update(&core.Context{})

	if got := a.Offset; got != (math.NewVector2(10, 20)) {
		t.Fatalf("a offset = %v, want (10,20)", got)
	}
	if got := b.Offset; got != (math.NewVector2(10, 35)) {
		t.Fatalf("b offset = %v, want (10,35)", got)
	}
	if got := c.Offset; got != (math.NewVector2(10, 55)) {
		t.Fatalf("c offset = %v, want (10,55)", got)
	}
}

func TestContainerHorizontalLayout(t *testing.T) {
	cont, a, b, c := newContainerFixture()
	cont.Initialize()
	cont.Layout = "horizontal"

	cont.Update(&core.Context{})

	if got := a.Offset; got != (math.NewVector2(10, 20)) {
		t.Fatalf("a offset = %v, want (10,20)", got)
	}
	if got := b.Offset; got != (math.NewVector2(55, 20)) {
		t.Fatalf("b offset = %v, want (55,20)", got)
	}
	if got := c.Offset; got != (math.NewVector2(90, 20)) {
		t.Fatalf("c offset = %v, want (90,20)", got)
	}
}

func TestContainerPadding(t *testing.T) {
	cont, a, _, _ := newContainerFixture()
	cont.Initialize()
	cont.Padding = math.Border{Left: 4, Top: 6}

	cont.Update(&core.Context{})

	if got := a.Offset; got != (math.NewVector2(14, 26)) {
		t.Fatalf("a offset with padding = %v, want (14,26)", got)
	}
}

func TestContainerSkipsInvisibleChild(t *testing.T) {
	cont, a, b, c := newContainerFixture()
	cont.Initialize()

	b.SetVisible(false)
	cont.Update(&core.Context{})

	if got := a.Offset; got != (math.NewVector2(10, 20)) {
		t.Fatalf("a offset = %v, want (10,20)", got)
	}
	// b is skipped and its slot collapsed; c follows a directly.
	if got := c.Offset; got != (math.NewVector2(10, 35)) {
		t.Fatalf("c offset = %v, want (10,35)", got)
	}
}
