# Structure Studio

Structure Studio ist ein eigener Arbeitsbereich in CraftBox. Die Oberfläche verwendet dieselben Dokument-Tabs, Dialoge und Farben wie die anderen Editoren. Baupläne arbeiten auf unabhängigen Daten; Ingame-Befehle ändern keine Blöcke auf dem Server.

## Einstieg

- **New** erstellt einen leeren Bauplan mit frei wählbarer Größe.
- **Open** öffnet Bedrock-`.mcstructure` oder portable Structure-Studio-`.craftbox`-Projekte.
- **From world** extrahiert einen Bereich aus einer CraftBox-Aufnahme, einem lokalen Bedrock-Weltordner oder einer `.mcworld`. Die angegebenen Weltecken sind inklusive. Die Dimension ist für lokale Weltdateien wählbar.
- Die Bibliothek enthält gespeicherte Wiederherstellungen. Ein Bauplan kann in einem eigenen Tab geöffnet werden. Über die Bibliothek links können weitere Baupläne geöffnet werden.

Die Auswahl verwendet blockgenaue lokale Koordinaten. Klicks setzen abwechselnd die beiden Ecken; die aktive Ecke lässt sich mit den farbigen 3D-Griffen oder Zahlenfeldern ändern. Mehrere Auswahlboxen und benannte Teilbereiche sind möglich. **Extract selection to new document** erzeugt daraus einen eigenen Bauplan. Nicht ausgewählte Zwischenräume werden Structure-Void, sodass sie beim Einfügen erhalten bleiben.

## Bearbeiten und bauen

Die Werkzeuge unterstützen einzelne Blöcke setzen/löschen, Pipette einschließlich Blockzuständen, Füllen, Ersetzen, Kopieren, Ausschneiden und Einfügen. Die Palette durchsucht die verfügbaren Vanilla-Blöcke und enthält zusätzlich die im Dokument vorkommenden Zustände. Eigene IDs und Zustände lassen sich im Inspector eingeben. Byte-Zustände behalten beim Aufnehmen ihren NBT-Typ; neue Byte-Zustände können im Bereich **State NBT types** markiert werden.

- `Ctrl+Z`: Undo; `Ctrl+Y` / `Ctrl+Shift+Z`: Redo.
- `Ctrl+C`, `Ctrl+X`, `Ctrl+V`: Auswahl kopieren, ausschneiden, einfügen.
- `Delete`: Auswahl leeren; `Ctrl+S`: Projekt speichern.
- Texteingaben behalten ihre normalen Tastenkürzel.

Drehen erfolgt in 90°-Schritten um Y; Spiegeln ist um X oder Z möglich. Bekannte Richtungszustände, Verbindungen, Anker, Entities und Block-Entity-Richtungen werden mitgeführt. Unbekannte Richtungszustände verhindern zunächst die Änderung. Die Meldung nennt die betroffenen Zustände; im Transformationsdialog kann der Anwender eine bewusste Ausnahme wählen.

Der Y-Regler zeigt einzelne Bauschichten an. **Isolate** erzeugt eine gesonderte Schichtgeometrie, damit auch vorher innenliegende Flächen sichtbar werden. **Adjacent** ergänzt durchscheinende Nachbarschichten. Die 2D-Ebenenansicht hat Koordinaten und eine Blocklegende; bei großen Grundflächen lässt sich das angezeigte 64×64-Fenster über Start X/Z verschieben.

**Materials** zählt Blöcke und geschätzte benötigte Items getrennt, gruppiert Zustände eines Blocktyps und berücksichtigt unter anderem obere Türhälften und doppelte Slabs. Die Anzeige unterstützt vollständigen Bauplan, Auswahl oder aktuelle Ebene, CSV und Drucken. Stapelangaben werden ausdrücklich in 64er-Einheiten angezeigt; Flüssigkeiten und eigene Blöcke bleiben Schätzungen.

**Compare** vergleicht zwei Baupläne mit einem wählbaren Versatz. Fehlende, zusätzliche, falsche und falsch ausgerichtete Blöcke werden unterschieden. Die Liste und 3D-Markierungen können nach Fehlerart und aktueller Ebene gefiltert werden. Unaufgezeichnete Bereiche zählen als **unknown**, nicht als Luft. Bei sehr vielen Abweichungen sind die Gesamtsummen vollständig, die Detailbeispiele auf 10.000 und die 3D-Markierungen auf 3.000 begrenzt.

Ein anderer Bauplan lässt sich als durchscheinende Vorschau anzeigen und am Anker numerisch oder mit den 3D-Griffen bewegen. Das Platzieren in ein anderes Bauplandokument kann rückgängig gemacht werden.

## Speichern und Weltkopien

Das portable `.craftbox`-Format enthält sparse Blockdaten, beide Blockebenen, bekannte/unbekannte Bereiche, NBT, Teilbereiche, Anker, Ansicht und optional vollständige Resource-/Behavior-Packs. Die automatische Wiederherstellung speichert unabhängig davon im Datenverzeichnis `structures/sessions/<id>/`. Abhängigkeiten werden in einen eigenen Ordner übernommen, sodass temporäre Imports und Änderungen an der Quellwelt den Bauplan nicht nachträglich verändern.

`.mcstructure` bleibt ein eigener Bedrock-Export. Structure-Void (`-1`) wird von Luft unterschieden. Unbekannte Bereiche müssen vor dem Export aufgelöst oder ausdrücklich als Structure-Void ausgelassen werden. Schildtexte und Container-NBT bleiben erhalten; Inventare, Block-Entity-Daten und Entities lassen sich beim Export gesondert auswählen. Spieler werden niemals exportiert.

Eigene Blocknamen und Zustände werden erhalten. Fehlende Abhängigkeiten werden markiert; Originalpacks können über **Locate original pack**, **Add .mcpack** oder **Add folder** ergänzt werden. Eine Ersetzung erfolgt nur über den ausdrücklichen Ersetzen-Befehl.

Weltplatzierung schreibt eine **neue `.mcworld`-Datei**, basierend auf einer lokalen Weltkopie oder einer leeren Overworld. Verfügbare Modi:

- **Replace all** übernimmt auch Luft aus dem Bauplan.
- **Ignore source air** lässt Luft aus der Quelle aus.
- **Only into air** schützt bereits belegte Zielpositionen.

Unbekannte Bauplanzellen verhindern Weltplatzierungen. Vorhandene Resource-Pack-Reihenfolgen bleiben in Weltkopien erhalten; zusätzliche Abhängigkeiten werden gezielt ergänzt. Abgebrochene oder fehlgeschlagene Exporte ersetzen keine vorhandene Ausgabedatei.

## Ingame-Befehle

Diese Befehle werden lokal von CraftBox verarbeitet:

```text
/cb structure
/cb structure pos1 [x y z]
/cb structure pos2 [x y z]
/cb structure status
/cb structure clear
/cb structure save Mein Haus
/cb structure cancel
/cb structure list
/cb structure open
```

`/blueprint` ist ein Alias für `/cb structure`. Ohne Koordinaten wird die aktuelle Spielerposition auf Blockkoordinaten abgerundet. Eine Auswahl gilt nur für den jeweiligen Weltbesuch. **save** erstellt eine unabhängige Momentaufnahme der verfügbaren Chunks und übernommenen Packs. Die laufende Aufnahme kann währenddessen weiterlaufen. Auch hier bleiben nicht erfasste Chunks unbekannt.

## Umfang und technische Grundlagen

Ein Dokument ist auf 32 Millionen Zellen und 4096 Blöcke je Achse begrenzt. Große Stadtbereiche sollten in mehrere Baupläne aufgeteilt werden. Blöcke werden intern in 16³-Abschnitten gespeichert; Undo verwendet gemeinsam genutzte, unveränderliche Abschnitte mit begrenzter Historie. Die UI zeigt Zellenzahl und belegten Blockspeicher. Import, Bearbeitung und Welt-/Dateiexporte bieten Fortschrittsanzeige und Abbruch; die Vorschau hat einen eigenen Abbruch.

Unterstützt werden ausschließlich Bedrock-Formate. Automatisches Bauen auf Servern, Ingame-Hologramme und Java-Schematic-Konvertierung gehören nicht zu diesem Modul.

Referenzen: [Microsoft StructureManager](https://learn.microsoft.com/en-us/minecraft/creator/scriptapi/minecraft/server/structuremanager?view=minecraft-bedrock-stable), [Microsoft Blockzustände und Traits](https://learn.microsoft.com/en-us/minecraft/creator/reference/content/blockreference/examples/blockstateandtraitlistings?view=minecraft-bedrock-stable), [mcstructure-NBT-Aufbau](https://gist.github.com/tryashtar/87ad9654305e5df686acab05cc4b6205).
