package layout

import (
	"github.com/jroimartin/gocui"
)

const (
	InputViewName = "input"
	MessagesViewName = "messages"
	TimeFormatting = "[15:04:05]"
)

func Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	g.Cursor = true

	if messages, err := g.SetView(MessagesViewName, 0, 0, maxX-1, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		messages.Title = " messages: "
		messages.Autoscroll = true
		messages.Wrap = true
	}

	if input, err := g.SetView(InputViewName, 0, maxY-5, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		if _, err := g.SetCurrentView(InputViewName); err != nil {
			return err
		}

		input.Title = " send: "
		input.Autoscroll = false
		input.Wrap = true
		input.Editable = true

	}
	//
	//if name, err := g.SetView("name", maxX/2-10, maxY/2-1, maxX/2+10, maxY/2+1); err != nil {
	//	if err != gocui.ErrUnknownView {
	//		return err
	//	}
	//	g.SetCurrentView("name")
	//	name.Title = " name: "
	//	name.Autoscroll = false
	//	name.Wrap = true
	//	name.Editable = true
	//}

	return nil
}
