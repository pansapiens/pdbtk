# Download

## Binary Releases

| OS      | Arch  | Binary                                                                                                        | Size   |
| ------- | ----- | ------------------------------------------------------------------------------------------------------------- | ------ |
| Linux   | amd64 | [pdbtk-linux-amd64](https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-linux-amd64)             | ~2.5MB |
| Linux   | arm64 | [pdbtk-linux-arm64](https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-linux-arm64)             | ~2.4MB |
| macOS   | amd64 | [pdbtk-darwin-amd64](https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-darwin-amd64)           | ~2.6MB |
| macOS   | arm64 | [pdbtk-darwin-arm64](https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-darwin-arm64)           | ~2.5MB |
| Windows | amd64 | [pdbtk-windows-amd64.exe](https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-windows-amd64.exe) | ~2.7MB |

## Installation

### User-space Installation

#### Linux / macOS

**1 -** Download the appropriate binary and make it executable

```bash
# Using ~/.local/bin (you could also just use ~/bin if you prefer)
mkdir -p ~/.local/bin

# Download the Linux binary to your user bin directory
wget -O ~/.local/bin/pdbtk https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-linux-amd64

# Or for macOS, use:
# curl -L -o ~/.local/bin/pdbtk https://github.com/pansapiens/pdbtk/releases/download/v0.1/pdbtk-darwin-amd64

# Make it executable
chmod +x ~/.local/bin/pdbtk
```

**2 -** Ensure the `~/.local/bin` directory is on your `PATH` by adding one of these lines to your shell configuration file:

For `~/.bashrc`:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

For `~/.zshrc`:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

**3 -** Reload your shell configuration:
```bash
source ~/.bashrc
# or
source ~/.zshrc
```

#### Windows

1. Download the Windows binary from the table above
2. Move `pdbtk-windows-amd64.exe` to a directory in your PATH (e.g., `C:\Users\YourName\bin`)
3. Rename it to `pdbtk.exe` for easier use
4. Add the directory to your PATH environment variable through System Properties

### Shell Completion

After installation, you can enable shell completion for better user experience:

#### Bash

```bash
# Generate completion script
mkdir -p ~/.local/share/bash-completion/completions
pdbtk completion bash > ~/.local/share/bash-completion/completions/pdbtk
```

#### Zsh

```bash
# Generate completion script
mkdir -p ~/.zsh/completion
pdbtk completion zsh > ~/.zsh/completion/_pdbtk

# Add to your ~/.zshrc if not already present:
fpath=(~/.zsh/completion $fpath)
autoload -U compinit && compinit
```

#### Fish

```bash
# Generate completion script
mkdir -p ~/.config/fish/completions
pdbtk completion fish > ~/.config/fish/completions/pdbtk.fish
```

### Verify Installation

Test that pdbtk is properly installed:

```bash
pdbtk --help
```

You should see the help output for pdbtk.

## Building from Source

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/pansapiens/pdbtk.git
cd pdbtk

# Build the binary
go build -o bin/pdbtk .

# Or use the Makefile
make build
```

## Requirements

- No external dependencies required
- Single binary executable
