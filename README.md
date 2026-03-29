# akctl

CLI tool for managing the [agentic kernel](https://github.com/elliottpolk/agentic-kernel) on a project or directory. Use it to initialize, update, and maintain the `.agentic/` system that gives AI agents stateful, structured context.

## What it does

`akctl` manages the lifecycle of the `.agentic/` directory — the kernel that defines agents, workflows, skills, and memory for AI-assisted development.

There are two core jobs:

1. **Init** — scaffold the `.agentic/` structure in any project or directory, pulling the canonical kernel from [elliottpolk/agentic-kernel](https://github.com/elliottpolk/agentic-kernel).
2. **Maintain** — keep the kernel's core elements (`core/`, `manifest.yml`, built-in agents/skills/workflows) up to date as the upstream kernel evolves, without touching anything you've added or customized.

## Install

### Quick install (Go toolchain required)

```sh
go install github.com/elliottpolk/akctl@main
```

The binary will land in `$(go env GOPATH)/bin/akctl`. Make sure that directory is on your `$PATH`.

### Build from source

```sh
git clone https://github.com/elliottpolk/akctl.git
cd akctl
go build -o akctl ./cmd
```

Then move or symlink the binary somewhere on your `$PATH`:

```sh
mv akctl /usr/local/bin/
```

## Usage

```sh
akctl [global options] command [command options]
```

### Commands

| Command | Alias | Description |
|---|---|---|
| `init` | `setup` | Initialize a project with the agentic kernel |

### Global options

| Flag | Short | Default | Description |
|---|---|---|---|
| `--log.level` | `-ll` | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `--log.format` | `-lf` | `text` | Log output format: `text`, `json` |

### Example

```sh
# Initialize the agentic kernel in the current directory
akctl init
```

## Contributing

Contributions are welcome. Please open an issue before starting significant work so we can discuss direction.

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-thing`
3. Make your changes and add tests where appropriate
4. Submit a pull request against `main`

Please keep pull requests focused. One thing per PR.

## License

See [LICENSE](LICENSE).
