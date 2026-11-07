# Adobe-style CraftBox interface

Active scope: all application dialogs (including error/info/confirmation), the
start/new-project flow, one accent family, inline validation across input flows,
custom tooltips, and form-control hover/focus/disabled/invalid states. Application
copy remains English. Native operating-system file pickers are outside the app's
DOM. Existing capture, rendering and export behavior must remain functional.

References consulted 2026-10-30:
- https://helpx.adobe.com/photoshop/desktop/create-open-import-images/create-images/create-a-new-image-in-photoshop.html
- https://www.adobe.com/learn/photoshop/web/create-new-document-photoshop
- https://spectrum.adobe.com/foundations/behavior/states
- https://react-spectrum.adobe.com/v3/Tooltip.html
- https://spectrum.adobe.com/web/rsp/components/help-text
- https://opensource.adobe.com/spectrum-design-data/tokens/color-palette/

Direction: neutral gray tool surfaces, distinct document-setup dialogs and
compact message dialogs, preset grid plus right-hand details, a single blue
accent with derived shades, quiet field borders and a visible two-pixel focus
indicator. Primary actions are rounded; toolbar controls stay compact. Validation
is local, explicit and accessible. Tooltips support keyboard focus as well as hover.

Baseline inventory: 23 DialogContent call sites, 28 title attributes, 204 native
input/select/textarea call sites, plus reusable UI primitives. Each requires
coverage, not just a stylesheet change.

Implementation / verification tracker (keep active until fully audited):
- [ ] Reference screenshots and exact accent evidence.
- [ ] Shared dialog/control tokens and full state matrix.
- [ ] Distinct error/info/warning/confirmation layout.
- [ ] Photoshop-style welcome/new-project layout and behavior.
- [ ] Native title replacement and accessible custom tooltips throughout.
- [ ] Native constraints and semantic/cross-field validation across all flows.
- [ ] Remove independent UI blue literals without changing scene/asset colors.
- [ ] Full component tests, browser checks at multiple sizes/themes, build.
- [x] Preserve and verify preceding merged-world rendering corrections.
- [ ] Updated EXE installed in D:/bedrock-tools with backup and verified hash.

Preceding rendering work is verified and installed; see
[merged-world-render-fixes.md](merged-world-render-fixes.md) for the real-world
audit, complete meshing/GLB fixtures, visual checks and installed EXE hash.

## Current evidence and remaining audit

The preceding goal turn made verified progress (renderer fixes, real-world
audit, tests, native build/start and installed EXE). This continuation makes
further authoritative UI changes; it is not a wait or a blocked turn.

Current inventory: 23 DialogContent sites and 215 input/select/textarea sites
(including shared primitives), recorded by `tmp/adobe-ui/inventory.cjs`.
All seven actual forms use StudioForm. Nested validation propagates to external
dialog submit buttons. Invalid numeric drafts cannot be submitted or used by
2D selection save/export actions; cancellation remains available. Coordinate
limits, source bounds ordering, filenames and profile names have local rules.
Continuous camera/lighting/atmosphere/animation fields retain arbitrary legal
decimals and use explicit keyboard increments instead of a quantizing step grid.
Empty time and custom numeric-text fields now show semantic errors.

New regression cases were observed failing before their fixes: nested/external
submit state, invalid 2D selection actions, shifted lens/scale step grids,
continuous keyboard increments, empty time, empty custom numeric text, and
hexadecimal-looking DNS hosts with ports. Browser verification also found and
fixed an over-specific base input selector that hid the red error border.

Browser artifacts in `tmp/adobe-ui/browser/` verify the actual shared components:
welcome/recent opening, preset changes, invalid/disabled actions, a red field
border, continuous arrow-key editing, custom tooltip, four message tones,
four themes, and narrow layout. `result.json` records the latest run. This is
evidence for shared behavior, **not** proof that every large application dialog
has been visually accepted.

Latest verification after the empty custom-text validation fix: **89 frontend
test files / 300 tests pass**, TypeScript build passes, Vite production build
passes (existing bundle-size warning remains). Browser checks pass with zero
page errors, including the red border after the CSS specificity correction and
continuous keyboard editing. No production native-title attributes/assignments
were found in the source scan. Read-only review confirmed nested form/external
submit handling and identified the 2D selection draft issue, now fixed and tested.

Reference evidence now also includes the official Adobe Learn tutorial's New
Document frame (local `photoshop-new-document-reference.png`) and the September
2026 Photoshop Preferences screenshot. Layout uses the documented preset grid
and right-hand details. Important distinction: the Preferences reference has an
outlined gray OK button; the New Document Create button is blue. The current
single accent is `#1473e6`; exact pixel identity to every Photoshop 2026 dialog
is not established by these compressed/reference images and must not be claimed.

Remaining required work:

- Visually exercise every real dialog, including source details, account,
  import/export, libraries, composer, preferences, dirty-project guard and errors,
  at relevant narrow sizes and themes. Check actual body/footer overflow.
- Finish the semantic input audit, including directory/path failures, live
  editor controls and actions outside ordinary forms; free-text search/comment
  inputs and enumerated selects should keep their appropriate unrestricted rules.
- Finish the control state matrix (including hover/pressed and open select/color
  controls), color-source audit and reference comparison. Do not infer full
  coverage from a gallery or zero native-title search alone.
- Run final checks and build/install the completed UI release. The currently
  installed merged-models EXE predates this continuation's validation fixes.

The full user objective remains active.
