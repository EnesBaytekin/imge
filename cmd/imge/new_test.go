package main

import "testing"

func TestPascalCase(t *testing.T) {
	cases := map[string]string{
		"enemy":       "Enemy",
		"enemy_brain": "EnemyBrain",
		"enemy-brain": "EnemyBrain",
		"EnemyBrain":  "EnemyBrain",
		"enemy brain": "EnemyBrain",
		"player2":     "Player2",
		"ENEMY":       "Enemy",
	}
	for in, want := range cases {
		if got := pascalCase(in); got != want {
			t.Errorf("pascalCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFileName(t *testing.T) {
	cases := map[string]string{
		"player":      "player",
		"Player":      "player",
		"EnemyBrain":  "enemybrain",
		"enemy_brain": "enemy_brain",
		"enemy brain": "enemy_brain",
		"enemy-brain": "enemy_brain",
	}
	for in, want := range cases {
		if got := fileName(in); got != want {
			t.Errorf("fileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTrimExt(t *testing.T) {
	cases := map[string]string{
		"player":     "player",
		"player.obj": "player",
		"player.OBJ": "player",
		"player.go":  "player.go",
		"scene":      "scene",
		".obj":       ".obj",
	}
	for in, want := range cases {
		if got := trimExt(in, ".obj"); got != want {
			t.Errorf("trimExt(%q, \".obj\") = %q, want %q", in, got, want)
		}
	}
}
