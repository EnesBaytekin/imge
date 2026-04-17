// Package components provides custom components for the basic sample game.
package components

import (
	"math"

	"github.com/EnesBaytekin/imge/core"
	coremath "github.com/EnesBaytekin/imge/core/math"
)

// EnemyAI component makes an object chase the player.
type EnemyAI struct {
	core.BaseComponent
	speed float64
	playerTag string
}

func (c *EnemyAI) Initialize(args []interface{}) error {
	c.speed = 100.0 // default speed (pixels per second)
	c.playerTag = "player"
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if speed, ok := argMap["speed"].(float64); ok {
				c.speed = speed
			}
			if tag, ok := argMap["player_tag"].(string); ok {
				c.playerTag = tag
			}
		}
	}
	return nil
}

func (c *EnemyAI) Update(deltaTime float64) {
	owner := c.GetOwner()
	if owner == nil {
		return
	}

	scene := core.GetSceneFromComponent(c)
	if scene == nil {
		return
	}

	// Find player object by tag
	playerObjects := scene.FindObjectsWithTag(c.playerTag)
	if len(playerObjects) == 0 {
		return // No player found
	}

	// Get first player object
	player := playerObjects[0]
	playerPos := player.GetPosition()
	enemyPos := owner.Transform.Position

	// Calculate direction vector towards player
	dir := coremath.Vector2{
		X: playerPos.X - enemyPos.X,
		Y: playerPos.Y - enemyPos.Y,
	}

	// Normalize direction vector
	length := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)
	if length > 0 {
		dir.X /= length
		dir.Y /= length
	} else {
		// Already at player position
		return
	}

	// Move enemy towards player
	moveDistance := c.speed * deltaTime
	if moveDistance > length {
		moveDistance = length // Don't overshoot
	}

	owner.Transform.Position = coremath.Vector2{
		X: enemyPos.X + dir.X * moveDistance,
		Y: enemyPos.Y + dir.Y * moveDistance,
	}
}

func (c *EnemyAI) Draw(renderer core.Renderer) {
	// No drawing needed
}

func init() {
	core.RegisterComponent("components/enemy.go", func(args []interface{}) (core.Component, error) {
		comp := &EnemyAI{}
		if err := comp.Initialize(args); err != nil {
			return nil, err
		}
		return comp, nil
	})
}