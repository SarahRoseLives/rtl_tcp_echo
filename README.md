# RTL_TCP_ECHO

**RTL_TCP_ECHO** is a Go application that acts as a proxy between an [`rtl_tcp`](https://osmocom.org/projects/rtl-sdr/wiki/Rtl-sdr) server and its client. It transparently passes control commands (such as frequency, gain, and sample rate), forwards IQ data, and records the IQ stream to a file. Later, you can run the application in playback mode, serving the recorded IQ data as a fake `rtl_tcp` server—allowing SDR software to connect and replay the IQ stream.

## Features

- **Proxy Mode:**  
  Forwards all rtl_tcp commands (including frequency and gain) and IQ data between client and server. Simultaneously records IQ data to a file.

- **Playback Mode:**  
  Serves a previously recorded IQ file as an rtl_tcp-compatible server for SDR software to connect and decode.

- **Transparent Command Handling:**  
  All client commands (frequency, gain, sample rate, etc.) are passed through with optional logging.

- **Simple Configuration:**  
  Easily specify listen/forward addresses and recording/playback file paths via command-line flags.

## Usage

### Build

```sh
go build -o rtl_tcp_echo
```

### Proxy Mode (Record IQ Data)

```sh
./rtl_tcp_echo --mode=proxy --listen=:1234 --forward=127.0.0.1:1234 --record=iq_recording.bin
```

- `--listen` : Address for rtl_tcp clients to connect to.
- `--forward`: Address of your actual rtl_tcp server.
- `--record` : File to save the IQ stream.

### Playback Mode (Serve Recorded IQ Data)

```sh
./rtl_tcp_echo --mode=playback --listen=:1234 --playback=iq_recording.bin
```

- `--listen`   : Address for SDR software to connect to.
- `--playback` : IQ file to serve.

## Command-Line Flags

| Flag        | Description                                  | Default                |
|-------------|----------------------------------------------|------------------------|
| `--mode`    | `proxy` (default) or `playback`              | `proxy`                |
| `--listen`  | Listen address                               | `0.0.0.0:1234`         |
| `--forward` | Forward address (proxy mode only)            | `127.0.0.1:1234`       |
| `--record`  | IQ recording file (proxy mode only)          | `iq_recording.bin`     |
| `--playback`| Playback IQ file (playback mode only)        | `iq_recording.bin`     |

## Example Workflow

1. **Start your rtl_tcp server** (e.g., on port 1234).
2. **Run RTL_TCP_ECHO in proxy mode** to intercept and record IQ data.
3. **Connect your SDR software** (e.g., SDR#, GQRX, etc.) to RTL_TCP_ECHO.
4. **Stop recording and switch to playback mode** to replay the IQ data for analysis or demonstrations.

## Protocol Support

RTL_TCP_ECHO is designed to transparently pass all rtl_tcp control commands, including:

- Set frequency
- Set sample rate
- Set gain / gain mode
- Tuner commands