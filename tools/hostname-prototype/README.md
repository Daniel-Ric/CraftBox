# Hostname-Umleitung: isolierter Prototyp

Der Test verbindet einen nativen Go-Minecraft-Client über einen logischen HTTP-Hostnamen mit einem lokalen NetherNet-Relay. Dieser leitet die unveränderten Nachrichten an einen zweiten lokalen Minecraft-Testserver weiter. Login, Spielstart und ein Textpaket werden tatsächlich ausgeführt. Es werden keine Konten oder bestehenden CraftBox-Sitzungen verwendet.

```powershell
./builds/hostname-prototype/CraftBox-Hostname-Probe.exe
```

Optionen: `-hostname craftbox-probe.test` und `-report tmp/hostname-prototype/report.json`. Aus dem Quellcode: `go run ./tools/hostname-prototype`.

Der HTTP-Transport löst ausschließlich den gewählten Namen und Port prozessintern auf die lokale Relay-Adresse um. Der HTTP-Host und die logische Minecraft-Login-Adresse bleiben erhalten. Andere Ziele und HTTP-Weiterleitungen werden abgewiesen. Windows-DNS, die hosts-Datei, Flarial und die produktive CraftBox-Anwendung werden nicht verändert. LAN-Ankündigungen sind deaktiviert. Beide Server verwenden freie lokale Ports und werden nach dem Test beendet.

## Aussagekraft

Ein erfolgreicher Bericht belegt den Transportweg und die Erhaltung des Hostnamens im Login des verwendeten Go-Clients. Er beweist **nicht**, dass Minecraft für Windows denselben Wert als interne Verbindungsadresse speichert oder Flarial ihn im HUD anzeigt. `flarialHUDVerified` bleibt deshalb ausdrücklich `false`.

Die verwendete Bibliothek ergänzt bei HTTP-Adressen im Login derzeit einen zweiten Port, beispielsweise `http://craftbox-probe.test:58938:58938`. Dies ist ihr bestehendes Kompatibilitätsverhalten; der Prototyp korrigiert oder verschweigt es nicht. Auch das Entfernen von `http://` und Port im Flarial-HUD ist nicht belegt.

## Noch benötigter echter Client-Test

Dieser Schritt ist getrennt vom automatischen Test. Auf diesem Rechner konnte die verfügbare Computersteuerung Minecraft nicht erreichen. Außerdem liegt die Windows-hosts-Datei außerhalb des beschreibbaren Arbeitsbereichs. Der Prototyp installiert deshalb keine systemweite Umleitung.

Für einen manuellen Test mit einer normalen laufenden CraftBox-Capture-Verbindung auf Port 19132:

1. Vorübergehend in der Windows-hosts-Datei als Administrator die Zeile `127.0.0.1 craftbox-probe.test` ergänzen. Keinen echten Servernamen überschreiben. Eine vorhandene abweichende Zuordnung nicht ersetzen.
2. In Minecraft denselben HTTP-Verbindungsweg verwenden wie bisher, mit `http://craftbox-probe.test:19132` anstelle von `http://127.0.0.1:19132`.
3. Verbindungsaufbau und Flarial-IP-Anzeige prüfen. Interessant sind der unveränderte Hostname sowie eventuell sichtbares Schema und Port.
4. Die ausschließlich für diesen Test ergänzte hosts-Zeile anschließend wieder entfernen.

Die Endung `.test` löst sich ohne diese zusätzliche Zuordnung nicht auf. `.localhost`-Unterdomains wurden auf diesem Rechner ebenfalls nicht automatisch aufgelöst. Der Prototyp schaltet keine aktive Minecraft-Verbindung um.

Erst ein erfolgreicher echter Client-Test rechtfertigt eine Integration in CraftBox. Diese müsste insbesondere die ursprüngliche Zielauflösung für die ausgehende Verbindung erhalten und Servertransfers berücksichtigen, damit keine Rückleitung in den eigenen Proxy entsteht.
