# EByte CAN-Ethernet to CANserver Bridge

Diese Go-Anwendung verbindet einen EByte CAN-zu-Ethernet-Adapter mit Clients, die das UDP-Protokoll des [CANserver](https://github.com/collin80/CANserver) sprechen (z. B. SavvyCAN). Sie übersetzt kontinuierlich zwischen dem proprietären EByte-Binärformat und den 16-Byte-CANserver-Datagrammen und hält sowohl die TCP-Verbindung zum Adapter als auch die UDP-Session der Clients stabil.

## Funktionsumfang

* Baut eine ausgehende TCP-Verbindung zum EByte CAN-zu-Ethernet-Adapter auf und versucht bei Fehlern automatisch eine Wiederverbindung.
* Öffnet einen UDP-Listener, der eingehende CANserver-Frames entgegennimmt und Bestätigungen versendet.
* Dekodiert empfangene Frames des Adapters (Standard- und Extended-Frames) und verteilt sie an alle aktuell verbundenen CANserver-Clients.
* Unterstützt sowohl das alte "hello"- als auch das neue "ehllo"-Handshake der CANserver-Protokollversionen 1 und 2.
* Bietet strukturierte Logs mit konfigurierbaren Logleveln.

## Installation & Build

```bash
go build ./cmd/bridge
```

Der resultierende Binärname kann optional mit `-o` angepasst werden:

```bash
go build -o ebyte-canserver-bridge ./cmd/bridge
```

## Ausführung

Alle Optionen werden ausschließlich per CLI-Flags konfiguriert:

```bash
./ebyte-canserver-bridge \
  -ebyte-host 192.0.2.10 \
  -ebyte-port 4001 \
  -listen-host 0.0.0.0 \
  -listen-port 1338 \
  -reconnect-delay 2s \
  -log-level info
```

### Wichtige Flags

| Flag | Standardwert | Beschreibung |
|------|---------------|--------------|
| `-ebyte-host` | `127.0.0.1` | Hostname oder IP des EByte-Adapters |
| `-ebyte-port` | `4001` | TCP-Port des Adapters |
| `-listen-host` | `0.0.0.0` | Adresse, an die der CANserver-kompatible UDP-Server bindet |
| `-listen-port` | `1338` | Port des UDP-Servers |
| `-reconnect-delay` | `2s` | Wartezeit, bevor nach Verbindungsverlust erneut verbunden wird |
| `-log-level` | `info` | Loglevel: `debug`, `info`, `warn`, `error` |

## Nutzung mit SavvyCAN

Nach dem Start der Bridge kann SavvyCAN unter "Connection" → "Connect" die Option **CANserver** wählen und die in `listen-host:listen-port` konfigurierte Adresse angeben. Nach erfolgreichem Handshake werden die vom Adapter empfangenen CAN-Frames an alle verbundenen Clients verteilt.

## Tests

Zum Ausführen der vorhandenen Unit-Tests:

```bash
go test ./...
```

## Lizenz

Die Software steht unter der MIT-Lizenz. Details siehe [LICENSE](LICENSE).
