package model

import (
	"math"
	"math/rand"
	"time"
)

const SQUARE_PERCENT = 0.5

const SENSOR_ANGLE = math.Pi / 4
const ROTATION_ANGLE = math.Pi / 4
const SENSOR_DISTANCE = 9
const CHEMO_DEPOSIT = 5.0
const CHEMO_DECAY = 0.2

const R2D2 = math.Sqrt2 / 2

var CARDINALS = []Point{
	{X: 0, Y: 1},
	{X: R2D2, Y: R2D2},
	{X: 1, Y: 0},
	{X: R2D2, Y: -R2D2},
	{X: 0, Y: -1},
	{X: -R2D2, Y: -R2D2},
	{X: -1, Y: -0},
	{X: -R2D2, Y: R2D2},
}

func round(x float64) int {
	return int(math.Round(x))
}

type Particle struct {
	position  Point
	direction float64
}

func randomDirection(r *rand.Rand) float64 {
	return float64(r.Float64()) * 2 * math.Pi
	// return float64(r.Intn(2)) * math.Pi
}

type Simulation struct {
	DataLayer  [][]*Particle
	Particles  map[int]*Particle
	TrailLayer [][]float64
	otherTrail [][]float64

	Size int
	r    *rand.Rand
}

func NewSimulation(size int) *Simulation {

	data := make([][]*Particle, size)
	particles := make(map[int]*Particle)
	trail := make([][]float64, size)
	otherTrail := make([][]float64, size)

	for i := range size {
		data[i] = make([]*Particle, size)
		trail[i] = make([]float64, size)
		otherTrail[i] = make([]float64, size)
	}

	simulation := Simulation{
		DataLayer:  data,
		Particles:  particles,
		TrailLayer: trail,
		otherTrail: otherTrail,
		Size:       size,
		r:          rand.New(rand.NewSource(time.Now().UnixMilli())),
	}

	center := size / 2
	start := center - int(float64(size)*SQUARE_PERCENT/2)
	stop := center + int(float64(size)*SQUARE_PERCENT/2)

	length := stop - start

	// r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	right := 0.0
	down := math.Pi / 2
	left := math.Pi
	up := 3 * math.Pi / 2

	for i := 0; i < length; i += 2 {
		simulation.AddParticle(start+i, start, right)
		simulation.AddParticle(start, start+i, up)
		simulation.AddParticle(stop-i, stop, left)
		simulation.AddParticle(stop, stop-i, down)
	}

	// simulation.AddParticle(50, 50, randomDirection(r))

	return &simulation
}

// func (s *Simulation) modPoint(point Point) Point {
//
// }

// the *real* modulo operator
func remEuclid(a float64, b float64) float64 {

	for a < 0 {
		a += b
	}

	ans := math.Mod(a, b)

	return ans
}

func (s *Simulation) addPoints(p1, p2 Point) Point {
	newPoint := AddPoints(p1, p2)

	newPoint.X = remEuclid(newPoint.X, float64(s.Size))
	newPoint.Y = remEuclid(newPoint.Y, float64(s.Size))

	return newPoint
}

func (s *Simulation) indexData(point Point) *Particle {
	return s.DataLayer[int(point.Y)][int(point.X)]
}

func (s *Simulation) setData(point Point, particle *Particle) {
	s.DataLayer[int(point.Y)][int(point.X)] = particle
}

func (s *Simulation) indexTrail(point Point) *float64 {
	return &s.TrailLayer[int(point.Y)][int(point.X)]
}

func (s *Simulation) AddParticle(x, y int, direction float64) {
	if s.DataLayer[y][x] != nil {
		// should prolly get some errors down the line
		return
	}

	particle := Particle{
		position:  Point{X: float64(x), Y: float64(y)},
		direction: direction,
	}

	s.DataLayer[y][x] = &particle

	s.Particles[len(s.Particles)] = &particle
}

func (s *Simulation) DepositAttractant(point Point) {
	s.TrailLayer[int(point.Y)][int(point.X)] += CHEMO_DEPOSIT
}

func (s *Simulation) AdvanceParticles() {
	for _, particle := range s.Particles {
		directionVector := PointFromTheta(particle.direction)
		newPos := s.addPoints(particle.position, directionVector)

		// fmt.Println(particle.position, "+", directionVector, "=")
		// fmt.Println(newPos)

		if s.indexData(newPos) == nil {
			s.setData(newPos, particle)
			s.setData(particle.position, nil)

			s.DepositAttractant(newPos)

			particle.position = newPos
		} else {
			particle.direction = randomDirection(s.r)
		}
	}
}

func (s *Simulation) SenseParticles() {
	for _, particle := range s.Particles {
		leftAngle := particle.direction - SENSOR_ANGLE
		rightAngle := particle.direction + SENSOR_ANGLE

		leftDirection := PointFromTheta(leftAngle)
		centerDirection := PointFromTheta(particle.direction)
		rightDirection := PointFromTheta(rightAngle)

		leftDirection.Scale(SENSOR_DISTANCE)
		centerDirection.Scale(SENSOR_DISTANCE)
		rightDirection.Scale(SENSOR_DISTANCE)

		leftSensor := s.addPoints(particle.position, leftDirection)
		centerSensor := s.addPoints(particle.position, centerDirection)
		rightSensor := s.addPoints(particle.position, rightDirection)

		left := *s.indexTrail(leftSensor)
		center := *s.indexTrail(centerSensor)
		right := *s.indexTrail(rightSensor)

		if (center > left) && (center > right) {

		} else if (center < left) && (center < right) {
			// apparently we r just supposed to pick randomly here
			// but it might make more sense to go with the greater one idunno
			if s.r.Intn(2) == 0 {
				particle.direction = leftAngle
			} else {
				particle.direction = rightAngle
			}

		} else if left > right {
			particle.direction = leftAngle
		} else if right > left {
			particle.direction = rightAngle
		}
	}
}

// 3x3 mean (hah get it) kernel
func (s *Simulation) AngryColonel(point Point) float64 {
	mean := *s.indexTrail(point)

	for _, cardinal := range CARDINALS {
		newPoint := s.addPoints(point, cardinal)
		mean += *s.indexTrail(newPoint)
	}

	return mean / 9
}

func (s *Simulation) DiffuseDecay() {
	// trail := make([][]float64, s.Size)

	// doing this linearly may pose some issues but we shall see
	for y := range s.Size {
		// trail[y] = make([]float64, s.Size)
		for x := range s.Size {
			s.otherTrail[y][x] = s.AngryColonel(Point{X: float64(x), Y: float64(y)})
		}
	}

	s.TrailLayer, s.otherTrail = s.otherTrail, s.TrailLayer

	for y := range s.Size {
		for x := range s.Size {
			if s.TrailLayer[y][x] >= 1e-3 {
				s.TrailLayer[y][x] *= CHEMO_DECAY
			} else {
				s.TrailLayer[y][x] = 0
			}
		}
	}
}

func (s *Simulation) Tick() {
	s.AdvanceParticles()

	s.SenseParticles()

	s.DiffuseDecay()
}

// func (s *Simulation) AddParticleLine(x1, y1, x2, y2 int) {
//
// }
