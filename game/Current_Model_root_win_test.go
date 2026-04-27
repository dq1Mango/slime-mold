//go:build current_model

package main

import (
	"testing"
)

func TestWinnerByRootOrigin(t *testing.T) {
	game := &LiveGame{
		model: Model{
			size:     11,
			grid:     gen_grid(11),
			rootGrid: genIntGrid(11, ROOT_NONE),
			players: []Player{
				{spawn: vectorFromPoint(Point{X: 2, Y: 2}), rootID: 0},
				{spawn: vectorFromPoint(Point{X: 8, Y: 8}), rootID: 1},
			},
		},
		HasClicked: []bool{true, true},
	}

	game.model.seedOriginCluster(Point{X: 2, Y: 2}, 1, 0)
	game.model.seedOriginCluster(Point{X: 8, Y: 8}, 2, 1)
	game.model.grid[8][10] = Trail{playerNum: 1, value: 1}

	game.updateGameOverState()

	if !game.GameOver || game.Winner != 1 {
		t.Fatalf("expected player 1 to win after blue origin zone was touched by red, got gameover=%v winner=%d", game.GameOver, game.Winner)
	}
}

func TestOriginClusterSeeding(t *testing.T) {
	model := Model{
		size:     7,
		grid:     gen_grid(7),
		rootGrid: genIntGrid(7, ROOT_NONE),
	}

	model.seedOriginCluster(Point{X: 3, Y: 3}, 1, 0)

	count := 0
	for y := 0; y < model.size; y++ {
		for x := 0; x < model.size; x++ {
			if model.grid[y][x].playerNum == 1 {
				count++
			}
		}
	}

	if count != 13 {
		t.Fatalf("expected 13 seeded origin cells, got %d", count)
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
