package proxy

import (
	"github.com/Daniel-Ric/CraftBox/internal/ingamefeedback"
	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"sync"
	"time"
)

var feedbackOnce sync.Once
var defaultFeedback *ingamefeedback.Policy

func feedback() *ingamefeedback.Policy {
	feedbackOnce.Do(func() { defaultFeedback = ingamefeedback.New(utils.PathData("settings")) })
	return defaultFeedback
}
func IngameHotbarMessages() bool                       { return feedback().Enabled() }
func IngameInterfaceSettings() ingamefeedback.Settings { return feedback().Settings() }
func SetIngameInterfaceSettings(settings ingamefeedback.Settings) error {
	if err := feedback().SetSettings(settings); err != nil {
		return err
	}
	messages.SendEvent(&messages.EventIngameFeedback{Enabled: settings.Enabled})
	return nil
}
func SetIngameHotbarMessages(enabled bool) error {
	if err := feedback().Set(enabled); err != nil {
		return err
	}
	messages.SendEvent(&messages.EventIngameFeedback{Enabled: enabled})
	return nil
}
func (s *Session) hotbarMessages() bool {
	if s.feedbackPolicy != nil {
		return s.feedbackPolicy.Enabled()
	}
	return IngameHotbarMessages()
}
func (s *Session) interfaceSettings() ingamefeedback.Settings {
	if s.feedbackPolicy != nil {
		return s.feedbackPolicy.Settings()
	}
	return IngameInterfaceSettings()
}
func (s *Session) SendModelPickFeedback(message string) error {
	if !s.interfaceSettings().ModelPickFeedback || message == "" {
		return nil
	}
	return s.ClientWritePacket(&packet.Text{TextType: packet.TextTypeTip, Message: locale.DisplayText(message, true)})
}
func (s *Session) ClearHotbarFeedback() {
	if s.Client == nil {
		return
	}
	for _, kind := range []byte{packet.TextTypePopup, packet.TextTypeTip} {
		_ = s.clientWritePacket(&packet.Text{TextType: kind, Message: " "})
	}
}

func (s *Session) SendActionBar(message string) {
	if s.canShowActionBar() && message != "" {
		_ = s.ClientWritePacket(&packet.SetTitle{ActionType: packet.TitleActionSetActionBar, Text: locale.DisplayText(message, true)})
	}
}

func (s *Session) SendActionBarOrMessage(message string) {
	if s.canShowActionBar() {
		s.SendActionBar(message)
	} else {
		s.SendMessage(message)
	}
}

func (s *Session) canShowActionBar() bool {
	last := s.serverActionBar.Load()
	return s.hotbarMessages() && (last == 0 || time.Since(time.Unix(0, last)) >= 3*time.Second)
}

func (s *Session) SendExportToast(title, message string) {
	_ = s.ClientWritePacket(&packet.ToastRequest{Title: locale.Text(title), Message: message})
}
