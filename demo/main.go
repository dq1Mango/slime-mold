package main

import (
	"doest/matter/slime-mold/actions"
	"doest/matter/slime-mold/components"
	"fmt"

	// "os"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	// "github.com/hexops/vecty/event"
	// "github.com/hexops/vecty/prop"
	// "github.com/hexops/vecty/style"
)

func main() {
	fmt.Println("Hello World!")
	// testGraphing()

	vecty.SetTitle("Graph Theory")
	vecty.AddStylesheet("style.css")
	vecty.AddStylesheet("colors.css")
	vecty.RenderBody(NewPageView())
}

func NewPageView() *PageView {
	canvas := components.NewCanvas("main-canvas")

	return &PageView{
		canvas: canvas,
	}
}

// PageView is our main page component.
type PageView struct {
	vecty.Core

	canvas *components.Canvas
	// algorithm *components.AlgorithmWalk
}

// Render implements the vecty.Component interface.
func (p *PageView) Render() vecty.ComponentOrHTML {
	canvas := p.canvas

	// graphStats := components.NewGraphStats(graph)

	// popup := components.NewPopup()

	// fmt.Println(popup)

	return elem.Body(

		elem.Div(
			elem.Header(
				elem.Heading1(vecty.Text("Mold Slime Simulation")),
				elem.Heading4(vecty.Text("This is a subtitle I'm sure I wont forget about")),
			),
		),

		// &popup,

		elem.Div(
			vecty.Markup(
				vecty.Class("content-box"),
			),

			elem.Div(
				vecty.Markup(
					vecty.Class("button-box"),
				),

				&components.Button{
					Text: "Reset",
					OnClick: func(*vecty.Event) {
						canvas.Actions <- &actions.Reset{}
					},
				},
				&components.Button{
					Text: "Start",
					OnClick: func(*vecty.Event) {
						canvas.Actions <- &actions.Start{}
					},
				},
				&components.Button{
					Text: "Pause",
					OnClick: func(*vecty.Event) {
						canvas.Actions <- &actions.Pause{}
					},
				},
			),
			canvas,
		),
	)
}

// func (p *PageView) Mount() {
// 	// p.theme.SetTheme("catppuccin")
//
// 	// model.GreenFlag = true
//
// 	go func() {
// 		for {
// 			select {
// 			case newTheme := <-model.ThemChan:
// 				p.theme.SetTheme(newTheme)
// 				p.graph.Actions <- &actions.Draw{}
// 			}
// 		}
// 	}()
// }
