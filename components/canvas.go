package components

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/dq1Mango/mold-slime/actions"
	"github.com/dq1Mango/mold-slime/model"
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

const CANVAS_SIZE = 100

type Canvas struct {
	vecty.Core
	ctx       js.Value
	Transform model.Transform

	Id   string
	Size int

	Actions chan actions.Action

	drawing bool
	x, y    float64
}

func NewCanvas(id string) *Canvas {
	return &Canvas{
		Id:      id,
		Size:    CANVAS_SIZE,
		Actions: make(chan actions.Action, 10),
	}
}

func (c *Canvas) Render() vecty.ComponentOrHTML {
	return elem.Div(

		elem.Div(

			vecty.Markup(vecty.Class("canvas-wrapper"), prop.ID("canvas-wrapper")),
			elem.Canvas(
				vecty.Markup(
					vecty.Class("canvas"),
					prop.ID(c.Id),

					event.MouseMove(func(e *vecty.Event) {

						// mouse position updates aren't super important,
						// and with WASM we only have one os thread,
						// so if we are busy in any way just dont add more fuel to the fire
						if len(c.Actions) < 2 {
							point := c.PointFromMouseEvent(e)
							// non blocking send just in case
							select {
							case c.Actions <- &actions.MouseMove{
								Pos: point,
							}:
							default:
								fmt.Println("action buffer busy...")
							}
						}
					}),

					event.MouseDown(func(e *vecty.Event) {

						// shift := e.Get("shiftKey").Bool()
						point := c.PointFromMouseEvent(e)

						c.Actions <- &actions.MouseDown{Pos: point}
					}),

					event.MouseUp(func(e *vecty.Event) {

						// shift := e.Get("shiftKey").Bool()
						point := c.PointFromMouseEvent(e)

						c.Actions <- &actions.MouseUp{Pos: point}
					}),

					// these closures are rly nice i have to say
				),
			),
		),
	)
}

func (c *Canvas) SetCanvasTransform() {

	dpr := js.Global().Get("devicePixelRatio").Float()
	canvas := c.ctx.Get("canvas")
	rect := canvas.Call("getBoundingClientRect")

	width := rect.Get("width").Float()
	height := rect.Get("width").Float()

	if width != height {
		fmt.Printf("width and height of canvas: %s not equal\n", c.Id)
	}

	// fmt.Printf("width: %f, height: %f\n", width, height)

	canvas.Set("width", width*dpr)
	canvas.Set("height", height*dpr)

	wrapper := js.Global().Get("document").Call("getElementById", "canvas-wrapper")

	wrapper_height := wrapper.Get("clientHeight").Float()
	wrapper_width := wrapper.Get("clientWidth").Float()

	limiter := math.Min(wrapper_height, wrapper_width)

	// wrapper.Set("clientWidth", limiter)
	// wrapper.Set("clientHeight", limiter)
	wrapper.Get("style").Set("width", fmt.Sprintf("%fpx", limiter))
	wrapper.Get("style").Set("height", fmt.Sprintf("%fpx", limiter))

	// canvas.Get("style").Set("width", fmt.Sprintf("%fpx", width))
	// canvas.Get("style").Set("height", fmt.Sprintf("%fpx", height))

	scale := dpr * math.Min(width, height) / (float64(c.Size))

	c.ctx.Call("setTransform", scale, 0, 0, scale, 0, 0)
	// shift := dpr * width / 2

	// g.ctx.Call("setTransform", scale, 0, 0, -scale, shift, shift)

	// g.Transform = model.NewTransform(scale, -scale, shift, shift)

	c.Actions <- &actions.Draw{}
}

// this is called by vecty when the element is FIRST inserted into the DOM
func (c *Canvas) Mount() {
	canvas := js.Global().Get("document").Call("getElementById", c.Id)
	wrapper := js.Global().Get("document").Call("getElementById", "canvas-wrapper")
	c.ctx = canvas.Call("getContext", "2d")

	resizeFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		c.SetCanvasTransform()
		return nil
	})

	observer := js.Global().Get("ResizeObserver").New(resizeFunc)
	observer.Call("observe", wrapper)
	// c.observer = observer

	c.SetCanvasTransform()

	// safe to draw here, DOM is ready
	// c.ctx.Set("fillStyle", model.CurrentPalette.Red)
	// c.ctx.Call("fillRect", 0, 0, 50, 25)

	go func() {
		c.handleActions()
	}()
}

func (c *Canvas) PointFromMouseEvent(e *vecty.Event) model.Point {
	point := model.Point{
		X: e.Get("offsetX").Float(),
		Y: e.Get("offsetY").Float(),
	}
	point.Scale(2)
	point = c.Transform.Backwards(point)
	return point

}

func (c *Canvas) Clear() {
	c.ctx.Call("clearRect", 0, 0, c.Size, c.Size)
}

func (c *Canvas) drawLine(point model.Point) {

	ctx := c.ctx

	ctx.Call("beginPath")
	ctx.Set("strokeStyle", "black")
	ctx.Set("lineWidth", 1)

	ctx.Call("moveTo", c.x, c.y)
	ctx.Call("lineTo", point.X, point.Y)
	ctx.Call("stroke")
	ctx.Call("closePath")

}

// Handle actions sent of canvas.Actions asychnronously
func (c *Canvas) handleActions() {
	for {
		action := <-c.Actions

		switch a := action.(type) {

		case *actions.Draw:
			fmt.Println("got a redraw request...")

		case *actions.MouseDown:
			c.drawing = true

		case *actions.MouseMove:
			c.drawLine(a.Pos)

		case *actions.MouseUp:
			c.drawLine(a.Pos)
			c.drawing = false

		default:
			fmt.Printf("Unknown action of type: %T\n", action)
		}
	}
}
