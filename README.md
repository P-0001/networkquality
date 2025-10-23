# networkquality

A minimal Go CLI for measuring network quality across download, upload, and latency metrics. The tool streams real traffic against public test endpoints and prints a concise summary styled after the macOS `networkquality` utility.

## Features
- **Active measurements**: Parallel HTTP downloads and uploads to estimate capacity in Mbps.
- **Latency insights**: Idle and loaded latency sampling to classify responsiveness.
- **Rich summary output**: ASCII dashboard with throughput bars and quality score.
- **Lightweight UX**: Spinner while tests run, quick-test mode, and built-in version reporting.
- **Extensible config**: Central `network/TestConfig` exposes test duration, connections, and server lists.

## Goals
- Simple to use
- Lightweight
- Cross platform
- Extensible
- Fast
- Accurate

## Installation

### Using `go install` (Recommended)
```bash
go install github.com/P-0001/networkquality@latest
```

This installs the `networkquality` command to your `$GOPATH/bin` directory.

### Using Releases
- Download the latest archive from the [releases page](https://github.com/P-0001/networkquality/releases).
- Place the binary somewhere on your `PATH` (e.g., `/usr/local/bin` or `%USERPROFILE%\bin`).

### From Source
- **Prerequisites**: Go 1.21+
- **Clone**:
```bash
git clone https://github.com/P-0001/networkquality.git
cd networkquality
```
- **Build**:
```bash
go build -o networkquality.exe .
```

## Usage
Run the CLI directly or via `go run`.

```bash
./networkquality.exe
```

Common flags:
- **`-d <seconds>`**: Total test duration (default `10`).
- **`-c <count>`**: Parallel connection count (default `4`).
- **`-q`**: Quick 5-second test.
- **`-v`**: Verbose mode (prints config and timing).
- **`-version`**: Display the CLI version.
- **`-h`**: Show inline help.

Example verbose run with spinner and sample output:
```bash
$ ./networkquality.exe -v
networkquality
==============
Configuration:
  Test duration: 10s
  Connections: 4

Running network quality test... done

=========== SUMMARY ===========
Uplink capacity: 200.132 Mbps
Downlink capacity: 855.863 Mbps
Responsiveness: High (10.000 milliseconds)
Idle Latency: 7.000 milliseconds

========== QUALITY ============
Overall: ⭐ Excellent

======== PERFORMANCE ==========
Download: [████████████████████] 855.86 Mbps
Upload:   [████████████████████] 200.13 Mbps
Latency:  [███████████████████░] 7.00 ms

Test completed in 17.45 seconds
```

## Configuration
All runtime options originate from `network/TestConfig` in `network/quality.go`:
- **`TestDuration`**: Total duration per measurement pass.
- **`NumConnections`**: Concurrent workers for load generation.
- **`TestServers`**: Download and latency endpoints (first value used for bulk download).
- **`UploadServers`**: POST targets for uplink throughput.
- **`UploadChunkSize`**: Payload size per POST (bytes).

Override these values in code before invoking `network.RunQualityTest()` or via CLI flags where available.

## Development
- **Run from source**:
```bash
go run ./cmd
```
- **Lint / fmt**: Standard Go tooling (`go fmt`, `go vet`).
- **Tests**: (Planned) — see roadmap.

## Roadmap
Upcoming ideas are tracked in `todo.md`:
- **Add color to console output**
- **Support config file**
- **Introduce automated tests**
- **Expand server coverage**
- **Collect additional metrics**
- **Expose more CLI options**

Contributions and suggestions are welcome.
