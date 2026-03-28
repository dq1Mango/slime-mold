package model

type Transform struct {
	scaleX, scaleY, shiftX, shiftY float64
}

func NewTransform(scaleX, scaleY, shiftX, shiftY float64) Transform {
	return Transform{scaleX: scaleX, scaleY: scaleY, shiftX: shiftX, shiftY: shiftY}
}

func (t *Transform) Forwards(point Point) Point {
	return Point{X: point.X*t.scaleX + t.shiftX, Y: point.Y*t.scaleY + t.shiftY}
}

func (t *Transform) Backwards(point Point) Point {
	return Point{X: (point.X - t.shiftX) / t.scaleX, Y: (point.Y - t.shiftY) / t.scaleY}
}
