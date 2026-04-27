//go:build current_model

package main

import (
	"testing"
)

func TestWinnerByRootOrigin(t *testing.T) {
	game := &LiveGame{
		model: Model{
			size:     5,
			grid:     gen_grid(5),
			rootGrid: genIntGrid(5, ROOT_NONE),
			players: []Player{
				{spawn: vectorFromPoint(Point{X: 1, Y: 1}), rootID: 0},
				{spawn: vectorFromPoint(Point{X: 3, Y: 3}), rootID: 1},
			},
		},
		HasClicked: []bool{true, true},
	}

	game.model.grid[1][1] = Trail{playerNum: 1, value: 1}
	game.model.grid[3][3] = Trail{playerNum: 1, value: 1}

	game.updateGameOverState()

	if !game.GameOver || game.Winner != 1 {
		t.Fatalf("expected player 1 to win after blue origin takeover, got gameover=%v winner=%d", game.GameOver, game.Winner)
	}
}

func TestTouchingEnemyCell(t *testing.T) {
	model := Model{size: 5, grid: gen_grid(5)}
	model.grid[2][2] = Trail{playerNum: 1, value: 1}
	model.grid[2][3] = Trail{playerNum: 2, value: 1}
	model.grid[1][2] = Trail{playerNum: 2, value: 1}

	contact, enemy, ok := model.touchingOpponentCell(Point{X: 2, Y: 2}, 1)
	if !ok {
		t.Fatal("expected one touching enemy cell")
	}
	if enemy != 2 {
		t.Fatalf("expected enemy player 2, got %d", enemy)
	}
	if contact != (Point{X: 3, Y: 2}) && contact != (Point{X: 2, Y: 1}) {
		t.Fatalf("unexpected contact cell: %+v", contact)
	}
}
