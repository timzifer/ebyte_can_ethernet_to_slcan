# EByte CAN-Ethernet to SLCAN Bridge

Dieses Repository enthält eine Go-Applikation, die einen EByte CAN-zu-Ethernet-Adapter mit einem TCP-Endpunkt im Lawicel/SLCAN-Format verbindet. Die Anwendung kann so als Gateway zwischen dem proprietären Binärprotokoll des Adapters und gängigen Tools wie SavvyCAN fungieren.

## Funktionsumfang

* Baut eine ausgehende TCP-Verbindung zu einem EByte CAN-zu-Ethernet-Adapter auf und hält sie bei Verbindungsabbrüchen automatisch aufrecht.
* Öffnet einen TCP-Listener, der angeschlossenen Clients einen Lawicel/SLCAN-kompatiblen Stream bereitstellt.
* Wandelt CAN-Frames zwischen dem EByte-Protokoll und dem SLCAN-Textformat um.
* Unterstützt die grundlegenden SLCAN-Kommandos `O` (open) und `C` (close) sowie Stubs für eine stabile Kommunikation.
* Bietet strukturierte Logs mit konfigurierbaren Logleveln.

## Installation & Build

```bash
go build ./cmd/bridge
```

Der resultierende Binärname kann optional mit `-o` angepasst werden:

```bash
go build -o ebyte-slcan-bridge ./cmd/bridge
```

## Ausführung

Alle Optionen werden ausschließlich per CLI-Flags konfiguriert:

```bash
./ebyte-slcan-bridge \
  -ebyte-host 192.0.2.10 \
  -ebyte-port 4001 \
  -listen-host 0.0.0.0 \
  -listen-port 20108 \
  -reconnect-delay 2s \
  -log-level info
```

### Wichtige Flags

| Flag | Standardwert | Beschreibung |
|------|---------------|--------------|
| `-ebyte-host` | `127.0.0.1` | Hostname oder IP des EByte-Adapters |
| `-ebyte-port` | `4001` | TCP-Port des Adapters |
| `-listen-host` | `0.0.0.0` | Adresse, an die der SLCAN-Server bindet |
| `-listen-port` | `20108` | Port des SLCAN-Servers |
| `-reconnect-delay` | `2s` | Wartezeit, bevor nach Verbindungsverlust erneut verbunden wird |
| `-log-level` | `info` | Loglevel: `debug`, `info`, `warn`, `error` |

## Nutzung mit SavvyCAN

Nach dem Start der Bridge kann SavvyCAN (oder ein anderes SLCAN-kompatibles Tool) eine TCP-Verbindung zum `listen-host:listen-port` herstellen. Sobald ein Client das `O`-Kommando sendet, werden CAN-Frames zwischen Adapter und Client bidirektional übertragen.

## Tests

Zum Ausführen der vorhandenen Unit-Tests:

```bash
go test ./...
```

## Lizenz

Die Software steht unter der MIT-Lizenz. Details siehe [LICENSE](LICENSE).
