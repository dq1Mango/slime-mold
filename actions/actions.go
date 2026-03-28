package actions

import "github.com/dq1Mango/mold-slime/model"

type Action any

type Draw struct{}

type Start struct{}
type Pause struct{}
type Reset struct{}

type MouseMove struct {
	Pos model.Point
}

type MouseDown struct {
	Pos model.Point
}

type MouseUp struct {
	Pos model.Point
}

// type Action struct {
// 	Id   int
// 	Data *any
// }
//
// const (
// 	Redraw int = iota
// )
