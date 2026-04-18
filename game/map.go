package main

import (
	"encoding/json"
	"math"
	"os"
)

const MAP_PATH = "../maps/"

type Food struct {
	Quantity        int
	ConsumptionRate float64 // pieces per tick or second idk yet
	Position        Point
}

type Map struct {
	Foods  []Food
	Spawn1 Point
	Spawn2 Point

	// dont export this one so it doens't get jsoned
	dangerGradient func(Vector) float64
}

// i dont rly know how to encode the json gradient, prolly just gonna be
// a list of anonymous (among us???) functions in .DATA
func loadMap(path string) *Map {

	// map is a keyword lol
	var theMap Map

	json_data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(json_data, &theMap)

	theMap.dangerGradient = defaulDangerGradient()

	return &theMap
}

func (m *Map) writeMap(path string) {

	marshallMathers, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	os.WriteFile(path, marshallMathers, 0644)

}

// yeah this is super shit just wanna see what its like
func defaulDangerGradient() func(v Vector) float64 {
	safeRect := Rect{x1: -75, x2: 75, y1: 25, y2: -25}

	return func(v Vector) float64 {
		if safeRect.inside(v) {
			return 0
		}

		return v.magnitude() / (math.Sqrt2 * GRID_SIZE)

	}
}

func defaultMap() *Map {

	Spawn1 := Point{-50, 0}
	Spawn2 := Point{50, 0}
	// Spawn1Vec := vectorFromPoint(Spawn1)
	// Spawn2Vec := vectorFromPoint(Spawn2)

	return &Map{
		Foods: []Food{
			{
				Quantity:        100,
				ConsumptionRate: 5,
				Position:        Point{50, 50},
			},
			{
				Quantity:        30,
				ConsumptionRate: 3,
				Position:        Point{50, 75},
			},
			{
				Quantity:        30,
				ConsumptionRate: 3,
				Position:        Point{50, 25},
			},
			{
				Quantity:        60,
				ConsumptionRate: 1,
				Position:        Point{25, 50},
			},
			{
				Quantity:        60,
				ConsumptionRate: 1,
				Position:        Point{75, 50},
			},
		},

		Spawn1:         Spawn1,
		Spawn2:         Spawn2,
		dangerGradient: defaulDangerGradient(),
	}
}

type Rect struct {
	x1, y1, x2, y2 int
}

// IMPORTANT: remember to flip the y comparisons when u remove the cartesian abstraction
// (hes going to spend 30 mins wondering why it doens't work and then get pissed
// when he reads this comment)
func (r *Rect) inside(v Vector) bool {
	x1, y1, x2, y2 := float64(r.x1), float64(r.y1), float64(r.x2), float64(r.y2)
	return v.x >= x1 && v.x <= x2 && v.y < y1 && v.y > y2
}
