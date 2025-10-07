# EByte CAN-Ethernet to GVRET Bridge

This Go application connects an EByte CAN-to-Ethernet adapter with clients that speak the TCP-based GVRET protocol (e.g. [SavvyCAN](https://github.com/collin80/SavvyCAN)). It continuously translates between the proprietary EByte binary format and GVRET messages while keeping both the TCP connection to the adapter and the GVRET session with clients alive.

## Features

* Establishes an outgoing TCP connection to the EByte CAN-to-Ethernet adapter and automatically retries when the link drops.
* Opens a TCP listener that accepts GVRET clients, performs the GVRET handshake, and responds to periodic validation requests.
* Decodes adapter frames (standard and extended) and distributes them to all currently connected GVRET clients.
  Identifiers greater than 0x7FF are automatically flagged as extended frames if the adapter omits the flag.
* Reports a configurable bus bitrate to the client and provides GVRET timestamps based on the system clock.
* Offers structured logging with configurable log levels.

## Installation & Build

```bash
go build ./cmd/bridge
```

The name of the resulting binary can optionally be adjusted via `-o`:

```bash
go build -o ebyte-canserver-bridge ./cmd/bridge
```

## Usage

All options are configured via CLI flags only:

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

### Important Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-ebyte-host` | `127.0.0.1` | Hostname or IP address of the EByte adapter |
| `-ebyte-port` | `4001` | TCP port of the adapter |
| `-listen-host` | `0.0.0.0` | Address the GVRET TCP server binds to |
| `-listen-port` | `23` | Port of the TCP server |
| `-can-bitrate` | `500000` | CAN bitrate reported to GVRET clients (in bit/s) |
| `-reconnect-delay` | `2s` | Waiting time before reconnecting after a disconnect |
| `-log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |

## Using SavvyCAN

After starting the bridge, choose **GVRET** under "Connection" → "Connect" in SavvyCAN and point it to the `listen-host:listen-port` pair. The bridge completes the GVRET handshake (including validation packets) and then forwards the CAN frames received from the adapter to all connected clients.

## Tests

Run the available unit tests with:

```bash
go test ./...
```

## License

The software is licensed under the MIT License. See [LICENSE](LICENSE) for details.
