// +build windows

package logprocessing

import (
	"errors"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"os"
	"syscall"
)

// logRotate is not supported on Windows
func logRotate(accessLogLocation string) error {
	return errors.New("log rotation is not supported on Windows")
}

// findTraefikProcess is not supported on Windows
func findTraefikProcess() (int, error) {
	return -1, errors.New("process finding is not supported on Windows")
}

// Signal is not available on Windows in the same way
func killProcess(pid int) error {
	// Use Windows-specific process termination
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}
	return process.Signal(syscall.SIGTERM)
}

func deleteFile(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Debugf("File %s does not exist, nothing to delete", path)
		return nil
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	logger.Debugf("Successfully deleted file: %s", path)
	return nil
}

func createFile(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		logger.Debugf("File %s already exists", path)
		return nil
	}

	// Create the file with appropriate permissions
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		logger.Warnf("Error closing file %s: %v", path, err)
	}

	logger.Infof("Created file: %s", path)
	return nil
}
