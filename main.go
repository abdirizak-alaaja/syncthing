package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/sqweek/dialog"
)

func main() {
	// Create a custom logger with color
	errorLog := log.New(os.Stderr, color.RedString("ERROR: "), log.Ldate|log.Ltime)

	opts := UploadThingOpts{
		// Configuration, do not change.
		Bucket:          "vrc",
		Region:          "auto",
		Endpoint:        "https://a9f0148f5973d8d81d95157fb58d7316.r2.cloudflarestorage.com",
		AccessKeyID:     "79713fda41dae7b930d26c6732711746",
		SecretAccessKey: "1f870e42dcdb99db7ae0b72cba8cf6b128feac4224c67ab9f8fdd699434631ee",
	}

	// Try to load previous sync data
	previousSync, err := LoadSyncData()
	if err != nil {
		errorLog.Printf("Failed to load previous sync data: %v", err)
	}

	// Mode selection
	var mode string
	modeOptions := []huh.Option[string]{
		huh.NewOption("Host (Upload)", "host"),
		huh.NewOption("Sync (Download)", "sync"),
	}

	// If we have previous sync data, add a resync option
	if previousSync != nil {
		modeOptions = append([]huh.Option[string]{
			huh.NewOption(fmt.Sprintf("Resync using last key: %s", previousSync.LastKey), "resync"),
		}, modeOptions...)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(color.HiCyanString("Select Mode")).
				Options(modeOptions...).
				Value(&mode),
		),
	)

	err = form.Run()
	if err != nil {
		errorLog.Fatalf("Mode selection failed: %v", err)
	}

	if mode == "resync" && previousSync != nil {
		// Set mode based on previous operation
		opts.Mode = previousSync.LastOperationMode
		opts.Key = previousSync.LastKey
		if opts.Mode == "host" {
			opts.SourcePath = previousSync.LastSourcePath
			opts.UpdateExisting = true // When rehosting, we want to update the existing key
		} else {
			opts.DestinationPath = previousSync.LastDestPath
		}
	} else {
		opts.Mode = mode
		opts.Key = GenerateThing()

		if mode == "host" {
			result, err := dialog.Directory().Browse()
			if err != nil {
				errorLog.Fatalf("File selection failed: %v", err)
			}
			opts.SourcePath = result

			// Ask if updating existing
			var updateExisting bool
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(color.HiCyanString("Update existing key?")).
						Value(&updateExisting),
				),
			)

			err = form.Run()
			if err != nil {
				errorLog.Fatalf("Update choice failed: %v", err)
			}
			opts.UpdateExisting = updateExisting

			if updateExisting {
				var key string
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title(color.HiCyanString("Enter existing key")).
							Value(&key),
					),
				)

				err = form.Run()
				if err != nil {
					errorLog.Fatalf("Key input failed: %v", err)
				}
				opts.Key = key
			}

		} else {
			// Get sync key
			var Key string
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title(color.HiCyanString("Enter sync key")).
						Value(&Key),
				),
			)

			err = form.Run()
			if err != nil {
				errorLog.Fatalf("Sync key input failed: %v", err)
			}
			opts.Key = Key

			// Directory selection for download
			result, err := dialog.Directory().Browse()
			if err != nil {
				errorLog.Fatalf("Folder selection failed: %v", err)
			}
			opts.DestinationPath = result
		}
	}

	ctx := context.Background()
	s3Client, err := NewS3Client(ctx, opts)
	if err != nil {
		errorLog.Fatalf("Failed to create S3 client: %v", err)
	}

	switch opts.Mode {
	case "host":
		err = s3Client.Host(ctx, opts.SourcePath, SyncBucketRoot+opts.Key, opts.UpdateExisting)
	case "sync":
		err = s3Client.Sync(ctx, SyncBucketRoot+opts.Key, opts.DestinationPath)
	default:
		errorLog.Fatalf("Unknown mode: %s", opts.Mode)
	}

	if err != nil {
		errorLog.Fatalf("Error: %v", err)
	}

	// Save sync data after successful operation
	syncData := SyncData{
		LastKey:           opts.Key,
		LastSourcePath:    opts.SourcePath,
		LastDestPath:      opts.DestinationPath,
		LastSyncTime:      time.Now(),
		LastOperationMode: opts.Mode,
	}
	if err := SaveSyncData(syncData); err != nil {
		errorLog.Printf("Failed to save sync data: %v", err)
	}

	color.Green("\n✓ Operation complete")
	if opts.Mode == "host" {
		fmt.Printf("%s %s\n", color.HiCyanString("Sync Key:"), color.HiGreenString(opts.Key))
	}
}