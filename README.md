# openconnect-tui

[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://go.dev/)[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A powerful, feature-rich **TUI** (Terminal User Interface) wrapper for [OpenConnect](https://www.infradead.org/openconnect/). 

Effortlessly manage VPN profiles and connections while retaining full access to OpenConnect's capabilities.

## Features

*   **Profile Management:** Create, edit, and select saved VPN profiles.
*   **IP Resolution:** Resolve domains using built-in lookup tools to choose specific target IPs for connection.
*   **Flag Configuration:** Visual interface to configure standard OpenConnect command-line flags.
*   **Native Binary Support:** Interacts directly with your local `openconnect` binary.
*   **Keyboard-driven UI:** Built using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).
*   **Cross-Platform Support:** Currently supports Windows, with Linux compatibility coming very soon.

## Prerequisites

This application requires the official `openconnect` binary to be installed on your system and available in your system path.

### Windows

1.  **Install OpenConnect:**
    *   The recommended method is installing via [openconnect-gui](https://gitlab.com/openconnect/openconnect-gui/-/releases).
    *   *Important:* During installation, ensure you select **Add openconnect-gui to the system PATH** and include the **console** components.
    *   Alternatively, you can build the binary from the official [source](https://gitlab.com/openconnect/openconnect). If building manually, ensure `openconnect.exe` is added to your PATH and the appropriate [`vpnc-script.js`](https://gitlab.com/openconnect/vpnc-scripts) is located in the same directory as the executable.
2.  Verify that `openconnect` is accessible from your command line.

### Linux

*Support is currently in development. coming soon*

## Installation and Usage

### Windows

#### Pre-built Binaries
Download the latest executable from the [releases page](https://github.com/bakhhag/openconnect-tui/releases).

#### Building from Source
Ensure you have Go installed, then clone and build the project:

```bash
git clone https://github.com/bakhhag/openconnect-tui
cd openconnect-tui
go build -ldflags="-s -w" -o bin/openconnect-tui.exe
```

*Note: Because OpenConnect requires administrative privileges to modify system network routes, the application will prompt for administrator elevation when launched.*

### Customizing Flags

Flags can be added or removed by modifying the `flags.csv` configuration file located in:
`%APPDATA%\OpenConnect-TUI\flags.csv`

The CSV file uses the following structure:
```csv
flag,selected,value
no-dtls,1,
disable-ipv6,0,
no-xmlpost,0,
```
#### Configuration Rules:

*   **Standard Flags (No values):** For flags that do not require an argument, set the `selected` column to `1` (enabled) or `0` (disabled), and leave the `value` column empty.
*   **Value-Based Flags:** For flags that require an argument, append an equals sign `=` to the flag name in the `flag` column (e.g., `proxy=`). You can predefine the argument in the `value` column or leave it blank to enter it later.

*Note: Native flag management from within the TUI interface is planned for a future release.*

## Keyboard Shortcuts 

### Global Navigation
*   `↑ / ↓` (Arrow Keys): Navigate menus and lists
*   `Tab`: Switch between application tabs
*   `?`: Toggle the advanced key help overlay
*   `Q`: Exit the application

### Connection Tab
*   `Enter`: Initiate or terminate the connection
*   `A`: Save the current connection configuration to your profiles

### Profiles Tab
*   `A`: Create a new profile
*   `E`: Edit the selected profile
*   `D`: Delete the selected profile

## Roadmap

- [ ] Complete Linux compatibility
- [ ] Refine UI components and layouts
- [ ] Customzing Flags from TUI

## Contributing

Contributions, bug reports, and pull requests are welcome. Please ensure any contributions align with the project's structure and existing formatting.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

Made with ❤️ using Go + Bubble Tea + Lipgloss
