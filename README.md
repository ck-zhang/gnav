# GNAV: GNOME Workspace Navigator

Navigate GNOME workspaces efficiently with **GNAV**, integrated with Wofi for fuzzy-search workspace switching 🚀

## Installation

```bash
go install github.com/ck-zhang/gnav@latest
```

## Usage

### Launch Wofi Workspace Picker

```bash
gnav wofi-run
```

### Launch Interactive Workspace Manager

```bash
gnav
```

### Available Commands:

- `create`      Create or expand static workspaces
- `dynamic`     Toggle dynamic workspaces
- `list`        Show workspace names
- `rename`      Rename a workspace
- `switch`      Switch workspace by index
- `wofi-run`    Interactive workspace picker via Wofi
- `wofi-switch` Switch workspace from stdin input

For additional commands:
```bash
gnav --help
```
