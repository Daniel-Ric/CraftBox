package locale

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

type Strmap map[string]interface{}

//go:embed *.yaml messages/*.json
var localesFS embed.FS
var bundle = i18n.NewBundle(language.English)
var active atomic.Pointer[localization]
var catalogs sync.Map
var minecraftFormatting = regexp.MustCompile(`(?i)§[0-9a-z]`)

type localization struct {
	code     string
	messages map[string]string
	legacy   *i18n.Localizer
}

func init() {
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	for _, code := range []string{"en", "zh-Hans", "es", "de", "fr", "pt", "ja"} {
		if _, err := bundle.LoadMessageFileFS(localesFS, code+".yaml"); err != nil {
			logrus.WithError(err).WithField("language", code).Warn("Could not load language catalog")
		}
	}
	SetLanguage("en")
}

func languageCode(value string) string {
	code := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "-")
	switch code {
	case "zh", "zh-cn", "zh-sg", "zh-hans", "zh-hans-cn":
		return "zh-Hans"
	}
	base, _, _ := strings.Cut(code, "-")
	switch base {
	case "en", "es", "de", "fr", "pt", "ja":
		return base
	default:
		return "en"
	}
}

func SetLanguage(value string) {
	code := languageCode(value)
	if current := active.Load(); current != nil && current.code == code {
		return
	}
	cached, found := catalogs.Load(code)
	if !found {
		next := &localization{code: code, legacy: i18n.NewLocalizer(bundle, code, "en")}
		if code != "en" {
			if raw, err := localesFS.ReadFile("messages/" + code + ".json"); err == nil {
				_ = json.Unmarshal(raw, &next.messages)
			}
		}
		cached, _ = catalogs.LoadOrStore(code, next)
	}
	active.Store(cached.(*localization))
}

func Language() string {
	return active.Load().code
}

func Text(source string) string {
	messages := active.Load().messages
	if len(messages) == 0 {
		return source
	}
	if result := messages[source]; result != "" {
		return result
	}
	trimmed := strings.TrimSpace(source)
	if trimmed != "" {
		if result := messages[strings.Join(strings.Fields(trimmed), " ")]; result != "" {
			before, after, _ := strings.Cut(source, trimmed)
			return before + result + after
		}
	}
	return source
}

func Format(source string, values ...any) string {
	return fmt.Sprintf(Text(source), values...)
}

func DisplayText(value string, preserveColors bool) string {
	if Language() == "en" && preserveColors {
		return value
	}
	var result strings.Builder
	cursor := 0
	appendText := func(raw string) {
		for index, line := range strings.Split(raw, "\n") {
			if index > 0 {
				result.WriteByte('\n')
			}
			result.WriteString(Text(line))
		}
	}
	for _, match := range minecraftFormatting.FindAllStringIndex(value, -1) {
		appendText(value[cursor:match[0]])
		if preserveColors {
			result.WriteString(value[match[0]:match[1]])
		}
		cursor = match[1]
	}
	appendText(value[cursor:])
	return result.String()
}

func Loc(id string, tmpl Strmap) string {
	s, err := active.Load().legacy.Localize(&i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: tmpl,
	})
	if err != nil && s == "" {
		return fmt.Sprintf("failed to translate! %s", id)
	}
	return s
}

func Locm(id string, tmpl Strmap, count int) string {
	s, err := active.Load().legacy.Localize(&i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: tmpl,
		PluralCount:  count,
	})
	if err != nil && s == "" {
		return fmt.Sprintf("failed to translate! %s", id)
	}
	return s
}
