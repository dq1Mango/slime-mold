package components

import (
	"fmt"
	"math"
	"syscall/js"
	"time"

	"github.com/dq1Mango/mold-slime/actions"
	"github.com/dq1Mango/mold-slime/model"
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

const CANVAS_SIZE = 100
const TPS = 20
const DELAY = 1.0 / TPS * 1000

type Canvas struct {
	vecty.Core
	ctx       js.Value
	Transform model.Transform

	Id   string
	Size int

	Actions    chan actions.Action
	Simulation *model.Simulation

	running bool
	drawing bool
	x, y    float64
}

func NewCanvas(id string) *Canvas {
	return &Canvas{
		Id:         id,
		Size:       CANVAS_SIZE,
		Actions:    make(chan actions.Action, 10),
		Simulation: model.NewSimulation(100),
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

	// tick := make(chan time.Time)

	go func() {
		// delay := int(1.0 / TPS * 1000)
		for {

			if c.running {
				if len(c.Actions) < 1 {
					c.Actions <- &actions.Tick{}
				} else {
					fmt.Println("actions busy")
				}
			}
			time.Sleep(time.Duration(math.Round(DELAY)) * time.Millisecond)
		}
	}()

	// rAF owns all canvas writes
	var rafCallback js.Func
	rafCallback = js.FuncOf(func(this js.Value, args []js.Value) any {
		// drawFrame(simState.Snapshot()) // read state, draw to canvas
		if c.running {
			if len(c.Actions) < 1 {
				c.Actions <- &actions.Draw{}
			} else {
				fmt.Println("actions busy")
			}
		}

		js.Global().Call("requestAnimationFrame", rafCallback)
		return nil
	})
	js.Global().Call("requestAnimationFrame", rafCallback)

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
	// c.ctx.Call("clearRect", 0, 0, c.Size, c.Size)
	c.ctx.Set("fillStyle", "black")
	c.ctx.Call("fillRect", 0, 0, c.Size, c.Size)
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
func round(x float64) int {
	return int(math.Round(x))
}

func greyScale(x float64) string {
	value := round(x / model.CHEMO_DEPOSIT * 255 * 10)

	value = min(value, 255)

	return fmt.Sprintf("rgba(%d,%d,%d,1)", value, value, value)
}

func (c *Canvas) Draw() {
	c.Clear()

	for y := range c.Size {
		for x := range c.Size {
			value := c.Simulation.TrailLayer[y][x]
			if value == 0 {
				continue
			}

			c.ctx.Set("fillStyle", greyScale(value))
			c.ctx.Call("fillRect", x, y, 1, 1)
		}
	}
}

func (c *Canvas) Tick() {
	fmt.Println("started tick")
	c.Simulation.Tick()
	fmt.Println("ended tick")
}

// Handle actions sent of canvas.Actions asychnronously
func (c *Canvas) handleActions() {
	for {
		action := <-c.Actions

		switch a := action.(type) {

		case *actions.Draw:
			c.Draw()
			fmt.Println("got a redraw request...")

		case *actions.Tick:
			c.Tick()

		case *actions.MouseDown:
			c.drawing = true

		case *actions.MouseMove:
			c.drawLine(a.Pos)

		case *actions.MouseUp:
			c.drawLine(a.Pos)
			c.drawing = false

		case *actions.Start:
			c.running = true

		case *actions.Pause:
			c.running = false

		default:
			fmt.Printf("Unknown action of type: %T\n", action)
		}
	}
}
