package main

import (
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
	"runtime"
)

func getAppDataDir() string {
	homeDir, err := homedir.Dir()
	if err != nil {
		return ".eccd"
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = os.Getenv("APPDATA")
		}

		if appData != "" {
			return filepath.Join(appData, "Eccd")
		}
	case "darwin":
		if homeDir != "" {
			return filepath.Join(homeDir, "Library", "Application Support", "Eccd")
		}
	case "linux":
		xdgDataHome := os.Getenv("XDG_DATA_HOME")

		if xdgDataHome == "" {
			if homeDir == "" {
				return filepath.Join(homeDir, ".eccd")
			}

			xdgDataHome = filepath.Join(homeDir, ".local", "share")
		}

		return filepath.Join(xdgDataHome, "eccd")
	default:
		if homeDir != "" {
			return filepath.Join(homeDir, ".eccd")
		}
	}

	return ".eccd"
}

func getAppConfigDir() string {
	homeDir, err := homedir.Dir()
	if err != nil {
		return ".eccd"
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = os.Getenv("APPDATA")
		}

		if appData != "" {
			return filepath.Join(appData, "Eccd")
		}
	case "darwin":
		if homeDir != "" {
			return filepath.Join(homeDir, "Library", "Application Support", "Eccd")
		}
	case "linux":
		xdgConfigHome := os.Getenv("XDG_DATA_HOME")

		if xdgConfigHome == "" {
			if homeDir == "" {
				return filepath.Join(homeDir, ".eccd")
			}

			xdgConfigHome = filepath.Join(homeDir, ".config")
		}

		return filepath.Join(xdgConfigHome, "eccd")
	default:
		if homeDir != "" {
			return filepath.Join(homeDir, ".eccd")
		}
	}

	return ".eccd"
}
