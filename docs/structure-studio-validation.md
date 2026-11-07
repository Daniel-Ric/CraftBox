# Structure Studio: Validierung

## Capture- und Ladeverbesserungen am 6. November 2026

Build: `1.64.0-structure-studio-capture-fix-dev`. EXE: `builds/Craft-Box-StructureStudio.exe` (241.662.464 Bytes).
SHA-256: `6E76A42F73B3E4D9802E8CA17910BD573EE34C95881F14C29BA2B843093EC6EC`.

- Ingame-Eckpunkte ohne explizite Koordinaten verwenden jetzt die Fußposition: Bedrock-Paket-Y minus 1,62, dann Abrunden. Positive und negative Koordinaten sowie ungültige Positionen sind getestet. Explizite Koordinaten bleiben unverändert.
- Die Snapshot-Chatmeldung verwendet normale Punkte statt des typografischen Auslassungszeichens.
- Der Bibliotheksdialog öffnet weitere gespeicherte Ingame-Baupläne als eigene Projekte; vorhandene Dokumente werden zuerst gesucht und behalten ihre Tabs.
- Gemeinsame Editor-Ladeanzeige beim Öffnen und beim Aufbau der Vorschau, einschließlich Nachbarebenen und Ghost. Abbruch und Fehler beenden den Ladezustand.
- 38 Frontend-Tests, TypeScript/Produktionsbuild und Go-Tests mit GUI-Tags für `handlers/worlds`, `ui/desktop`, `internal/structure` und `utils/proxy` bestanden.

Visuelle Prüfbilder: `tmp/structure-loading-screen.png`, `tmp/structure-blueprint-picker.png`.

## UI-Abgleich am 6. November 2026

Aktueller Build: `1.64.0-structure-studio-ui-dev`, Windows GUI/Production.
Ausgabe: `builds/Craft-Box-StructureStudio.exe` (241.656.832 Bytes).
SHA-256: `116C458817E2DE76EDD48C85E38CAE566F9445D01A927A99EE5FFB457FCA0BE2`.

- Bibliothek, Vorschau, Eigenschaften und Leerzustand an die bestehenden Workspaces angepasst; gemeinsame Theme-Tokens statt eigener Farben.
- Alle vier Modals verwenden gemeinsame Titel-, Inhalts- und Aktionsbereiche. Der Inhalt scrollt bei niedrigen Fenstern unabhängig vom Footer.
- Playwright-Vergleich mit Resource Packs und dem Render-Dialog: Workspace-Leiste 39 px, Seitenbereiche 250/310 px, Dialog-Titelleiste 44 px und Eingaben 28 px. Farben und Dialog-Footer stimmen mit den Referenzen überein.
- Helles und dunkles Theme, Eigenschaftsansicht bei 800 px Fensterbreite und Importdialog bei 480 px Fensterhöhe geprüft. Keine JavaScript-Laufzeitfehler.
- Alle 36 Frontend-Tests sowie TypeScript und Produktionsbuild erfolgreich.
- Go-Tests mit `-tags gui` für `ui/desktop`, `internal/structure` und `handlers/worlds` erfolgreich.
- Native UI-Integration erneut erfolgreich: Bearbeiten, Undo, Ebenenwechsel, Materialliste, Bedrock-Export und mehrere Dokument-Tabs.

Messwerte: `tmp/structure-redesign-metrics.json`. Vergleichsbilder: `tmp/structure-redesign-*.png`.

## Ursprüngliche Modulvalidierung am 5. November 2026

Build: `1.64.0-structure-studio-dev`, Windows GUI/Production.

Ausgabe: `builds/Craft-Box-StructureStudio.exe` (241.647.104 Bytes).

SHA-256: `D4B106C3ABE01BB909158C59329B38184EC3E0AED04684BE3B802421A7097469`.

Die 18.235 Dateien des Vanilla-Asset-Verzeichnisses wurden nach `builds/assets/vanilla-3d` übernommen. Die EXE verwendet diesen danebenliegenden Asset-Ordner.

## Bestandene Prüfungen

- Frontend: alle 34 Tests aus acht Testdateien; TypeScript und Vite-Produktionsbuild erfolgreich.
- Go: Programm-Pakete unter `handlers`, `internal`, `ui`, `utils`, `subcommands`, `cmd`, `assets` und `locale` erfolgreich geprüft; zusätzlich GUI-Prüfung der Desktop-Integration.
- Windows-EXE mit `GOWORK=off`, `-mod=readonly`, `gui,production` und `-trimpath` erfolgreich gebaut.
- Eigenständige NBT-Fixtures prüfen Little-Endian, Z/Y/X-Reihenfolge, beide Blockebenen, Structure-Void, unbekannte Bereiche und int64-Präzision.
- Regressionen prüfen COW-Undo, Ausschneiden von Entities, Richtungsdaten, Mauerverbindungen, Shulker-Facing, Doppeltruhenreferenzen und getrennte Entity-/Spieler-IDs.
- Portable Projekte und Recovery prüfen eingebettete sowie selbst übernommene Packs, Ansicht, Anker und Schutz vor ZIP-Pfadüberschreitungen.
- Live-Snapshots bleiben nach weiteren Capture-Updates unverändert; fehlende Chunks bleiben unbekannt. Ein Ressourcenpack mit gültigem JSONC-Manifest wird übernommen.
- Weltkopie-Tests prüfen erhaltene Originaldaten, Platzierung, Pack-Reihenfolge, inaktive Packs und Schutz einer bestehenden Ausgabedatei bei Fehlern.
- Vorschau-Abbrüche sind pro Viewer isoliert; Hauptansicht und Ghost desselben Dokuments können gleichzeitig laden.

Die Playwright-Prüfung verwendet `tmp/structure-proof/main.go` als lokale Verbindung zum echten Desktop-Go-Datenkern. Sie hat Bearbeiten, natives Undo, schnelle Ebenenwechsel, Materialberechnung, `.mcstructure`-Export und mehrere Dokument-Tabs gemeinsam durchlaufen. Ergebnis: keine JavaScript-Laufzeitfehler und keine unerwarteten Fehler der Structure-API.

Prüfbilder: `tmp/structure-studio-3d.png`, `tmp/structure-studio-layer.png`, `tmp/structure-studio-materials.png`.

## Einschränkung der globalen Repository-Prüfung

`go test ./...` wird durch bereits vorhandene Hilfsdateien außerhalb der Programm-Pakete unter `tmp` blockiert: veraltete Importpfade, gemischte Paketnamen und unter `subcommands/resourcepack-d/resourcepack-d.go` NUL-Zeichen. Diese Dateien wurden nicht geändert. Die oben genannten Programm-Pakete und die neue EXE lassen sich separat erfolgreich prüfen beziehungsweise bauen.
