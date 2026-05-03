# sharp

`sharp` is a local developer toolkit for terminal-first workflows. It provides both an interactive Bubble Tea TUI and scriptable CLI commands for JSON, encoding, hash, time, text, network, conversion, JWT, and generators.

## Install

```bash
go install ./cmd/sharp
```

## TUI

```bash
go run ./cmd/sharp
```

Shortcuts:

- `/`: search tools
- `j/k`: move selection
- `i`: edit input
- `o`: edit options
- `Tab`: switch focus
- `Enter`: run selected tool
- `y`: copy output
- `s`: save output to `sharp-output.txt`
- `p`: move output into input for chaining
- `?`: help
- `q`: quit

## CLI Examples

```bash
echo '{"data":{"id":1}}' | go run ./cmd/sharp json pretty
echo '{"data":{"id":1}}' | go run ./cmd/sharp json get --path data.id
go run ./cmd/sharp b64 encode hello
go run ./cmd/sharp b64 decode aGVsbG8=
go run ./cmd/sharp url encode "a=b&c=d"
go run ./cmd/sharp hash sha256 hello
go run ./cmd/sharp time now
go run ./cmd/sharp time from 1714723200
go run ./cmd/sharp uuid v4
go run ./cmd/sharp password
```

List tools:

```bash
go run ./cmd/sharp list
```

## Documentation

- [PRD](docs/PRD.md)
- [Development Plan](docs/DEVELOPMENT_PLAN.md)
