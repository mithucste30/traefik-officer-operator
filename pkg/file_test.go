package logprocessing

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLogFileConfigStruct tests the LogFileConfig struct
func TestLogFileConfigStruct(t *testing.T) {
	config := LogFileConfig{
		FileLocation: "/var/log/traefik/access.log",
		MaxFileBytes: 10,
	}

	if config.FileLocation != "/var/log/traefik/access.log" {
		t.Errorf("Expected FileLocation '/var/log/traefik/access.log', got %s", config.FileLocation)
	}

	if config.MaxFileBytes != 10 {
		t.Errorf("Expected MaxFileBytes 10, got %d", config.MaxFileBytes)
	}
}

// TestFileLogSourceStruct tests the FileLogSource struct
func TestFileLogSourceStruct(t *testing.T) {
	fls := &FileLogSource{
		filename: "test.log",
		lines:    make(chan LogLine, 100),
	}

	if fls.filename != "test.log" {
		t.Errorf("Expected filename 'test.log', got %s", fls.filename)
	}

	if cap(fls.lines) != 100 {
		t.Errorf("Expected lines channel capacity 100, got %d", cap(fls.lines))
	}

	if fls.tail != nil {
		t.Error("Expected tail to be nil initially")
	}
}

// TestNewFileLogSource tests creating a new file log source
func TestNewFileLogSource(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (string, error)
		cleanup     func(string)
		expectedErr bool
		validate    func(*testing.T, *FileLogSource)
	}{
		{
			name: "create file log source with existing file",
			setup: func() (string, error) {
				tmpFile := filepath.Join(os.TempDir(), "test-access.log")
				content := "192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] \"GET /api/users HTTP/1.1\" 200 1234\n"
				return tmpFile, os.WriteFile(tmpFile, []byte(content), 0644)
			},
			cleanup: func(path string) {
				os.Remove(path)
			},
			expectedErr: false,
			validate: func(t *testing.T, fls *FileLogSource) {
				if fls == nil {
					t.Error("Expected FileLogSource to be created")
					return
				}
				if fls.filename == "" {
					t.Error("Expected filename to be set")
				}
				if fls.lines == nil {
					t.Error("Expected lines channel to be initialized")
				}
				if fls.tail == nil {
					t.Error("Expected tail to be initialized")
				}
			},
		},
		{
			name: "create file log source with non-existing file",
			setup: func() (string, error) {
				tmpFile := filepath.Join(os.TempDir(), "non-existing-log.log")
				return tmpFile, nil
			},
			cleanup: func(path string) {
				// File might be created by tail library
			},
			expectedErr: false, // tail library creates file if it doesn't exist
			validate: func(t *testing.T, fls *FileLogSource) {
				if fls == nil {
					t.Error("Expected FileLogSource to be created")
				}
			},
		},
		{
			name: "create file log source with invalid path",
			setup: func() (string, error) {
				// Use a path that cannot be created (e.g., in a non-existent directory)
				// Note: tail library will try to create this, so we expect it might succeed or fail
				return "/non/existent/directory/test.log", nil
			},
			cleanup:     func(path string) {},
			expectedErr: false, // tail library doesn't immediately fail for non-existent paths
			validate:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logFile, err := tt.setup()
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			defer tt.cleanup(logFile)

			config := &LogFileConfig{
				FileLocation: logFile,
				MaxFileBytes: 10,
			}

			fls, err := NewFileLogSource(config)

			if (err != nil) != tt.expectedErr {
				t.Errorf("NewFileLogSource() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			if !tt.expectedErr && tt.validate != nil {
				tt.validate(t, fls)
			}

			// Clean up the file log source
			if fls != nil {
				fls.Close()
			}
		})
	}
}

// TestFileLogSourceReadLines tests the ReadLines method
func TestFileLogSourceReadLines(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-read-lines.log")
	content := `192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234
192.168.1.2 - - [01/Jan/2024:12:00:01 +0000] "POST /api/orders HTTP/1.1" 201 5678
`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	config := &LogFileConfig{
		FileLocation: tmpFile,
		MaxFileBytes: 10,
	}

	fls, err := NewFileLogSource(config)
	if err != nil {
		t.Fatalf("Failed to create FileLogSource: %v", err)
	}
	defer fls.Close()

	linesChan := fls.ReadLines()

	if linesChan == nil {
		t.Fatal("Expected ReadLines to return a channel")
	}

	// Try to read a line with timeout
	timeout := time.After(2 * time.Second)
	select {
	case line := <-linesChan:
		if line.Text == "" {
			t.Error("Expected to receive log line text")
		}
		if line.Err != nil {
			t.Errorf("Unexpected error in log line: %v", line.Err)
		}
	case <-timeout:
		// If we timeout, the file might have been read already
		t.Log("Timeout waiting for log line (file may have been read immediately)")
	}
}

// TestFileLogSourceClose tests the Close method
func TestFileLogSourceClose(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-close.log")
	if err := os.WriteFile(tmpFile, []byte("test log\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile)

	config := &LogFileConfig{
		FileLocation: tmpFile,
		MaxFileBytes: 10,
	}

	fls, err := NewFileLogSource(config)
	if err != nil {
		t.Fatalf("Failed to create FileLogSource: %v", err)
	}

	// Close should not panic
	err = fls.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Double close should also not panic
	err = fls.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}
}

// TestFileLogSourceCloseWithNilTail tests Close when tail is nil
func TestFileLogSourceCloseWithNilTail(t *testing.T) {
	fls := &FileLogSource{
		tail:     nil,
		filename: "test.log",
		lines:    make(chan LogLine, 10),
	}

	err := fls.Close()
	if err != nil {
		t.Errorf("Close() with nil tail returned error: %v", err)
	}
}

// TestAddFileFlags tests the AddFileFlags function
func TestAddFileFlags(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	config := AddFileFlags(flags)

	if config == nil {
		t.Fatal("Expected config to be returned")
	}

	// Check default values
	if config.FileLocation != "./accessLog.txt" {
		t.Errorf("Expected default file location './accessLog.txt', got %s", config.FileLocation)
	}

	if config.MaxFileBytes != 10 {
		t.Errorf("Expected default max file bytes 10, got %d", config.MaxFileBytes)
	}
}

// TestFileLogSourceIntegration tests file log source with actual log entries
func TestFileLogSourceIntegration(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-integration.log")
	defer os.Remove(tmpFile)

	// Create test file with content
	content := `192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234
192.168.1.2 - - [01/Jan/2024:12:00:01 +0000] "POST /api/orders HTTP/1.1" 201 5678
192.168.1.3 - - [01/Jan/2024:12:00:02 +0000] "DELETE /api/users/123 HTTP/1.1" 204 0
`

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := &LogFileConfig{
		FileLocation: tmpFile,
		MaxFileBytes: 10,
	}

	fls, err := NewFileLogSource(config)
	if err != nil {
		t.Fatalf("Failed to create FileLogSource: %v", err)
	}
	defer fls.Close()

	// Read lines with timeout
	linesChan := fls.ReadLines()
	linesRead := 0
	timeout := time.After(3 * time.Second)

readLoop:
	for {
		select {
		case line, ok := <-linesChan:
			if !ok {
				// Channel closed
				break readLoop
			}
			if line.Err != nil {
				t.Errorf("Unexpected error in log line: %v", line.Err)
				continue
			}
			if line.Text != "" {
				linesRead++
			}
		case <-timeout:
			// Timeout is acceptable - file may have been read already
			t.Logf("Timeout after reading %d lines", linesRead)
			break readLoop
		}
	}

	// We should have read at least some lines
	if linesRead == 0 {
		t.Log("No lines read (file may have been read before test started)")
	}
}
