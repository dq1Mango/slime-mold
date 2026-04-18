package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/montanaflynn/stats"
	// "slices"
)

const END_RATIO = 0.01
const SIZE = 101

const SACRIFICE = 0.01
const SELFISH = 0.9

const SPLIT = 0.1

const (
	FPS         = 20.0
	SCREEN_SIZE = 960
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
	radius_fraction := v.magnitude() / (SIZE / 2)
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

const MAX_DISTANCE = 100

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

	radius_fraction := relativePos.magnitude() / (SIZE / 2)

	output := relativePos

	output.rotate(delat_theta * radius_fraction)

	output.normalize()

	return output
}

func mid_point() Point {
	mid := (SIZE - 1) / 2

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
}

type Model struct {
	grid     Grid
	nextGrid Grid
	walkers  []TreeWalker
	grids    []Grid
	size     int
	spawn    Vector
	// p        float64
	// people   int
	// infected int
	radius int
	time   int
	// distance int
}

// func remove[T any](slice []T, s int) []T {
// 	return slices.Delete(slice, s, s+1)
// }

// // this isnt dumb at all
// type Walkers []Walker
//
// func (w Walkers) remove(i int) {
// 	w[i] = w[len(w)-1]
// 	w = w[:len(w)-1]
// }

func gen_grid(size int) Grid {

	if size%2 == 0 {
		panic("grid size must be odd you doofus")
	}

	grid := make(Grid, size)

	for row := range grid {
		grid[row] = make([]SiteState, size)
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

	if size%2 == 0 {
		panic("grid size must be odd you doofus")
	}

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
		walkers:  []TreeWalker{{location: vectorFromPoint(mid_point()), intensity: 100}},
		grids:    make([]Grid, 0, 100),
		size:     size,
		spawn:    Vector{50, 50},
		time:     0,
	}

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

	if point.X == 1 || point.X == m.size-2 || point.Y == 1 || point.Y == m.size-2 {
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

func (m *Model) treeTick(r *rand.Rand) bool {

	mouseTarget := subtract(vectorFromPoint(LIVE_MOUSE_POINT), m.spawn)

	for i, walker := range m.walkers {
		var new_vec Vector
		var new_point Point
		// time.Sleep(1 * time.Millisecond)

		// start := time.Now()

		// step := random_step(r)

		// index := r.Int63n(4)
		// if index == 3 {
		// 	continue
		// }
		// direction := CARDINALS[index]

		velo := walker.velocity
		force := mouseTarget
		force.normalize()
		if LIVE_FORCE_ATTRACT {

			// change this if u want it to be like it was before
			force = curvyForce(subtract(walker.location, m.spawn), mouseTarget)
			// force = attractionForce(walker.location, LIVE_MOUSE_TARGET)
		}
		velo.add(force)
		// velo.add(WEIGHT_VECTOR)

		// probs2 := weightedDirection(WEIGHT_VECTOR)

		// for i := range probs2 {
		//
		// 	probs[i] += probs2[i] * 2
		// }

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

		// probs[(last_move+2)/4] = 0.0

		// selection := selectProbability(probs, r)

		// last_move = selection

		// new_vec = add_vectors(walker.location, UNITS[selection])

		velo.normalize()
		m.walkers[i].location.add(velo)
		m.walkers[i].velocity = velo

		quantized := m.walkers[i].location.roundToPoint()
		*m.nextGrid.index(quantized) += 1

		// conditions to reset walkers
		// if m.onPerimeter(quantized) || distance(m.walkers[i].location, LIVE_MOUSE_TARGET) < 2 {
		if m.onPerimeter(quantized) ||
			distance(m.walkers[i].location, vectorFromPoint(LIVE_MOUSE_POINT)) < 2 {
			return true
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

	return false

	// panic("shouldnt reach this")
}

func (m *Model) run_trial(r *rand.Rand) Data {
	model := m

	for m.time < int(1000) {

		model.nextGrid = gen_grid(m.size)
		// fmt.Println(m.time)
		// end := model.tick(r)
		end := model.treeTick(r)

		for i := range m.size {
			for j := range m.size {
				model.grid[i][j] += model.nextGrid[i][j]
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

		model := init_model(SIZE, p, distance)

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
		model:    init_model(SIZE, p, distance),
		theMap:   defualtMap(),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		p:        p,
		distance: distance,
	}
}

func (g *LiveGame) reset() {
	g.model = init_model(SIZE, g.p, g.distance)
}

func (g *LiveGame) step() bool {
	g.model.nextGrid = gen_grid(g.model.size)
	end := g.model.treeTick(g.rng)

	for i := range g.model.size {
		for j := range g.model.size {
			g.model.grid[i][j] += g.model.nextGrid[i][j]
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

	x := (float64(cursorX)/float64(width) - 0.5) * float64(SIZE-1)
	y := (0.5 - float64(cursorY)/float64(height)) * float64(SIZE-1)

	return Vector{x: x, y: y}
}

func mouseTargetPoint(cursorX, cursorY, width, height int) Point {
	return Point{X: cursorX * SIZE / width, Y: cursorY * SIZE / height}
}

func (g *LiveGame) Update() error {

	w, h := ebiten.WindowSize()
	x, y := ebiten.CursorPosition()

	// LIVE_MOUSE_TARGET = mouseTargetVector(x, y, w, h)
	LIVE_MOUSE_POINT = mouseTargetPoint(x, y, w, h)
	fmt.Println(LIVE_MOUSE_POINT)
	// WEIGHT_VECTOR = vectorFromPoint(LIVE_MOUSE_POINT)
	//uhhh
	// WEIGHT_VECTOR.normalize()
	// WEIGHT_VECTOR = mouseWeightVector(x, y, w, h)

	// Take multiple simulation steps per frame to keep visible growth speed.
	for range 1 {
		if g.step() {
			g.reset()
			break
		}
	}

	return nil
	// return errors.New("bruh")
}

func (g *LiveGame) Draw(screen *ebiten.Image) {

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(0.1, 0.1)
	screen.DrawImage(OatImage, opts)

	// this should be made a bg image istead of making it every frame
	// for _, food := range g.theMap.Foods {
	// 	opts := &ebiten.DrawImageOptions{}
	// 	opts.GeoM.Translate(
	// 		float64(food.Position.X*SCREEN_SIZE)/SIZE/2,
	// 		float64(food.Position.Y*SCREEN_SIZE)/SIZE/2,
	// 	)
	// 	// opts.GeoM.Translate(SCREEN_SIZE/SIZE, SCREEN_SIZE/SIZE)
	// 	screen.DrawImage(OatImage, opts)
	// }

	copyGrid2Image(g.model.grid, screen)
	// screen.WritePixels(frame.Pix)
}

func (g *LiveGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := SCREEN_SIZE / SIZE
	side := SIZE * scale
	return side, side
}

func runLive() {
	ebiten.SetTPS(TPS)

	game := newLiveGame(0.1, 20)
	LIVE_FORCE_ATTRACT = true

	scale := SCREEN_SIZE / SIZE
	side := SIZE * scale

	ebiten.SetWindowSize(side, side)
	ebiten.SetWindowTitle("Slime Mold - Mouse Force")

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func main() {

	WEIGHT_VECTOR.normalize()
	// testing()
	// return

	// var data stats.Series

	// args := parse_args()
	// time.Sleep(time.Second * 4)
	runLive()

	// outputName := *args.output
	// if outputName == "" {
	// 	outputName = "testing"
	// 	fmt.Println("no -out provided, defaulting to:", outputName)
	// }
	//
	// if *args.chart == "progress" {
	// 	data = run_simulation()
	// 	renderProgressBar(data)
	// 	return
	// }
	//
	// if *args.live {
	// 	runLive()
	// 	return
	// }

	// one_trial(outputName)

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

func copyGrid2Image(grid Grid, image *ebiten.Image) {
	size := len(grid)
	screen_size := 960

	scale := screen_size / size

	largest := 0.0
	for _, row := range grid {
		for _, value := range row {
			if float64(value) > largest {
				largest = float64(value)
			}
		}
	}

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
				color = calc_color(float64(value) / largest)
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

	largest := 0.0
	for _, row := range grid {
		for _, value := range row {
			if float64(value) > largest {
				largest = float64(value)
			}
		}
	}

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
				color = calc_color(float64(value) / largest)
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
