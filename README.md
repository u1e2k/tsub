# tsub - A fast CLI micro-blogging client for Obsidian Vaults

`tsub` (Twitter Style Ubiquitous Blogging) is a minimalist, keyboard-centric command-line interface (CLI) tool for capturing and reviewing micro-posts directly from your terminal. It seamlessly integrates with your existing Obsidian Vault, allowing you to maintain a consistent micro-blogging habit without ever leaving your shell.

## Features

### ⚡ Lightning Fast
- **Instant Capture**: Draft and publish micro-posts in seconds with a single keystroke.
- **Optimized Rendering**: Pre-renders posts with full Unicode and East Asian Width support for a stutter-free viewing experience.

###  Obsidian Integration
- **Smart Vault Detection**: Automatically detects your Obsidian Vault at `.vault` or uses the `TSUB_VAULT_DIR` environment variable.
- **Daily Notes**: Posts are appended to your daily note (`YYYY-MM-DD.md`) with proper YAML frontmatter and nested bullet formatting.

### 📱 Flexible Input Methods
- **Native Mode**: Type directly in your terminal using **Tab** to start typing (content follows automatically).
- **External Editor Mode**: Use your favorite editor (like VS Code, Vim, or Nano) for longer posts.
  - **Keyboard Shortcut**: `Ctrl+E` to open your external editor.
  - **Automatic Updates**: Changes in the external editor are automatically synced to your live preview.

### 🎯 TUI Experience
- **Live Preview**: See your posts rendered in real-time as you type or edit.
- **Compact Design**: Clean, modern interface using the [Charm](https://github.com/charmbracelet) ecosystem.
- **Full Unicode Support**: Proper rendering of emojis and double-width characters using `go-runewidth`.

## Installation

### 🚀 Quick Install (Raspberry Pi & Linux)

Install or update `tsub` to `/usr/local/bin` using the automated install script:

```bash
curl -fsSL https://raw.githubusercontent.com/u1e2k/tsub/main/install.sh | bash
```

The script automatically detects your architecture (`arm64` / `armv7l` / `x86_64`) and downloads the latest release binary.

### 📦 Pre-built Binaries

Pre-compiled standalone binaries for Linux are available on the [Releases](https://github.com/u1e2k/tsub/releases) page:
- **Raspberry Pi OS 64-bit**: `tsub-linux-arm64`
- **Raspberry Pi OS 32-bit**: `tsub-linux-armv7`
- **x86_64 Linux PC/Server**: `tsub-linux-amd64`

### 🛠️ Building from Source

#### Prerequisites
- Go 1.22+

1. Clone the repository:
   ```bash
   git clone https://github.com/u1e2k/tsub.git
   cd tsub
   ```

2. Build the binary:
   ```bash
   go build -ldflags="-s -w" -o tsub .
   ```

3. (Optional) Install to PATH:
   ```bash
   sudo install -m 755 tsub /usr/local/bin/
   ```

## Usage

### Basic Commands

**Start the application:**
```bash
./tsub
```

**Check the version:**
```bash
./tsub --version
```

### Configuration

By default, `tsub` stores your daily micro-posts at `~/vault/Daily` (`$HOME/vault/Daily`). You can customize the storage directory by setting the `TSUB_VAULT_DIR` environment variable (supports `~` expansion):

```bash
# Example 1: Use a custom Obsidian vault directory
export TSUB_VAULT_DIR="~/Documents/Obsidian/Daily"
./tsub

# Example 2: Use an absolute path
export TSUB_VAULT_DIR="/mnt/storage/ObsidianVault/Daily"
./tsub
```

## Workflow

### 1. Starting Up
When you run `./tsub`, the application opens in **View Mode**. You will see your posts for the current day (if any).

### 2. Entering Edit Mode
- Press **`i`** (Insert) to switch to **Edit Mode**.
- Your cursor will move to the input box.
- **Press `Tab`** to start typing your micro-post. The timestamp and prefix will be added automatically.
- Type your content (you can use newlines).
- Press **`Ctrl+C`** (or `Esc` then `Ctrl+C`) to finish editing and save the post.

### 3. Using an External Editor
- In **Edit Mode**, press **`Ctrl+E`**.
- Your default text editor (or `$EDITOR`) will open with your current post content.
- Make your edits and save the file.
- Close the editor.
- `tsub` will automatically detect the changes and reload the preview.

### 4. Switching Modes
- Use **`v`** (View) to switch back to the live preview when you are done editing.
- Use **`i`** (Insert) to start editing again.

## Keyboard Shortcuts

| Key | Mode | Action |
|-----|------|--------|
| **`i`** | View → Edit | Start typing or editing a new post. |
| **`v`** | Edit → View | Switch back to viewing the timeline. |
| **`Tab`** | Edit | **Start typing** your post content. |
| **`Ctrl+E`** | Edit | Open **external editor** (VS Code, Vim, etc.). |
| **`Ctrl+C`** | Edit | Finish editing and save the post. |
| **`Ctrl+C`** | View | Quit the application. |
| **`Ctrl+Z`** | View | Suspend the application (save state and return to shell). |
| **`Ctrl+S`** | View | Manually refresh and reload the timeline. |

## Contributing

We welcome contributions! Please feel free to fork the repository and open a Pull Request.

## License

This project is licensed under the MIT License.