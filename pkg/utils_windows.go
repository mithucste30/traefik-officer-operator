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

func createFileUnix(path string) error {
	return errors.New("createFileUnix is not supported on Windows")
}

func deleteFileUnix(path string) error {
	return errors.New("deleteFileUnix is not supported on Windows")
}
