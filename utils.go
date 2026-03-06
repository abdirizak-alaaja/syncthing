package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh/spinner"
	"github.com/fatih/color"
)

// logInfo prints an info message in a consistent color format
func LogInfo(format string, args ...interface{}) {
	prefix := color.HiBlueString("INFO:")
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", prefix, color.WhiteString(message))
}

// logPath formats a path operation with source and destination
func LogPath(operation, source, destination string) {
	prefix := color.HiBlueString("INFO:")
	op := color.HiMagentaString(" %s", operation)
	src := color.HiWhiteString(" %s", source)
	arrow := color.HiBlueString(" → ")
	dest := color.CyanString("%s", destination)
	fmt.Printf("%s%s%s%s%s\n", prefix, op, src, arrow, dest)
}

// generateRandomKey returns a random string of letters of the given length.
func GenerateRandomKey(length int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		idx := rand.Intn(len(letters))
		b[i] = letters[idx]
	}
	return string(b)
}

// getRandomSpinner returns a random spinner type
func GetRandomSpinner() spinner.Type {
	spins := []spinner.Type{
		spinner.Moon,
		spinner.Line,
		spinner.Dots,
		spinner.Jump,
		spinner.Points,
		spinner.Pulse,
	}
	return spins[rand.Intn(len(spins))]
}

// getSyncFilePath returns the path to the .syncthing file in the user's home directory
func getSyncFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".syncthing"), nil
}

// saveSyncData saves the sync information to the .syncthing file
func SaveSyncData(data SyncData) error {
	path, err := getSyncFilePath()
	if err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync data: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write sync file: %w", err)
	}
	return nil
}

// loadSyncData loads the sync information from the .syncthing file
func LoadSyncData() (*SyncData, error) {
	path, err := getSyncFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read sync file: %w", err)
	}

	var syncData SyncData
	if err := json.Unmarshal(data, &syncData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sync data: %w", err)
	}
	return &syncData, nil
}
