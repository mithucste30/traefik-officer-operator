// +build !windows

package logprocessing

import (
	"errors"
	"fmt"
	ps "github.com/mitchellh/go-ps"
	logger "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"syscall"
)

func logRotate(accessLogLocation string) error {
	if accessLogLocation == "" {
		return errors.New("access log location cannot be empty")
	}

	// Get the Traefik process
	traefikPid, err := findTraefikProcess()
	if err != nil {
		return fmt.Errorf("failed to find Traefik process: %w", err)
	}

	if traefikPid == -1 {
		return errors.New("traefik process not found")
	}

	logger.Infof("Found Traefik process @ PID %d", traefikPid)

	traefikProcess, err := os.FindProcess(traefikPid)
	if err != nil {
		return fmt.Errorf("failed to find process with PID %d: %w", traefikPid, err)
	}

	// Delete and recreate the log file
	if err := deleteFile(accessLogLocation); err != nil {
		return fmt.Errorf("failed to delete log file: %w", err)
	}

	if err := createFile(accessLogLocation); err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	// Send USR1 signal to Traefik to reopen log files
	if err := traefikProcess.Signal(syscall.SIGUSR1); err != nil {
		return fmt.Errorf("failed to send SIGUSR1 to Traefik process: %w", err)
	}

	logger.Info("Successfully rotated log file and signaled Traefik")
	return nil
}

// findTraefikProcess finds the Traefik process and returns its PID
func findTraefikProcess() (int, error) {
	processList, err := ps.Processes()
	if err != nil {
		return -1, fmt.Errorf("failed to list processes: %w", err)
	}

	for _, process := range processList {
		if process.Executable() == "traefik" {
			return process.Pid(), nil
		}
	}

	return -1, nil
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

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create the file with appropriate permissions (read/write for owner, read for others)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		logger.Warnf("Error closing file %s: %v", path, err)
	}

	logger.Infof("Created file: %s", path)
	return nil
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
