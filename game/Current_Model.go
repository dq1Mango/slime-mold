//go:build current_model

package main

import (
	"flag"
	"fmt"

	// "golang.org/x/text/language"
	_ "embed"
	"image"
	"image/color"
	_ "image/png"
	"log/slog"
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/montanaflynn/stats"

	// "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	// "slices"
)

const END_RATIO = 0.01
const GRID_SIZE = 200

const SACRIFICE = 0
const SELFISH = 0

const SPLIT = 0
const MAX_CELL_PARTICLES = 6
const TOTAL_PARTICLE_RESOURCES = 1000
const HIGH_FLOW_THRESHOLD = 10
const RESOURCE_PRESSURE_THRESHOLD = 2
const RESOURCE_REFILL_BATCH = 8
const RESOURCE_CAP_REGEN_PER_TICK = 50
const RESOURCE_CAP_BONUS_MAX = 200
const RECLAIM_SAMPLE_TRIES = 48
const RECLAIM_MAX_CONNECTIVITY_CHECKS = 6
const MIN_WALKER_INTENSITY = 1.0
const FORCE_CAMP_RADIUS = 8
const FORCE_CAMP_PULL = 1.0
const FORCE_CAMP_MIN_COMMIT = 0.92
const FORCE_CAMP_COMMIT_FRACTION = 0.35
const FORCE_CAMP_MIXED_ROOT_WEIGHT = 0.2
const ROOT_NONE = -1
const ROOT_MIXED = -2

const (
	FPS         = 20.0
	SCREEN_SIZE = 1000
	SCALE       = float64(SCREEN_SIZE) / GRID_SIZE
	LIVE_PULL   = 0.2

	// game ticks per second
	TPS = 20
	// DELAY = 1 / TPS
)

var WEIGHT_VECTOR = Vector{x: -1, y: 1}
var LIVE_FORCE_ATTRACT = false
var LIVE_MOUSE_TARGET = Vector{x: 0, y: 0}
var LIVE_MOUSE_POINT = Point{X: 0, Y: 0}

var OatImage *ebiten.Image

type SiteState int

const (
	Empty SiteState = iota
	Filled

	Origin SiteState = -2
	Active SiteState = -1
)

var StateColor = map[SiteState]color.NRGBA{
	Empty: {
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	},
	Filled: {
		R: 255,
		G: 0,
		B: 0,
		A: 255,
	},
	Origin: {
		R: 0,
		G: 0,
		B: 255,
		A: 255,
	},
	Active: {
		R: 0,
		G: 255,
		B: 0,
		A: 255,
	},
}

// ffdsf go:embed NotoSans-Regular.ttf
// var arabicTTF []byte

var fontFace text.Face = text.NewGoXFace(basicfont.Face7x13)

// var arabicFaceSource *text.GoTextFaceSource

type Grid [][]SiteState

// these field r now exported to get json-ed
type Point struct {
	X int
	Y int
}

var CARDINALS = []Point{
	{X: 1, Y: 0},
	{X: 0, Y: 1},
	{X: -1, Y: 0},
	{X: 0, Y: -1},
}

type Vector struct {
	x float64
	y float64
}

func (v *Vector) add(v2 Vector) {
	v.x += v2.x
	v.y += v2.y
}

func (v *Vector) rotate(theta float64) {
	a, b, c, d := math.Cos(theta), -math.Sin(theta), math.Sin(theta), math.Cos(theta)

	output := *v

	output.x = a*v.x + b*v.y
	output.y = c*v.x + d*v.y

	*v = output
}

func subtract(v1, v2 Vector) Vector {

	return Vector{x: v1.x - v2.x, y: v1.y - v2.y}
}

func distance(v1, v2 Vector) float64 {
	delta := subtract(v1, v2)

	return delta.magnitude()
}

var UNITS = []Vector{
	{x: 1, y: 0},
	{x: 0, y: 1},
	{x: -1, y: 0},
	{x: 0, y: -1},
}

func (v *Vector) scale(r float64) {
	v.x *= r
	v.y *= r
}

func (v *Vector) magnitude() float64 {
	return math.Sqrt(v.x*v.x + v.y*v.y)
}

func (v *Vector) normalize() {
	magnitude := v.magnitude()

	if magnitude == 0 {
		return
	}

	v.scale(1 / magnitude)
}

func vectorFromPoint(p Point) Vector {
	return Vector{
		x: float64(p.X),
		y: float64(p.Y),
	}
}

func forceField(v, weight Vector) Vector {

	if v.magnitude() == 0 || weight.magnitude() == 0 {
		return Vector{x: 0, y: 0}
	}

	angle := math.Atan2(v.x*weight.y-v.y*weight.x, dotProduct(v, weight))
	radius_fraction := v.magnitude() / (GRID_SIZE / 2)
	if radius_fraction > 1 {
		radius_fraction = 1
	}

	output := v

	output.rotate(angle * radius_fraction)

	output.normalize()

	return output
}

func attractionForce(from, to Vector) Vector {
	force := Vector{x: to.x - from.x, y: to.y - from.y}

	if force.magnitude() == 0 {
		if WEIGHT_VECTOR.magnitude() == 0 {
			return Vector{x: 0, y: 0}
		}

		fallback := WEIGHT_VECTOR
		fallback.normalize()
		return fallback
	}

	force.normalize()
	return force
}

const MAX_DISTANCE = 1000

var ZERO_VECTOR = Vector{x: 0, y: 0}

func curvyForce(relativePos, weight Vector) Vector {

	weight.normalize()

	if relativePos.magnitude() == 0 {
		return weight
		// return Vector{0, 0}
		// return ZERO_VECTOR
	}

	if relativePos.magnitude() > MAX_DISTANCE {
		// v.normalize()
		return weight
	}

	delat_theta := math.Acos(
		dotProduct(
			relativePos,
			weight,
		) / relativePos.magnitude() / weight.magnitude(),
	)

	if math.IsNaN(delat_theta) {
		return weight
	}

	if delat_theta > math.Pi/4 {
		return weight
	}

	weight_theta := math.Atan(weight.y / weight.x)

	if weight.x < 0 {
		weight_theta += math.Pi
	}

	v_theta := math.Atan(relativePos.y / relativePos.x)

	if relativePos.x < 0 {
		v_theta += math.Pi
	}

	if v_theta < weight_theta {
		delat_theta *= -1
	}

	radius_fraction := relativePos.magnitude() / (GRID_SIZE / 2)

	output := relativePos

	output.rotate(delat_theta * radius_fraction)

	output.normalize()

	return output
}

func mid_point() Point {
	mid := (GRID_SIZE - 1) / 2

	return Point{X: mid, Y: mid}
}

func real_mid_point(size int) Point {

	mid := (size - 1) / 2

	return Point{X: mid, Y: mid}
}

func add_points(p1, p2 Point) Point {
	return Point{X: p1.X + p2.X, Y: p1.Y + p2.Y}
}

func add_vectors(v1, v2 Vector) Vector {
	return Vector{x: v1.x + v2.x, y: v1.y + v2.y}
}

// func (p *Point) add(p2 Point) {
// 	p.x += p2.x
// 	p.y += p2.y
// }

// func (p *Point) flip() {
// 	p.x *= -1
// 	p.y *= -1
// }

func abs(x int) int {
	return max(x, x*-1)
}

func (p *Point) radius() int {
	return max(abs(p.X), abs(p.Y))
}

type Walker struct {
	location Point
	ttl      int
}

type TreeWalker struct {
	location  Vector
	intensity float64
	velocity  Vector
	rootID    int
}

type Model struct {
	grid            Grid
	nextGrid        Grid
	rootGrid        [][]int
	nextRoot        [][]int
	walkers         []TreeWalker
	grids           []Grid
	size            int
	spawn           Vector
	nextRootID      int
	particlesInGrid int
	freeParticles   int
	// p        float64
	// people   int
	// infected int
	radius int
	time   int
	// distance int
}

func (m *Model) spawnWalker() {
	middle := mid_point()
	rootID := m.allocateRootID()
	m.walkers = append(m.walkers, TreeWalker{location: vectorFromPoint(middle), intensity: 100, rootID: rootID})
}

// This spawns walkers on all existing trail points near target (usually the mouse).
// The selected region is the closest radial distance plus a small outward buffer.
func (m *Model) spawnWalkerAtNearestPlacedParticle(target Point) bool {
	if len(m.grid) == 0 {
		return false
	}

	closestDistSq := 0
	found := false

	type candidate struct {
		point  Point
		distSq int
	}

	candidates := make([]candidate, 0, m.size*m.size)

	for y := range m.size {
		for x := range m.size {
			if m.grid[y][x] <= Empty {
				continue
			}

			dx := x - target.X
			dy := y - target.Y
			distSq := dx*dx + dy*dy
			candidates = append(candidates, candidate{point: Point{X: x, Y: y}, distSq: distSq})

			if !found || distSq < closestDistSq {
				closestDistSq = distSq
				found = true
			}
		}
	}

	if !found {
		return false
	}

	const blendWindow = 8.0
	const maxSelectedPoints = 5
	minDist := math.Sqrt(float64(closestDistSq))

	type nearbyCandidate struct {
		point Point
		dist  float64
	}

	selected := make([]nearbyCandidate, 0, len(candidates))

	for _, c := range candidates {
		dist := math.Sqrt(float64(c.distSq))
		if dist <= minDist+blendWindow {
			selected = append(selected, nearbyCandidate{point: c.point, dist: dist})
		}
	}

	if len(selected) == 0 {
		return false
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].dist < selected[j].dist
	})

	if len(selected) > maxSelectedPoints {
		selected = selected[:maxSelectedPoints]
	}

	const spawnResourceBudget = 100.0
	rootID := m.allocateRootID()
	totalWeight := 0.0
	weights := make([]float64, len(selected))
	for i, c := range selected {
		weight := 1.0 / (c.dist + 1.0)
		weights[i] = weight
		totalWeight += weight
	}

	for i, c := range selected {
		intensity := spawnResourceBudget * (weights[i] / totalWeight)
		m.walkers = append(
			m.walkers,
			TreeWalker{location: vectorFromPoint(c.point), intensity: intensity, rootID: rootID},
		)
	}

	return true
}

func (m *Model) clear() {
	m.grid = gen_grid(m.size)
	m.nextGrid = gen_grid(m.size)
	m.rootGrid = genIntGrid(m.size, ROOT_NONE)
	m.nextRoot = genIntGrid(m.size, ROOT_NONE)
	m.nextRootID = 0
	m.particlesInGrid = 0
	m.freeParticles = TOTAL_PARTICLE_RESOURCES
	m.walkers = []TreeWalker{}
}

func (m *Model) allocateRootID() int {
	rootID := m.nextRootID
	m.nextRootID++
	return rootID
}

func (m *Model) cullWeakWalkers() {
	alive := m.walkers[:0]
	for _, walker := range m.walkers {
		if walker.intensity >= MIN_WALKER_INTENSITY {
			alive = append(alive, walker)
		}
	}
	m.walkers = alive
}

func directionStep(v Vector) Point {
	if math.Abs(v.x) >= math.Abs(v.y) {
		if v.x > 0 {
			return Point{X: 1, Y: 0}
		}
		if v.x < 0 {
			return Point{X: -1, Y: 0}
		}
	}

	if v.y > 0 {
		return Point{X: 0, Y: 1}
	}
	if v.y < 0 {
		return Point{X: 0, Y: -1}
	}

	return Point{X: 0, Y: 0}
}

func (m *Model) isOccupied(p Point) bool {
	if p.X < 0 || p.X >= m.size || p.Y < 0 || p.Y >= m.size {
		return false
	}

	return m.grid[p.Y][p.X] > 0
}

func (m *Model) isEdgeParticleCell(p Point) bool {
	if !m.isOccupied(p) {
		return false
	}

	for _, d := range CARDINALS {
		n := add_points(p, d)
		if !m.isOccupied(n) {
			return true
		}
	}

	return false
}

func (m *Model) canRemoveCellSafely(p Point) bool {
	if !m.isOccupied(p) {
		return false
	}

	totalCells := m.size * m.size
	remaining := 0
	start := Point{X: -1, Y: -1}
	for y := range m.size {
		for x := range m.size {
			if x == p.X && y == p.Y {
				continue
			}
			if m.grid[y][x] <= 0 {
				continue
			}
			remaining++
			if start.X == -1 {
				start = Point{X: x, Y: y}
			}
		}
	}

	if remaining <= 1 {
		return true
	}

	queue := []Point{start}
	visited := make([]bool, totalCells)
	visited[start.Y*m.size+start.X] = true
	visitedCount := 1

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range CARDINALS {
			n := add_points(cur, d)
			if n.X < 0 || n.X >= m.size || n.Y < 0 || n.Y >= m.size {
				continue
			}
			if n.X == p.X && n.Y == p.Y {
				continue
			}
			idx := n.Y*m.size + n.X
			if m.grid[n.Y][n.X] <= 0 || visited[idx] {
				continue
			}
			visited[idx] = true
			visitedCount++
			queue = append(queue, n)
		}
	}

	return visitedCount == remaining
}

func (m *Model) dissolveDetachedFromAnchor(anchor Point) {
	start := Point{X: -1, Y: -1}
	bestDistSq := math.MaxInt

	for y := range m.size {
		for x := range m.size {
			if m.grid[y][x] <= 0 {
				continue
			}
			dx := x - anchor.X
			dy := y - anchor.Y
			distSq := dx*dx + dy*dy
			if distSq < bestDistSq {
				bestDistSq = distSq
				start = Point{X: x, Y: y}
			}
		}
	}

	if start.X == -1 {
		return
	}

	queue := []Point{start}
	visited := map[Point]bool{start: true}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range CARDINALS {
			n := add_points(cur, d)
			if n.X < 0 || n.X >= m.size || n.Y < 0 || n.Y >= m.size {
				continue
			}
			if m.grid[n.Y][n.X] <= 0 || visited[n] {
				continue
			}
			visited[n] = true
			queue = append(queue, n)
		}
	}

	for y := range m.size {
		for x := range m.size {
			p := Point{X: x, Y: y}
			if m.grid[y][x] <= 0 || visited[p] {
				continue
			}
			removed := int(m.grid[y][x])
			m.particlesInGrid -= removed
			m.freeParticles += removed
			m.grid[y][x] = 0
			m.rootGrid[y][x] = ROOT_NONE
		}
	}
}

func (m *Model) reclaimFromWouldBeDisconnected(anchor, cut Point) bool {
	start := Point{X: -1, Y: -1}
	bestDistSq := math.MaxInt

	for y := range m.size {
		for x := range m.size {
			if (x == cut.X && y == cut.Y) || m.grid[y][x] <= 0 {
				continue
			}
			dx := x - anchor.X
			dy := y - anchor.Y
			distSq := dx*dx + dy*dy
			if distSq < bestDistSq {
				bestDistSq = distSq
				start = Point{X: x, Y: y}
			}
		}
	}

	if start.X == -1 {
		return false
	}

	queue := []Point{start}
	connected := map[Point]bool{start: true}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range CARDINALS {
			n := add_points(cur, d)
			if n.X < 0 || n.X >= m.size || n.Y < 0 || n.Y >= m.size {
				continue
			}
			if (n.X == cut.X && n.Y == cut.Y) || m.grid[n.Y][n.X] <= 0 || connected[n] {
				continue
			}
			connected[n] = true
			queue = append(queue, n)
		}
	}

	detachedFound := false
	detachedBest := Point{}
	detachedBestWeight := -1.0

	for y := range m.size {
		for x := range m.size {
			p := Point{X: x, Y: y}
			if (x == cut.X && y == cut.Y) || m.grid[y][x] <= 0 || connected[p] {
				continue
			}
			detachedFound = true

			level := m.grid[y][x]
			weight := 1.0 / float64(level)
			if level >= HIGH_FLOW_THRESHOLD {
				weight *= 0.35
			}
			if weight > detachedBestWeight {
				detachedBestWeight = weight
				detachedBest = p
			}
		}
	}

	if !detachedFound {
		return false
	}

	m.grid[detachedBest.Y][detachedBest.X] -= 1
	if m.grid[detachedBest.Y][detachedBest.X] <= 0 {
		m.rootGrid[detachedBest.Y][detachedBest.X] = ROOT_NONE
	}
	m.particlesInGrid--
	m.freeParticles++
	return true
}

func (m *Model) refillFreeParticles(anchor Point) bool {
	if m.freeParticles > 0 {
		return true
	}

	for range RESOURCE_REFILL_BATCH {
		if !m.reclaimOneParticle(anchor) {
			break
		}
		if m.freeParticles > 0 {
			return true
		}
	}

	return m.freeParticles > 0
}

func (m *Model) regenerateResourceCap() {
	maxFree := TOTAL_PARTICLE_RESOURCES + RESOURCE_CAP_BONUS_MAX
	if m.freeParticles >= maxFree {
		return
	}

	m.freeParticles = min(maxFree, m.freeParticles+RESOURCE_CAP_REGEN_PER_TICK)
}

func (m *Model) applyResourcePressure(anchor Point) bool {
	utilization := float64(m.particlesInGrid) / float64(TOTAL_PARTICLE_RESOURCES)
	if utilization <= RESOURCE_PRESSURE_THRESHOLD {
		return true
	}

	// Past threshold, reclaim more than we add so dense states thin out gradually.
	pressure := (utilization - RESOURCE_PRESSURE_THRESHOLD) / (1.0 - RESOURCE_PRESSURE_THRESHOLD)
	reclaims := 1 + int(pressure)

	for range reclaims {
		if !m.reclaimOneParticle(anchor) {
			return false
		}
	}

	return true
}

func (m *Model) reclaimOneParticle(anchor Point) bool {
	_ = anchor
	totalCells := m.size * m.size

	bestFallback := Point{}
	bestWeight := -1.0
	foundFallback := false
	connectivityChecks := 0

	for range RECLAIM_SAMPLE_TRIES {
		idx := rand.Intn(totalCells)
		y := idx / m.size
		x := idx % m.size

		candidate := Point{X: x, Y: y}
		level := m.grid[y][x]
		if level <= 0 || !m.isEdgeParticleCell(candidate) {
			continue
		}
		if level == 1 {
			if connectivityChecks >= RECLAIM_MAX_CONNECTIVITY_CHECKS {
				continue
			}
			connectivityChecks++
			if !m.canRemoveCellSafely(candidate) {
				continue
			}
		}

		weight := 1.0 / float64(level)
		if level >= HIGH_FLOW_THRESHOLD {
			weight *= 0.35
		}

		if !foundFallback || weight > bestWeight {
			bestWeight = weight
			bestFallback = candidate
			foundFallback = true
		}

		if rand.Float64() < weight {
			m.grid[candidate.Y][candidate.X] -= 1
			if m.grid[candidate.Y][candidate.X] <= 0 {
				m.rootGrid[candidate.Y][candidate.X] = ROOT_NONE
			}
			m.particlesInGrid--
			m.freeParticles++
			return true
		}
	}

	if !foundFallback {
		start := rand.Intn(totalCells)
		for offset := range totalCells {
			idx := (start + offset) % totalCells
			y := idx / m.size
			x := idx % m.size

			candidate := Point{X: x, Y: y}
			level := m.grid[y][x]
			if level <= 0 || !m.isEdgeParticleCell(candidate) {
				continue
			}
			if level == 1 {
				if connectivityChecks >= RECLAIM_MAX_CONNECTIVITY_CHECKS {
					continue
				}
				connectivityChecks++
				if !m.canRemoveCellSafely(candidate) {
					continue
				}
			}

			weight := 1.0 / float64(level)
			if level >= HIGH_FLOW_THRESHOLD {
				weight *= 0.35
			}
			if !foundFallback || weight > bestWeight {
				bestWeight = weight
				bestFallback = candidate
				foundFallback = true
			}
		}
	}

	if foundFallback {
		m.grid[bestFallback.Y][bestFallback.X] -= 1
		if m.grid[bestFallback.Y][bestFallback.X] <= 0 {
			m.rootGrid[bestFallback.Y][bestFallback.X] = ROOT_NONE
		}
		m.particlesInGrid--
		m.freeParticles++
		return true
	}

	return false
}

func (m *Model) addParticleAt(p Point, rootID int) bool {
	if p.X < 0 || p.X >= m.size || p.Y < 0 || p.Y >= m.size {
		return false
	}
	if m.grid[p.Y][p.X] >= MAX_CELL_PARTICLES {
		return false
	}

	if !m.applyResourcePressure(p) {
		return false
	}

	if !m.refillFreeParticles(p) {
		if !m.reclaimOneParticle(p) {
			return false
		}
	}

	m.grid[p.Y][p.X] += 1
	m.particlesInGrid++
	m.freeParticles--

	existingOwner := m.rootGrid[p.Y][p.X]
	if existingOwner == ROOT_NONE {
		m.rootGrid[p.Y][p.X] = rootID
	} else if existingOwner != rootID {
		m.rootGrid[p.Y][p.X] = ROOT_MIXED
	}

	return true
}

func (m *Model) depositWithOverflow(target Point, travel Vector, rootID int) bool {
	spot := target
	step := directionStep(travel)

	for range 4 {
		if m.addParticleAt(spot, rootID) {
			return true
		}

		if step.X == 0 && step.Y == 0 {
			break
		}

		spot = add_points(spot, step)
		if spot.X < 0 || spot.X >= m.size || spot.Y < 0 || spot.Y >= m.size {
			break
		}
	}

	return false
}

// func remove[T any](slice []T, s int) []T {
// 	return slices.Delete(slice, s, s+1)
// }

func gen_grid(size int) Grid {

	// if size%2 == 0 {
	// 	panic("grid size must be odd you doofus")
	// }

	grid := make(Grid, size)

	for row := range grid {
		grid[row] = make([]SiteState, size)
	}

	return grid
}

func genIntGrid(size int, fill int) [][]int {
	grid := make([][]int, size)
	for y := range size {
		grid[y] = make([]int, size)
		for x := range size {
			grid[y][x] = fill
		}
	}

	return grid
}

// func heart_equation_derive(x, y float64) float64 {
// 	return 3 * math.Pow(x * x + y * y - 1, 2)
//
// }

func round(x float64) int {
	return int(math.Round(x))
}

func heart_equation(t float64) (float64, float64) {
	x := 16 * math.Pow(math.Sin(t), 3)
	y := 13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t)

	y /= -15
	x /= 15

	return x, y
}

func gen_heart_grid(size int, radius float64) Grid {

	if int(radius) > size/2 {
		panic(fmt.Errorf("ahhhhh: radius too large for grid size"))
	}

	grid := gen_grid(size)

	points := 1000

	for i := range points {
		t := float64(i) / float64(points) * 2 * math.Pi

		x, y := heart_equation(t)
		point := Point{X: int(math.Round(x * radius)), Y: int(math.Round(y * radius))}

		*grid.index(point) = Filled

	}

	return grid

}

func (g Grid) raw_index(point Point) *SiteState {
	return &g[point.Y][point.X]
}

func (g Grid) index(point Point) *SiteState {
	// radius := len(g) / 2
	// point.X = max(-radius, min(radius, point.X))
	// point.Y = max(-radius, min(radius, point.Y))
	// point.Y *= -1

	point.X = max(0, min(len(g), point.X))
	point.Y = max(0, min(len(g), point.Y))

	// real_point := add_points(point, real_mid_point(len(g)))

	return &g[point.Y][point.X]
}

func (v *Vector) roundToPoint() Point {
	return Point{X: round(v.x), Y: round(v.y)}
}

func (g Grid) vectorIndex(vector Vector) *SiteState {

	copied := vector
	return g.index(copied.roundToPoint())

}

func (g *Grid) is_valid_point(point Point) bool {

	radius := len(*g) / 2

	if point.X >= -radius && point.X <= radius && point.Y >= -radius && point.Y <= radius {
		return true
	} else {
		return false
	}
}

func clear_screen() {
	print("\u001b[2J")
}

func clear_line() {
	print("\u001b[2K")
	print("\r")
}

func random_step(r *rand.Rand) Point {

	value := r.Float64()

	if value < 0.25 {
		return Point{X: 1, Y: 0}
	} else if value < 0.5 {
		return Point{X: -1, Y: 0}
	} else if value < 0.75 {
		return Point{X: 0, Y: 1}
	} else {
		return Point{X: 0, Y: -1}
	}

}

func init_model(size int, _ float64, distance int) Model {

	// if size%2 == 0 {
	// 	panic("grid size must be odd you doofus")
	// }

	if distance <= 0 {
		panic("spawning distacne must be non-negative")
	}

	// grid_type := "normal"
	// grid_type := "heart"
	heart := false

	var grid Grid
	var nextgrid Grid
	if !heart {
		grid = gen_grid(size)
		*grid.index(mid_point()) = Filled

		nextgrid = gen_grid(size)
		*nextgrid.index(mid_point()) = Filled
	} else {

		heart_radius := 30.0
		grid = gen_heart_grid(size, heart_radius)
	}

	// walkers := make([]Walker, 1)
	// walkers[0] = Walker{
	// 	location: middle,
	// 	ttl:      tau,
	// }

	model := Model{
		grid:     grid,
		nextGrid: nextgrid,
		rootGrid: genIntGrid(size, ROOT_NONE),
		nextRoot: genIntGrid(size, ROOT_NONE),
		// walkers:  []TreeWalker{{location: vectorFromPoint(mid_point()), intensity: 100}},
		walkers:         []TreeWalker{},
		grids:           make([]Grid, 0, 100),
		size:            size,
		spawn:           Vector{50, 50},
		time:            0,
		nextRootID:      0,
		particlesInGrid: 0,
		freeParticles:   TOTAL_PARTICLE_RESOURCES,
	}

	for y := range size {
		for x := range size {
			if model.grid[y][x] > 0 {
				model.particlesInGrid += int(model.grid[y][x])
				model.rootGrid[y][x] = ROOT_MIXED
			}
		}
	}
	model.freeParticles = max(0, TOTAL_PARTICLE_RESOURCES-model.particlesInGrid)

	return model

}

func (m *Model) origin() Point {
	return Point{X: 0, Y: 0}
}

func (m *Model) countNeibors(point Point) int {
	neighbors := 0
	for _, step := range CARDINALS {
		new_point := add_points(point, step)
		if m.grid.is_valid_point(new_point) && *m.grid.index(new_point) > 0 {
			neighbors++
		}
	}

	return neighbors
}

func (m *Model) onPerimeter(point Point) bool {

	// radius := (m.size-1)/2 - 1
	// if point.X == radius || point.X == -radius || point.Y == radius || point.Y == -radius {
	// 	return true
	// } else {
	// 	return false
	// }

	if point.X <= 1 || point.X >= m.size-2 || point.Y <= 1 || point.Y >= m.size-2 {
		return true
	} else {
		return false
	}

}

func (m *Model) countOnRadius(radius int) int {
	count := 0

	if radius == 0 {
		return 1
	}

	for i := range radius*2 + 1 {
		i -= radius

		if *m.grid.index(Point{X: i, Y: -radius}) > 0 {
			count++
		}
		if *m.grid.index(Point{X: i, Y: radius}) > 0 {
			count++
		}
	}
	// dont wanna double count the corners
	for i := range radius*2 - 1 {
		i -= radius

		if *m.grid.index(Point{X: -radius, Y: i}) > 0 {
			count++
		}
		if *m.grid.index(Point{X: radius, Y: i}) > 0 {
			count++
		}
	}

	return count
}

func dotProduct(v1, v2 Vector) float64 {
	return v1.x*v2.x + v1.y*v2.y
}

func selectProbability(probs []float64, r *rand.Rand) int {

	sumProbs := 0.0

	for _, value := range probs {
		sumProbs += value
	}

	selection := r.Float64() * sumProbs

	runningProb := sumProbs

	i := len(probs) - 1

	for i > 0 {

		runningProb -= probs[i]
		if selection > runningProb {
			return i
		}
		i--
	}
	return 0
}

func weightedDirection(weight Vector) []float64 {

	probabilites := make([]float64, 4)

	for i, unit := range UNITS {
		dot := dotProduct(weight, unit)

		// probabilites[i] = math.Exp(dot)
		probabilites[i] = 1 + (dot / 2)
	}

	return probabilites
	// panic("ahhhhh")

}

func (m *Model) forceCAMPish(from Vector, rootID int) (Vector, bool) {
	center := from.roundToPoint()
	radius := FORCE_CAMP_RADIUS

	yMin := max(0, center.Y-radius)
	yMax := min(m.size-1, center.Y+radius)
	xMin := max(0, center.X-radius)
	xMax := min(m.size-1, center.X+radius)

	totalWeight := 0.0
	summedDirection := Vector{x: 0, y: 0}
	bestWeight := 0.0
	bestTarget := Point{}
	foundTarget := false

	for y := yMin; y <= yMax; y++ {
		for x := xMin; x <= xMax; x++ {
			trailStrength := m.grid[y][x]
			if trailStrength <= Empty {
				continue
			}
			owner := m.rootGrid[y][x]
			if owner == ROOT_NONE || owner == rootID {
				continue
			}

			ownershipFactor := 1.0
			if owner == ROOT_MIXED {
				ownershipFactor = FORCE_CAMP_MIXED_ROOT_WEIGHT
			}

			dx := float64(x - center.X)
			dy := float64(y - center.Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist == 0 || dist > float64(radius) {
				continue
			}

			radialWeight := (float64(radius) - dist) / float64(radius)
			if radialWeight <= 0 {
				continue
			}

			weight := float64(trailStrength) * radialWeight * ownershipFactor
			direction := Vector{x: dx, y: dy}
			direction.normalize()
			direction.scale(weight)
			summedDirection.add(direction)
			totalWeight += weight

			if !foundTarget || weight > bestWeight {
				bestWeight = weight
				bestTarget = Point{X: x, Y: y}
				foundTarget = true
			}
		}
	}

	if totalWeight == 0 || summedDirection.magnitude() == 0 || !foundTarget {
		return ZERO_VECTOR, false
	}

	commitment := summedDirection.magnitude() / totalWeight
	if commitment < FORCE_CAMP_MIN_COMMIT {
		return ZERO_VECTOR, false
	}

	campForce := Vector{x: float64(bestTarget.X - center.X), y: float64(bestTarget.Y - center.Y)}
	campForce.normalize()
	campForce.scale(FORCE_CAMP_PULL * FORCE_CAMP_COMMIT_FRACTION)

	return campForce, true
}

func (m *Model) treeTick(r *rand.Rand) bool {

	mouseTarget := subtract(vectorFromPoint(LIVE_MOUSE_POINT), m.spawn)

	for i, walker := range m.walkers {
		if walker.intensity < MIN_WALKER_INTENSITY {
			m.walkers[i].intensity = 0
			continue
		}

		var new_vec Vector
		var new_point Point

		velo := walker.velocity
		force := mouseTarget
		if LIVE_FORCE_ATTRACT {
			force = subtract(force, m.spawn)
			// change this if u want it to be like it was before
			// force = curvyForce(subtract(walker.location, m.spawn), mouseTarget)
			// force = attractionForce(walker.location, LIVE_MOUSE_TARGET)
		}
		force.normalize()
		velo.add(force)
		// velo.add(WEIGHT_VECTOR)

		totalTrail := 0.0
		for _, bird := range UNITS {
			totalTrail += float64(*m.grid.vectorIndex(add_vectors(walker.location, bird)))
		}

		if totalTrail > 0 {
			for i := range UNITS {

				new_vec = add_vectors(walker.location, UNITS[i])

				new_point = new_vec.roundToPoint()

				if *m.grid.index(new_point) > 0 {
					direction := UNITS[i]
					direction.scale(float64(*m.grid.index(new_point)) / totalTrail)
					velo.add(direction)
					// probs[i] *= SACRIFICE
				}
			}
		}

		if campForce, ok := m.forceCAMPish(walker.location, walker.rootID); ok && r.Float64() < FORCE_CAMP_COMMIT_FRACTION {
			velo = campForce
		}

		// new_vec = add_vectors(walker.location, UNITS[selection])

		velo.normalize()
		m.walkers[i].location.add(velo)
		m.walkers[i].velocity = velo

		quantized := m.walkers[i].location.roundToPoint()

		// *m.nextGrid.index(quantized) += 1
		// i think this is the next thing to work on
		// we need to find some model the walkers loosing intensity as they walk
		//

		if m.depositWithOverflow(quantized, velo, walker.rootID) {
			m.walkers[i].intensity -= 1
		}

		// conditions to reset walkers
		// if m.onPerimeter(quantized) || distance(m.walkers[i].location, LIVE_MOUSE_TARGET) < 2 {
		if m.onPerimeter(quantized) ||
			distance(m.walkers[i].location, vectorFromPoint(LIVE_MOUSE_POINT)) < 2 {
			// m.walkers = slices.Delete(m.walkers, i, i+1)
			m.walkers[i].intensity = 0
			continue
			// return true
		}

		// dont split if we dont have any food / intensity ig
		if r.Float64() < SPLIT && walker.intensity >= 2 {
			og := m.walkers[i]
			newVelo := og.velocity

			if rand.Float64() < 0.5 {
				newVelo.rotate(math.Pi / 2)
			} else {
				newVelo.rotate(-math.Pi / 2)
			}

			newVelo.scale(2)

			m.walkers = append(
				m.walkers,
				TreeWalker{
					location:  add_vectors(og.location, newVelo),
					intensity: og.intensity / 2,
					velocity:  newVelo,
					rootID:    og.rootID,
				},
			)

			m.walkers[i].intensity /= 2
		}

		// if *m.grid.index(walker) == Empty {
		// 	*m.grid.index(walker) = Filled

		// if m.onPerimeter(new_point) {
		// 	return true
		// } else {
		// 	return false
		// }
		// }

	}

	m.cullWeakWalkers()

	return false

	// panic("shouldnt reach this")
}

func (m *Model) run_trial(r *rand.Rand) Data {
	model := m

	for m.time < int(1000) {
		m.regenerateResourceCap()

		model.nextGrid = gen_grid(m.size)
		model.nextRoot = genIntGrid(m.size, ROOT_NONE)
		// fmt.Println(m.time)
		// end := model.tick(r)
		end := model.treeTick(r)

		for i := range m.size {
			for j := range m.size {
				model.grid[i][j] += model.nextGrid[i][j]
				incomingOwner := model.nextRoot[i][j]
				if incomingOwner == ROOT_NONE {
					continue
				}

				existingOwner := model.rootGrid[i][j]
				if existingOwner == ROOT_NONE {
					model.rootGrid[i][j] = incomingOwner
				} else if existingOwner != incomingOwner {
					model.rootGrid[i][j] = ROOT_MIXED
				}
			}
		}

		copied := make(Grid, m.size)
		for i := range copied {
			copied[i] = slices.Clone(m.grid[i])
		}
		// *copied.index(new_point) = Active

		m.grids = append(m.grids, copied)

		m.time++
		// end := model.different_tick(r)

		// fmt.Println("ticked me off")

		if end {
			break
		}
	}

	data := make(Data, 0, model.radius)
	// running_total := 1
	//
	// for r := 1; r < model.radius; r++ {
	// 	running_total += model.countOnRadius(r)
	// 	data = append(data, DataPoint{radius: r, filled: running_total})
	// }
	return data

}

func run_simulation() stats.Series {
	distance := 20
	num_points := 100.0

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	series := make(stats.Series, 0, int(num_points))

	for p := 0.01; p < 1.0; p += 0.01 {
		// p := p / num_points

		clear_line()
		fmt.Print("this much done: ", p*100, "%")

		model := init_model(GRID_SIZE, p, distance)

		data := model.run_trial(r)
		casted := data.toSeries()
		logged := logLog(casted)

		_, gradient, err := LinearRegression(logged)

		if err != nil {
			panic(err)
		}

		series = append(series, stats.Coordinate{X: p, Y: gradient})

	}

	// pretty_picture(model, "testing", 5)
	return series

}

type DataPoint struct {
	radius int
	filled int
}

type Data []DataPoint

type Arguments struct {
	file *string
	// operation *string
	chart  *string
	output *string
	live   *bool
}

func parse_args() Arguments {
	// args := os.Args[1:]
	// if len(args) >= 2 {
	// 	if args[0] == "--file" || args[0] == "-f" {
	// 		return Arguments{
	// 			file:    args[1],
	// 			is_file: true,
	// 		}
	// 	}
	// }
	//
	// return Arguments{
	// 	file:    "",
	// 	is_file: false,
	// }

	args := Arguments{
		file: flag.String("file", "", "path to data file"),
		// operation: flag.String("op", "", "operation to perform"),
		chart:  flag.String("chart", "", "type of chart to make"),
		output: flag.String("out", "", "prefix of output files"),
		live:   flag.Bool("live", false, "run interactive mode with mouse-controlled force"),
	}

	flag.Parse()

	return args
}

func (d *Data) toSeries() stats.Series {
	series := make([]stats.Coordinate, 0, len(*d))

	for _, point := range *d {
		series = append(
			series,
			stats.Coordinate{X: float64(point.radius), Y: float64(point.filled)},
		)
	}

	return series
}

func logLog(series stats.Series) stats.Series {

	logged := make(stats.Series, 0, len(series))

	for _, point := range series {
		logged = append(logged, stats.Coordinate{X: math.Log(point.X), Y: math.Log(point.Y)})
	}

	return logged
}

func LinearRegression(s stats.Series) (float64, float64, error) {

	if len(s) == 0 {
		return 0, 0, nil
	}

	// Placeholder for the math to be done
	var sum [5]float64

	// Loop over data keeping index in place
	i := 0
	for ; i < len(s); i++ {
		sum[0] += s[i].X
		sum[1] += s[i].Y
		sum[2] += s[i].X * s[i].X
		sum[3] += s[i].X * s[i].Y
		sum[4] += s[i].Y * s[i].Y
	}

	// Find gradient and intercept
	f := float64(i)
	gradient := (f*sum[3] - sum[0]*sum[1]) / (f*sum[2] - sum[0]*sum[0])
	intercept := (sum[1] / f) - (gradient * sum[0] / f)

	return intercept, gradient, nil
}

func renderProgressBar(series stats.Series) {
	if len(series) == 0 {
		fmt.Println("No data to display")
		return
	}

	maxY := series[0].Y
	for _, coord := range series {
		if coord.Y > maxY {
			maxY = coord.Y
		}
	}

	if maxY == 0 {
		maxY = 1
	}

	barWidth := 50
	fmt.Println()
	fmt.Println("Progress Chart (X = probability, Y = gradient):")
	fmt.Println(strings.Repeat("-", barWidth+20))

	for _, coord := range series {
		filledWidth := int((coord.Y / maxY) * float64(barWidth))
		if filledWidth < 0 {
			filledWidth = 0
		}
		if filledWidth > barWidth {
			filledWidth = barWidth
		}

		bar := strings.Repeat("#", filledWidth) + strings.Repeat("-", barWidth-filledWidth)
		fmt.Printf("p=%.2f | %s | %.4f\n", coord.X, bar, coord.Y)
	}

	fmt.Println(strings.Repeat("-", barWidth+20))
}

// right now all the globals arent actually here
// maybe they should be or maybe this should have a different name
func initGlobals() {

	// s, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansTFF))
	// if err != nil {
	// 	slog.Error(err)
	// 	panic(err)
	// }
	// arabicFaceSource = s

	// data, err := os.ReadFile("../assests/oat.png")
	//
	// if err != nil {
	// 	panic(err)
	// }
	//
	// img, _, err := image.Decode(bytes.NewReader(data))
	//
	// if err != nil {
	// 	log.Fatal(err)
	// }

	var err error
	OatImage, _, err = ebitenutil.NewImageFromFile("../assests/oat.png")
	if err != nil {
		panic(err)
	}
}

type LiveGame struct {
	model    Model
	theMap   *Map
	rng      *rand.Rand
	p        float64
	distance int
}

func newLiveGame(p float64, distance int) *LiveGame {

	initGlobals()

	return &LiveGame{
		model:    init_model(GRID_SIZE, p, distance),
		theMap:   defaultMap(),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		p:        p,
		distance: distance,
	}
}

func (g *LiveGame) reset() {
	g.model = init_model(GRID_SIZE, g.p, g.distance)
}

func (g *LiveGame) step() bool {
	g.model.regenerateResourceCap()
	g.model.nextGrid = gen_grid(g.model.size)
	g.model.nextRoot = genIntGrid(g.model.size, ROOT_NONE)
	end := g.model.treeTick(g.rng)

	for i := range g.model.size {
		for j := range g.model.size {
			g.model.grid[i][j] += g.model.nextGrid[i][j]
			incomingOwner := g.model.nextRoot[i][j]
			if incomingOwner == ROOT_NONE {
				continue
			}

			existingOwner := g.model.rootGrid[i][j]
			if existingOwner == ROOT_NONE {
				g.model.rootGrid[i][j] = incomingOwner
			} else if existingOwner != incomingOwner {
				g.model.rootGrid[i][j] = ROOT_MIXED
			}
		}
	}

	g.model.time++

	return end || g.model.time >= 1000
}

func mouseWeightVector(cursorX, cursorY, width, height int) Vector {
	v := Vector{
		x: float64(width/2 - cursorX),
		y: float64(height/2 - cursorY),
	}

	v.normalize()

	if v.magnitude() == 0 {
		return Vector{x: 0, y: 0}
	}

	return v
}

func mouseTargetVector(cursorX, cursorY, width, height int) Vector {
	if width <= 0 || height <= 0 {
		return Vector{x: 0, y: 0}
	}

	x := (float64(cursorX)/float64(width) - 0.5) * float64(GRID_SIZE-1)
	y := (0.5 - float64(cursorY)/float64(height)) * float64(GRID_SIZE-1)

	return Vector{x: x, y: y}
}

func mouseTargetPoint(cursorX, cursorY, width, height int) Point {
	return Point{X: cursorX * GRID_SIZE / width, Y: cursorY * GRID_SIZE / height}
}

// there should probably be a dedicated input handling function
func (g *LiveGame) Update() error {
	w, h := ebiten.WindowSize()
	x, y := ebiten.CursorPosition()

	LIVE_MOUSE_POINT = mouseTargetPoint(x, y, w, h)

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.model.clear()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if !g.model.spawnWalkerAtNearestPlacedParticle(LIVE_MOUSE_POINT) {
			g.model.spawnWalker()
		}
	}

	// fmt.Println(w, h, x, y)

	// LIVE_MOUSE_TARGET = mouseTargetVector(x, y, w, h)
	// fmt.Println(LIVE_MOUSE_POINT)
	// WEIGHT_VECTOR = vectorFromPoint(LIVE_MOUSE_POINT)
	//uhhh
	// WEIGHT_VECTOR.normalize()
	// WEIGHT_VECTOR = mouseWeightVector(x, y, w, h)

	// Take multiple simulation steps per frame to keep visible growth speed.
	for range 1 {
		if g.step() {
			// g.reset()
			break
		}
	}

	return nil
	// return errors.New("bruh")
}

func centeredTextOpts(theText string, scale float64, x, y float64) *text.DrawOptions {
	w, h := text.Measure(
		theText,
		fontFace,
		0,
	) // The left upper point is not x but x-w, since the text runs in the rigth-to-left direction.
	// fmt.Println(w, h)

	x, y = x-scale*w/2, y-scale*h/2
	fmt.Println(x, y)
	// x, y = 50, 50
	// vector.FillRect(screen, float32(x)-float32(w), float32(y), float32(w), float32(h), gray, false)
	op := &text.DrawOptions{}
	op.ColorScale.ScaleWithColor(color.White)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x), float64(y))
	return op

}

func (g *LiveGame) DrawStats(screen *ebiten.Image) {
	// const arabicText = "لمّا كان الاعتراف بالكرامة المتأصلة في جميع"
	const someText = "hello there"
	// f := &text.GoTextFace{
	// 	Source:    arabicFaceSource,
	// 	Direction: text.DirectionRightToLeft,
	// 	Size:      24,
	// 	Language:  language.Arabic,
	// }

	// textScale := 5.0

	op := centeredTextOpts(someText, 2, SCREEN_SIZE/2, 50)

	text.Draw(screen, someText, fontFace, op)

}

func (g *LiveGame) Draw(screen *ebiten.Image) {

	ebitenutil.DebugPrint(screen, "Click to spawn\nC to clear")
	// fmt.Println(screen)

	// opts := &ebiten.DrawImageOptions{}
	// opts.GeoM.Scale(0.1, 0.1)
	// opts.GeoM.Translate(SCREEN_SIZE/2, SCREEN_SIZE/2)
	// screen.DrawImage(OatImage, opts)
	//
	oatScale := 0.2

	// this should be made a bg image istead of making it every frame
	for _, food := range g.theMap.Foods {
		size := float64(OatImage.Bounds().Size().X)
		size *= oatScale

		opts := &ebiten.DrawImageOptions{}
		// fmt.Println("ycoord; ", float64(food.Position.X*SCREEN_SIZE)/SIZE)
		opts.GeoM.Scale(oatScale, oatScale)

		opts.GeoM.Translate(
			float64(food.Position.X)*SCALE,
			float64(food.Position.Y)*SCALE,
		)
		opts.GeoM.Translate(-size/2, -size/2)
		// opts.GeoM.Translate(
		// 	float64(food.Position.X*SCREEN_SIZE)/SIZE/2,
		// 	float64(food.Position.Y*SCREEN_SIZE)/SIZE,
		// )
		// opts.GeoM.Translate(SCREEN_SIZE/SIZE, SCREEN_SIZE/SIZE)
		screen.DrawImage(OatImage, opts)
	}

	copyGrid2Image(g.model.grid, screen)

	g.DrawStats(screen)
	// screen.WritePixels(frame.Pix)
}

func (g *LiveGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := SCREEN_SIZE / GRID_SIZE
	side := GRID_SIZE * scale
	return side, side
}

func runLive() {
	ebiten.SetTPS(TPS)

	game := newLiveGame(0.1, 20)
	LIVE_FORCE_ATTRACT = true

	// SCALE = float64(SCREEN_SIZE) / GRID_SIZE
	side := int(GRID_SIZE * SCALE)

	ebiten.SetWindowSize(side, side)
	ebiten.SetWindowTitle("Slime Mold - Mouse Force")

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func main() {

	// u kind of dont need this slog.Warn cuz the compiler gets mad
	// if SCALE isn't a whole number
	if float64(int(SCALE)) != SCALE {
		slog.Warn(
			"Screen Size is not divisible by Grid Size\n",
			SCREEN_SIZE,
			" % ",
			GRID_SIZE,
			" != 0",
		)
	}

	WEIGHT_VECTOR.normalize()
	// testing()
	// return

	// var data stats.Series

	// args := parse_args()
	// time.Sleep(time.Second * 4)
	runLive()

}

func calc_color(percent float64) color.NRGBA {
	RStart, REnd := 1.0, 255.0

	// WOW !!! great code
	return color.NRGBA{
		R: uint8(RStart + (REnd-RStart)*percent),
		A: 255,
		// A: uint8(int(color_start.A) + round(float64(color_end.A-color_start.A)*percent)),
	}
}

const FLOW_LEVELS = 10

var FLOW_PALETTE = [FLOW_LEVELS]color.NRGBA{
	{R: 30, G: 42, B: 56, A: 255},
	{R: 54, G: 92, B: 141, A: 255},
	{R: 57, G: 123, B: 152, A: 255},
	{R: 71, G: 152, B: 110, A: 255},
	{R: 126, G: 177, B: 68, A: 255},
	{R: 184, G: 188, B: 72, A: 255},
	{R: 217, G: 163, B: 62, A: 255},
	{R: 229, G: 117, B: 53, A: 255},
	{R: 216, G: 68, B: 58, A: 255},
	{R: 181, G: 36, B: 48, A: 255},
}

func quantizeFlowBucket(value float64, levels int) int {
	if levels <= 1 || value <= 0 {
		return 0
	}

	bucket := int(math.Round(value)) - 1
	return max(0, min(levels-1, bucket))
}

func copyGrid2Image(grid Grid, image *ebiten.Image) {

	scale := int(SCALE)

	*grid.index(mid_point()) = Origin

	// cropped := model.size - model.distance*2

	// for y, row := range grid[model.distance : model.size-model.distance] {
	// 	for x, value := range row[model.distance : model.size-model.distance] {
	for y, row := range grid {
		for x, value := range row {

			var color color.Color
			if value < 0 {
				color = StateColor[value]
			} else if value > 0 {
				flowBucket := quantizeFlowBucket(float64(value), FLOW_LEVELS)
				color = FLOW_PALETTE[flowBucket]
			} else {
				continue
			}

			// image.WritePixels()
			for i := range scale {
				for j := range scale {
					image.Set(x*scale+j, y*scale+i, color)
				}
			}
		}
	}

}

func grid2png(grid Grid) *image.NRGBA {

	size := len(grid)
	screen_size := 960

	scale := screen_size / size

	*grid.index(mid_point()) = Origin

	// cropped := model.size - model.distance*2

	img := image.NewNRGBA(image.Rect(0, 0, size*scale, size*scale))

	// for y, row := range grid[model.distance : model.size-model.distance] {
	// 	for x, value := range row[model.distance : model.size-model.distance] {
	for y, row := range grid {
		for x, value := range row {

			var color color.Color
			if value <= 0 {
				color = StateColor[value]
			} else {
				flowBucket := quantizeFlowBucket(float64(value), FLOW_LEVELS)
				color = FLOW_PALETTE[flowBucket]
			}

			for i := range scale {
				for j := range scale {
					img.Set(x*scale+j, y*scale+i, color)
				}
			}
		}
	}

	return img
}

func Pointer[T any](t T) *T {
	return &t
}

/*
	func find_max(data []opts.Chart3DData) float32 {
		greatest := float64(0)
		return float32(greatest)
	}

	func make_3d_chart(data []opts.Chart3DData) {
		surface := charts.NewSurface3D()
	}
*/

// func cast_to_float(input []any) ([]float64, error) {
// 	output := make([]float64, len(input))
