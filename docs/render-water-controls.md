# Wasser, Nebel und Bildqualität

Die Regler gelten für Render- und Cinematic-Editor und werden mit Projekten und eigenen Looks gespeichert. Ältere Dokumente erhalten Standardwerte für neue Regler. Die vorhandenen Szenen-Looks enthalten abgestimmte Wasserfarben und Tiefenwerte.

## Wasser im Tab „Details“

| Regler | Wirkung |
| --- | --- |
| Shallow / Deep water color | Eigene Farbe für flaches und tiefes Wasser. |
| Water palette strength | Mischung mit den Ressourcenpaket-Farben. 0 erhält die Paketfarbe, 1 verwendet vollständig die eigene Palette. |
| Color depth | Strecke in Blöcken, über die die tiefe Farbe stärker wird. Unabhängig von Transparenz und Sichtweite. |
| Depth light color / intensity | Einstellbare Aufhellung tiefer Bereiche. Ein gestalterischer Emissionsanteil; keine Simulation von Unterwasser-Lichtquellen. |
| Water surface opacity | Deckkraft des oberflächennahen Wassers. |
| Overall water transparency | Transparenz des gesamten Wassers einschließlich tiefer Bereiche. 0 behält die berechnete Deckkraft, 1 blendet Wasser vollständig aus. |
| Underwater visibility | Sichtweite durch das Wasser. Geringe Werte verbergen den Grund schneller. |
| Shore softness | Weicher Übergang der Oberflächen-Deckkraft an flachen Ufern. Die Tiefentextur wird nicht über Felskanten hinweg weichgezeichnet. |
| Water texture detail | Kontrast der Wassertextur. 0 verwendet die mittlere Kachelfarbe, 1 das ursprüngliche Muster, Werte darüber verstärken es. |
| Water ripples / Ripple scale | Stärke und relative Größe der statischen Oberflächenwellen. Sehr kleine, entfernte Wellen werden gefiltert, um Flimmern zu begrenzen. |
| Water reflections / roughness | HDR-Umgebungsreflexion und Glätte. Gebäude werden weiterhin nicht als echte Spiegelung berechnet. |
| Water tint | Zusätzliche multiplikative Tönung; Weiß lässt die übrigen Farben unverändert. |

Die Parameter greifen auf Wasser-Meshes neuer Importe/Captures. Bei alten gepackten `.craftbox`-Dateien mit gemeinsamem Wasser-/Glas-Mesh muss die ursprüngliche Welt neu importiert werden. Glas, Eis und Portale bleiben getrennt.

## Nebel und Details

Unter „Atmosphere“ reduziert **Protect visible surfaces** den Nebel unmittelbar vor sichtbarem Gelände und Gebäuden. **Surface clear range** bestimmt die Entfernung in Blöcken. Die Wolkendichte bleibt davon unabhängig. 0 bei der Schutzstärke stellt das bisherige Verhalten wieder her.

**Protect fine detail** unter „Details“ begrenzt die Schärfung bei starken Kontrasten auf einzelnen Pixeln. Es ergänzt das PNG-Sampling, ohne eine allgemeine Unschärfe über die Map zu legen.

## Bildausschnitt und Export

Unter „Output“ steuert **Fit margin** den Rand der automatischen orthografischen Einpassung. **Vertical frame shift** verschiebt das Motiv vertikal; negative Werte platzieren es höher. Die Einpassung berücksichtigt die Verschiebung, damit die Karte vollständig im Bild bleibt. Das Ändern dieser Regler führt eine neue Einpassung aus. Ein Wechsel des Looks erhält diese Werte.

**Image sampling** bietet 1, 4 oder 16 Samples. 4 ist der allgemeine Standard; 16 eignet sich für kleine Vegetation und feine Dachkanten. Die Ausgabegröße bleibt gleich. Wolkenzeit und Wasserstruktur werden beim Sampling nicht weiterbewegt.

## Korrektur der Shader-Einbindung

Der Atlas-Schritt ersetzt die Alpha-Test-Einfügemarke vollständig. Die bisher dort angehängte Wassertiefenberechnung wurde deshalb nicht zuverlässig eingebaut. Wasser verwendet jetzt eine erhaltene Einfügemarke nach dem Texture-Sampling. Ein Regressionstest setzt Atlas-, Emissions- und Wasser-Hooks gemeinsam zusammen. Zusätzliche GPU-Tests prüfen Farbe, Tiefenwirkung, Tiefenaufhellung, Transparenz und einen Übergang zwischen zwei Wasserstücken.

Vollständig ausgeblendetes Wasser wird auch aus dem transparenten Tiefenpass für die Schärfentiefe entfernt.
