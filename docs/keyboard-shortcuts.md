# Keyboard shortcuts

Open Preferences → Keyboard shortcuts to configure the 92 application, workspace,
editor, camera, cinematic, structure, rig, texture, audio, document and console actions. The command search
(Ctrl+K by default), menus, workspace tooltips and shortcut help show the saved
bindings.

Binding fields, recording previews and Help show compact key icons. Letters use
a keyboard glyph followed by the letter; punctuation, arrows, modifiers and
numpad keys use readable symbols. Focus a binding field to edit its original
physical key syntax; leaving the field restores the icon display.

- Enter physical key codes, for example `Ctrl+KeyS`, `Ctrl+Shift+Digit2`, `F1`
  or `Numpad7`. Modifier order is Ctrl, Alt, Shift. Ctrl also accepts Command.
- Separate alternative bindings with `;`. An action accepts up to eight bindings.
- Separate sequence steps with ` > `, for example `Ctrl+KeyG > KeyF`.
  Up to four steps are supported; the timeout is configurable from 250–5000 ms.
- Record opens an inline recorder beside the action. Choose Replace binding
  or Add alternative after pressing your keys. Escape cancels and Tab moves
  to the recorder buttons. Release modifiers between sequence steps. Camera
  controls use a single combination; movement continues while held. Tab and
  Escape bindings can still be entered directly in the field.
- Clear a field to disable an action. Reset restores one action; Restore all
  defaults restores the entire profile. Valid changes save immediately. Invalid
  edits leave the last valid profile active; Revert unsaved changes restores it.
- Search by action, context or key, filter by context, or show modified actions.
- Advanced & profiles contains syntax help, sequence timeout, import/export and
  Restore all defaults. Import validates and applies a JSON profile. Export
  downloads the current valid profile.
  Invalid keys, unknown actions, duplicates and ambiguous sequence prefixes in
  overlapping contexts prevent saving. Profiles are stored locally on this device.

The Edit keyboard shortcuts link in Help closes Help and opens the shortcut
category directly in Preferences. The category appears below Console in the
settings navigation. There is no floating save bar over the action list.

New defaults include Ctrl+backtick for the console, Ctrl+Shift+H for Start,
Ctrl+Shift+A for the account, Ctrl+Alt+T for theme switching, Ctrl+R to reload,
Ctrl+Alt+O to open a project, and Ctrl+Alt+E for 3D export. In the scene editor,
F fits the world, V enters inspect view, Numpad0 selects isometric view and Numpad7
selects top view. Cinematic adds K for a keyframe, M for a marker, P for the camera
path, I for scene styling, and Home/End for the sequence endpoints. Existing
project, workspace, playback, frame, structure and rig bindings remain available.

Shortcuts pause behind dialogs and open menus, and terminal input remains owned
by the terminal. Text entry suppresses plain keys and editing actions; configured
application commands and project saving with Ctrl/Command remain available.
IME composition and AltGr text input do not trigger shortcuts. Holding keys only
repeats frame stepping. Focused free-camera controls consume their own keys before
editor actions, and changing profiles or losing focus clears held movement.
Space/C movement still follows the existing vertical navigation preference.
Standard field, tab, dialog and terminal keyboard interactions retain their usual
behavior.

Verification: `npm test` and `npm run build` from `ui/desktop/frontend`.
