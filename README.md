# Syncthing

A simple and secure file synchronization tool that allows you to host and sync files using Cloudflare R2 storage.

## Features

- **Host Mode**: Upload files/folders and get a sync key
- **Sync Mode**: Download files/folders using a sync key
- **Resync**: Quickly resync previously synced files
- **User-friendly Interface**: Interactive CLI with color-coded messages
- **File History**: Keeps track of your previous sync operations

## Installation

1. Make sure you have Go installed on your system
2. Clone this repository
3. Run `go mod download` to install dependencies
4. Build the project with `go build`

## Usage

Run the executable and follow the interactive prompts. The tool offers three main modes:

### Host Mode (Upload)
1. Select "Host (Upload)"
2. Choose the folder you want to upload
3. Choose whether to:
   - Create a new sync key (default)
   - Update an existing sync key
4. Share the sync key with others who need access to your files

### Sync Mode (Download)
1. Select "Sync (Download)"
2. Enter the sync key you received
3. Choose the destination folder for downloaded files
4. Wait for the sync to complete

### Resync
- If you've previously synced files, you'll see a "Resync" option
- This allows you to quickly sync using your last configuration

## Dependencies

- github.com/charmbracelet/huh - For interactive CLI forms
- github.com/fatih/color - For colored output
- github.com/sqweek/dialog - For native file dialogs

## Security

- Files are stored securely in Cloudflare R2 storage
- Sync keys are randomly generated and unique
- No authentication required - security through key obscurity

## Notes

- The tool remembers your last sync operation for convenience
- Progress is displayed in real-time with color-coded status messages
- Errors are clearly displayed in red for easy troubleshooting
