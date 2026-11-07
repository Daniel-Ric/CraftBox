# Account identity

The account dialog reads Xbox display claims from the selected account's existing
XSTS client. It shows the modern gamertag (including its suffix), classic
gamertag when different, Xbox ID and sign-in status. The editable local profile
name is separate. No access token, refresh token or user hash reaches the UI.

Lookup is bounded to 12 seconds. Errors keep sign-out and profile switching
available and offer Retry. Both backend and frontend discard stale account
responses. Missing claims are not replaced by the local profile name.

The dialog identifies online captures and authenticated source catalogs as
requiring sign-in, and import, rendering, GLB export, local packs and replay as
available offline. Signed-out source tabs show a lock and a direct sign-in action.
Public status checks remain usable without signing in; connectivity checks may
need authentication. The download area also marks online targets accordingly.

Verification includes Xbox-claim formatting, stale identity rejection, failure
handling and signed-out catalog behavior. A real saved-account lookup and browser
checks confirmed identity display, responsive layout and no exposed token fields.
