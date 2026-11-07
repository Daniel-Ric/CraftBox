# Konsolen und PowerShell-Terminals

Die Konsole folgt dem Aufbau eines JetBrains-Toolfensters: Tab-Leiste oben,
kompakte Aktionen, Suche und eine Ausgabe, die den verbleibenden Platz ausfüllt.
Der obere Trennbalken lässt sich ziehen oder mit Pfeiltasten, Home und End
bedienen. Maximieren nutzt die gesamte Arbeitsfläche; Wiederherstellen bringt
die vorherige Höhe zurück. Beim Verkleinern des Fensters passt sich die Höhe an.

## Mehrere CraftBox-Konsolen

Mit **+** entsteht eine neue CraftBox-Konsole. „New console purpose“ wählt den
Zweck: alle Meldungen, Probleme, Verbindung, Ressourcen oder Render/Export.
Ein Doppelklick auf den Tab-Namen benennt ihn um; **×** entfernt nur diese
Ansicht. Die laufende Minecraft-Aufnahme bleibt dabei aktiv.

Jeder Tab behält unabhängig:

- Zweck, Textsuche mit Hervorhebung und Groß-/Kleinschreibung;
- Level einschließlich fatal/panic als Fehler, Quelle und auszuschließenden Text;
- Pause-Snapshot, Nachführung und eigene Leseposition;
- Lesezeichen, Zeilennummern, Zeitstempel und Schriftgröße;
- weichen Zeilenumbruch und Zusammenfassung aufeinanderfolgender Wiederholungen;
- markierte Meldung, Detailansicht und Befehlsentwurf.

Leeren betrifft den aktuellen Tab. Andere Ansichten behalten ihre Meldungen.
Die Anwendung hält die letzten 2.000 Meldungen vor; eine pausierte Ansicht hält
ihren Snapshot bis zum Fortsetzen. Die Detailansicht öffnet sich per Doppelklick
oder Rechtsklick und bietet vollständigen Text, JSON-Kopie und Quellfilter.
Export speichert die gefilterten Meldungen als UTF-8-Logdatei über den nativen
Speicherdialog. „Copy diagnostics“ kopiert ausgewählte Versions-/Sitzungsdaten.

Ansichten und Filter werden lokal wiederhergestellt. Alte Log-IDs,
Lesezeichen und Leeren-Positionen werden beim Neustart nicht auf einen neuen
Logstrom übertragen. Bis zu 16 Tabs sind möglich.

## CraftBox-Befehle

Die untere Eingabe führt folgende CraftBox-Aktionen aus. Sie ist keine Shell.
Mit ↑/↓ wird der eigene Befehlsverlauf durchsucht; `/help` zeigt die Übersicht.

| Befehl | Wirkung |
| --- | --- |
| `/find text`, `/exclude text` | Suche oder Ausschluss ändern. Ohne Text zurücksetzen. |
| `/source name`, `/level error` | Quelle bzw. Level wählen; `all` setzt zurück. |
| `/purpose problems` | Zweck wechseln: all, problems, network, assets, render. |
| `/pause`, `/resume` | Ausgabe der Ansicht einfrieren bzw. fortsetzen. |
| `/follow on` / `off`, `/wrap on` / `off` | Nachführung und Umbruch einstellen. |
| `/clear`, `/copy`, `/export` | Aktuelle Ansicht leeren, kopieren oder speichern. |
| `/bookmark` | Markierte Meldung als Lesezeichen umschalten. |
| `/open worlds`, `packs`, `skins`, `render`, `cinematic` | Arbeitsbereich öffnen. |
| `/stop` | Die laufende CraftBox-Sitzung beenden; empfangene Daten werden nach dem bestehenden Speicherablauf gesichert. |

Strg+F fokussiert die Suche. F3/Shift+F3 navigieren zwischen Treffern. Strg+L
leert die Ansicht. Shift+Escape blendet das Toolfenster aus. Strg+Shift+F12
maximiert es, solange der Fokus in der Konsole liegt.

## Echte PowerShell-Terminals

Das Terminal-Symbol neben **+** öffnet eine eigene PowerShell-Sitzung. Unter
„Console tab options“ lässt sich das Arbeitsverzeichnis für neue Terminals
festlegen. Bereits offene Shells bleiben davon unberührt. PowerShell 7 wird
verwendet, wenn es im PATH installiert ist, sonst Windows PowerShell.

Jeder PowerShell-Tab hat einen eigenen Prozess, Variablen, aktuelles Verzeichnis
und Bildschirmpuffer. Terminalhöhe und -breite ändern sich mit der Oberfläche.
Es gibt Ctrl+C-Unterbrechung, Neustart, Suche, Kopieren, Einfügen, Export und
Schriftgrößenregler. Große Einfügungen werden unter Beibehaltung von Unicode
und Eingabereihenfolge in begrenzten Blöcken übertragen.

Tabs wechseln oder die Konsole ausblenden lässt die Shell weiterlaufen.
**×** beendet die Shell einschließlich ihrer untergeordneten Prozesse.
Beim Beenden von CraftBox werden alle Terminals geschlossen. Befehle und
Sitzungen werden beim nächsten Programmstart nicht automatisch neu ausgeführt.
Bis zu acht PowerShell-Tabs sind möglich. CraftBox-Konsolen teilen weiterhin
die eine aktive Minecraft-Aufnahmesitzung.

Die PowerShell läuft mit den Rechten von CraftBox. Das eingebettete Terminal
verwendet den xterm.js-Emulator und die Windows-Pseudokonsole; es benötigt kein
zusätzliches sichtbares Konsolenfenster.

## Layout-Referenzen und Prüfung

Grundlage sind die offiziellen JetBrains-Dokumentationen zu
[Toolfenstern](https://www.jetbrains.com/help/idea/tool-windows.html) und
[Konsolen-Umbruch und Puffer](https://www.jetbrains.com/help/idea/settings-console-folding.html).
Native Schnittstellen und Begrenzungen stehen in
[console-terminal-api.md](console-terminal-api.md).

Die Prüfung umfasst React-Interaktionstests, Browser-Aufnahmen in breiter,
schmaler und maximierter Ansicht sowie echte PowerShell-Prozesse für Trennung,
Größenänderung, Ctrl+C, große Ausgaben, Neustart und Prozessbaum-Beendigung.
