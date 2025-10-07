# EByte CAN-Ethernet zu GVRET-Bridge

Diese Go-Anwendung verbindet einen EByte CAN-zu-Ethernet-Adapter mit Clients, die das TCP-basierte GVRET-Protokoll (z. B. [SavvyCAN](https://github.com/collin80/SavvyCAN)) sprechen. Sie übersetzt kontinuierlich zwischen dem proprietären EByte-Binärformat und den GVRET-Nachrichten und hält sowohl die TCP-Verbindung zum Adapter als auch die GVRET-Session der Clients stabil.

## Funktionsumfang

* Baut eine ausgehende TCP-Verbindung zum EByte CAN-zu-Ethernet-Adapter auf und versucht bei Fehlern automatisch eine Wiederverbindung.
* Öffnet einen TCP-Listener, der GVRET-Clients akzeptiert, das GVRET-Handshake bedient und periodische Validierungsanfragen beantwortet.
* Dekodiert empfangene Frames des Adapters (Standard- und Extended-Frames) und verteilt sie an alle aktuell verbundenen GVRET-Clients.
  Dabei werden Identifier > 0x7FF automatisch als Extended Frames markiert, falls der Adapter das Flag nicht setzt.
* Meldet dem Client eine konfigurierbare Busgeschwindigkeit und liefert GVRET-Zeitstempel auf Basis der Systemuhr.
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
  -listen-port 23 \
  -can-bitrate 500000 \
  -reconnect-delay 2s \
  -log-level info
```

### Wichtige Flags

| Flag | Standardwert | Beschreibung |
|------|---------------|--------------|
| `-ebyte-host` | `127.0.0.1` | Hostname oder IP des EByte-Adapters |
| `-ebyte-port` | `4001` | TCP-Port des Adapters |
| `-listen-host` | `0.0.0.0` | Adresse, an die der GVRET-TCP-Server bindet |
| `-listen-port` | `23` | Port des TCP-Servers |
| `-can-bitrate` | `500000` | Gemeldete CAN-Bitrate für GVRET-Clients (in bit/s) |
| `-reconnect-delay` | `2s` | Wartezeit, bevor nach Verbindungsverlust erneut verbunden wird |
| `-log-level` | `info` | Loglevel: `debug`, `info`, `warn`, `error` |

## Nutzung mit SavvyCAN

Nach dem Start der Bridge kann SavvyCAN unter "Connection" → "Connect" die Option **GVRET** wählen und die in `listen-host:listen-port` konfigurierte Adresse angeben. Die Bridge beantwortet das GVRET-Handshake (inklusive Validierungspakete) und verteilt danach die vom Adapter empfangenen CAN-Frames an alle verbundenen Clients.

## Tests

Zum Ausführen der vorhandenen Unit-Tests:

```bash
go test ./...
```

## Lizenz

Die Software steht unter der MIT-Lizenz. Details siehe [LICENSE](LICENSE).
