package model

import "math"

// import "math"

// represnet an X,Y coordinate pair
// currently (0,0) is top left, but subject to change
type Point struct {
	X float64
	Y float64
}

func PointFromTheta(theta float64) Point {
	return Point{X: math.Cos(theta), Y: math.Sin(theta)}
}

//
// func (p *Point) Magnitude() float64 {
// 	return math.Sqrt(p.X*p.X + p.Y*p.Y)
// }

func (p *Point) Scale(r float64) {
	p.X *= r
	p.Y *= r
}

func AddPoints(p1, p2 Point) Point {
	return Point{p1.X + p2.X, p1.Y + p2.Y}
}

func SubtractPoint(p1, p2 Point) Point {
	return Point{p1.X - p2.X, p1.Y - p2.Y}
}

// func (p *Point) Distance(point Point) float64 {
// 	distanceVector := SubtractPoint(*p, point)
//
// 	return distanceVector.Magnitude()
// }
