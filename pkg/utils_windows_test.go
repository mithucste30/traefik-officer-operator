// +build windows

package logprocessing

import (
	"testing"
)

// TestFindTraefikProcessWindows tests the Windows stub of findTraefikProcess
func TestFindTraefikProcessWindows(t *testing.T) {
	pid, err := findTraefikProcess()

	if err == nil {
		t.Error("Expected error on Windows, got nil")
	}

	if pid != -1 {
		t.Errorf("Expected PID = -1 on Windows, got %d", pid)
	}

	expectedErrMsg := "process finding is not supported on Windows"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestLogRotateWindows tests the Windows stub of logRotate
func TestLogRotateWindows(t *testing.T) {
	err := logRotate("test-path")

	if err == nil {
		t.Error("Expected error on Windows, got nil")
	}

	expectedErrMsg := "log rotation is not supported on Windows"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestCreateFileUnixWindows tests the Windows stub of createFileUnix
func TestCreateFileUnixWindows(t *testing.T) {
	err := createFileUnix("test-path")

	if err == nil {
		t.Error("Expected error on Windows, got nil")
	}

	expectedErrMsg := "createFileUnix is not supported on Windows"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestDeleteFileUnixWindows tests the Windows stub of deleteFileUnix
func TestDeleteFileUnixWindows(t *testing.T) {
	err := deleteFileUnix("test-path")

	if err == nil {
		t.Error("Expected error on Windows, got nil")
	}

	expectedErrMsg := "deleteFileUnix is not supported on Windows"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestAllWindowsStubs tests all Windows stub functions
func TestAllWindowsStubs(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		expectedErr string
	}{
		{
			name:        "logRotate",
			fn:          func() error { return logRotate("") },
			expectedErr: "log rotation is not supported on Windows",
		},
		{
			name:        "findTraefikProcess",
			fn: func() error {
				_, err := findTraefikProcess()
				return err
			},
			expectedErr: "process finding is not supported on Windows",
		},
		{
			name:        "createFileUnix",
			fn:          func() error { return createFileUnix("") },
			expectedErr: "createFileUnix is not supported on Windows",
		},
		{
			name:        "deleteFileUnix",
			fn:          func() error { return deleteFileUnix("") },
			expectedErr: "deleteFileUnix is not supported on Windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Error("Expected error on Windows, got nil")
				return
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}
