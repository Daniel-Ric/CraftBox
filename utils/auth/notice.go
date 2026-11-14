package auth

import (
	"fmt"
	"io"

	"github.com/Daniel-Ric/CraftBox/locale"
)

func AntiCheatNotice() string {
	return locale.Loc("anti_cheat_notice", nil)
}

type cliAuthCodeHandler struct{ Writer io.Writer }

func newCliAuthCodeHandler(w io.Writer) cliAuthCodeHandler { return cliAuthCodeHandler{Writer: w} }
func (c cliAuthCodeHandler) DisplayCode(uri, code string) {
	fmt.Fprintf(c.Writer, "\n%s\n\nAuthenticate at %s using the code %s.\n", AntiCheatNotice(), uri, code)
}
func (c cliAuthCodeHandler) Successful() {}
