package popups

import (
	"fmt"
	"os"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/Daniel-Ric/CraftBox/ui/gui/guim"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/updater"
)

const (
	stateInit = iota
	stateLoading
	stateAsk
	stateDownloading
	stateError
	stateFinished
)

type UpdatePopup struct {
	g           guim.Guim
	state       int
	startButton widget.Clickable
	updating    bool

	update *updater.Update
	err    error
}

var _ Popup = &UpdatePopup{}

func NewUpdatePopup(g guim.Guim) Popup {
	return &UpdatePopup{
		g:     g,
		state: stateInit,
	}
}

func (p *UpdatePopup) ID() string {
	return "update"
}

func (p *UpdatePopup) Close() error {
	return nil
}

func (p *UpdatePopup) Layout(gtx C, th *material.Theme) D {
	if p.state == stateInit {
		p.state = stateLoading
		go func() {
			update, err := updater.GetLatest()
			if err != nil {
				p.err = err
				p.state = stateError
				return
			}
			p.update = update
			p.state = stateAsk
		}()
	}

	if p.startButton.Clicked(gtx) && !p.updating && p.update != nil {
		p.updating = true
		p.state = stateDownloading
		go func() {
			p.err = updater.DoUpdate(p.update)
			if p.err == nil {
				p.state = stateFinished
				p.err = updater.Restart()
				if p.err == nil {
					os.Exit(0)
					return
				}
				p.state = stateError
			} else {
				p.state = stateError
			}
			p.updating = false
		}()
	}

	return LayoutPopupBackground(gtx, th, p.ID(), func(gtx C) D {
		return layout.Inset{
			Top:    unit.Dp(25),
			Bottom: unit.Dp(25),
			Right:  unit.Dp(35),
			Left:   unit.Dp(35),
		}.Layout(gtx, func(gtx C) D {
			var children []layout.FlexChild
			switch p.state {
			case stateInit:
			case stateLoading:
				return layout.Center.Layout(gtx, material.H3(th, "Loading...").Layout)
			case stateAsk:
				children = append(children,
					layout.Rigid(material.Label(th, 20, fmt.Sprintf("Current: %s\nNew:     %s", utils.Version, p.update.Version)).Layout),
					layout.Rigid(material.Button(th, &p.startButton, "Do Update").Layout),
				)
			case stateDownloading:
				return layout.Center.Layout(gtx, material.H3(th, "Updating...").Layout)
			case stateError:
				if p.err == nil {
					return layout.Center.Layout(gtx, material.H3(th, "Update failed").Layout)
				}
				return layout.Center.Layout(gtx, material.H1(th, p.err.Error()).Layout)
			case stateFinished:
				return layout.Center.Layout(gtx, material.H3(th, "Restarting...").Layout)
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (p *UpdatePopup) HandleEvent(event any) error {
	return nil
}
