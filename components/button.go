package components

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
)

type Button struct {
	vecty.Core
	Text    string
	OnClick func(*vecty.Event)
}

func (b *Button) Render() vecty.ComponentOrHTML {
	return elem.Button(
		vecty.Text(b.Text),
		vecty.Markup(
			event.Click(b.OnClick),
		),
	)

}
