package gui

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/gui/layout"
	"github.com/nymtech/nym-mixnet/logger"
	"time"
)

const (
	defaultNoticePrefix = "NOTICE"
)

func initControlKeybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	// switch between views with tab
	// allow scrolling with arrows

	//if err := g.SetKeybinding("stdin", gocui.KeyArrowUp, gocui.ModNone,
	//	func(g *gocui.Gui, v *gocui.View) error {
	//		scrollView(v, -1)
	//		return nil
	//	}); err != nil {
	//	return err
	//}
	//if err := g.SetKeybinding("stdin", gocui.KeyArrowDown, gocui.ModNone,
	//	func(g *gocui.Gui, v *gocui.View) error {
	//		scrollView(v, 1)
	//		return nil
	//	}); err != nil {
	//	return err
	//}
	return nil
}

func WriteMessage(msg, senderID string, g *gocui.Gui) {
	g.Update(func(gui *gocui.Gui) error {
		messagesView, err := g.View(layout.MessagesViewName)
		if err != nil {
			return err
		}

		currentTime := time.Now()

		formattedTime := fmt.Sprintf("\x1b[%dm%s\x1b[0m",
			logger.ColorWhite,
			currentTime.Format(layout.TimeFormatting),
		)

		formattedSender := fmt.Sprintf("\x1b[1m%s:\x1b[0m",
			senderID,
		)

		formattedMessage := fmt.Sprintf("%s %s %s",
			formattedTime,
			formattedSender,
			msg,
		)

		if _, err := messagesView.Write([]byte(formattedMessage)); err != nil {
			return err
		}

		return nil
	})
}

func WriteNotice(content string, g *gocui.Gui, noticePrefix ...string) {
	g.Update(func(gui *gocui.Gui) error {
		messagesView, err := g.View(layout.MessagesViewName)
		if err != nil {
			return err
		}

		currentTime := time.Now()

		formattedTime := fmt.Sprintf("\x1b[%dm%s\x1b[0m",
			logger.ColorWhite,
			currentTime.Format(layout.TimeFormatting),
		)

		noticeText := defaultNoticePrefix
		if len(noticePrefix) == 1 {
			noticeText = noticePrefix[0]
		}
		formattedMessage := fmt.Sprintf("%s \x1b[%dm%s: %s\x1b[0m",
			formattedTime,
			logger.ColorYellow,
			noticeText,
			content,
		)

		if _, err := messagesView.Write([]byte(formattedMessage)); err != nil {
			return err
		}

		return nil
	})
}

func WriteInfo(content string, g *gocui.Gui, infoPrefix ...string) {
	g.Update(func(gui *gocui.Gui) error {
		messagesView, err := g.View(layout.MessagesViewName)
		if err != nil {
			return err
		}

		infoText := ""
		if len(infoPrefix) == 1 {
			infoText = infoPrefix[0]
		}
		formattedMessage := fmt.Sprintf("\x1b[%dm%s: %s\x1b[0m",
			logger.ColorWhite,
			infoText,
			content,
		)

		if _, err := messagesView.Write([]byte(formattedMessage)); err != nil {
			return err
		}

		return nil
	})
}

func CreateGUI() (*gocui.Gui, error) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	//g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		return nil, err
	}

	g.SetManagerFunc(layout.Layout)

	if err := initControlKeybindings(g); err != nil {
		return nil, err
	}

	return g, nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
