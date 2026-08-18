package components

import (
	"github.com/EnesBaytekin/imge/core"
)

// Health is an HP pool. Damage/Heal mutate it and emit scene-global events:
// "damaged" (data = amount) and "died" (data = the owner object).
//
// Export variables (JSON args): max.
type Health struct {
	core.BaseComponent

	Max     int `json:"max"`
	current int
}

// Initialize applies defaults and fills the pool.
func (h *Health) Initialize() {
	if h.Max <= 0 {
		h.Max = 100
	}
	h.current = h.Max
}

// Damage subtracts amount (ignored when dead or non-positive) and emits events.
func (h *Health) Damage(amount int) {
	if amount <= 0 || h.current <= 0 {
		return
	}
	h.current -= amount
	if h.current < 0 {
		h.current = 0
	}
	if h.current <= 0 {
		h.Emit("died", h.GetOwner())
		return
	}
	h.Emit("damaged", amount)
}

// Heal restores amount, clamped to Max.
func (h *Health) Heal(amount int) {
	if amount <= 0 || h.current >= h.Max {
		return
	}
	h.current += amount
	if h.current > h.Max {
		h.current = h.Max
	}
}

// Current returns the current HP.
func (h *Health) Current() int { return h.current }

// IsDead reports whether the pool is empty.
func (h *Health) IsDead() bool { return h.current <= 0 }
