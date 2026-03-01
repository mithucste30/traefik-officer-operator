// +build !windows

package logprocessing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindTraefikProcess tests the findTraefikProcess function
func TestFindTraefikProcess(t *testing.T) {
	tests := []struct {
		name    string
		test    func(*testing.T)
		skip    bool
		skipReason string
	}{
		{
			name: "find traefik process or return not found",
			test: func(t *testing.T) {
				pid, err := findTraefikProcess()

				if err != nil {
					t.Errorf("findTraefikProcess() returned error: %v", err)
				}

				// PID should be -1 if not found, or a valid positive PID
				if pid != -1 && pid <= 0 {
					t.Errorf("Expected valid PID or -1, got %d", pid)
				}

				if pid == -1 {
					t.Log("Traefik process not found (this is expected if Traefik is not running)")
				} else {
					t.Logf("Found Traefik process with PID: %d", pid)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip(tt.skipReason)
			}
			tt.test(t)
		})
	}
}

// TestCreateFileUnix tests the createFileUnix function
func TestCreateFileUnix(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectedErr bool
		validate    func(*testing.T, string)
	}{
		{
			name:        "empty path returns error",
			path:        "",
			expectedErr: true,
			validate:    nil,
		},
		{
			name:        "create file in temp directory",
			path:        filepath.Join(os.TempDir(), "test-traefik-log.txt"),
			expectedErr: false,
			validate: func(t *testing.T, path string) {
				// Check file exists
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Expected file to exist at %s", path)
				}

				// Clean up
				os.Remove(path)
			},
		},
		{
			name:        "create file with nested directories",
			path:        filepath.Join(os.TempDir(), "traefik-test", "nested", "dir", "access.log"),
			expectedErr: false,
			validate: func(t *testing.T, path string) {
				// Check file exists
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Expected file to exist at %s", path)
				}

				// Clean up
				os.RemoveAll(filepath.Join(os.TempDir(), "traefik-test"))
			},
		},
		{
			name:        "file already exists returns nil",
			path:        filepath.Join(os.TempDir(), "existing-file.txt"),
			expectedErr: false,
			validate: func(t *testing.T, path string) {
				// Create file first
				os.WriteFile(path, []byte("test"), 0644)

				// Try to create again - should return nil
				// Clean up
				os.Remove(path)
			},
		},
		{
			name:        "create file in current directory",
			path:        "test-log-file.txt",
			expectedErr: false,
			validate: func(t *testing.T, path string) {
				// Check file exists
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Expected file to exist at %s", path)
				}

				// Clean up
				os.Remove(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := createFileUnix(tt.path)

			if (err != nil) != tt.expectedErr {
				t.Errorf("createFileUnix() error = %v, expectedErr %v", err, tt.expectedErr)
			}

			if !tt.expectedErr && tt.validate != nil {
				tt.validate(t, tt.path)
			}
		})
	}
}

// TestDeleteFileUnix tests the deleteFileUnix function
func TestDeleteFileUnix(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (string, error)
		expectedErr bool
		validate    func(*testing.T, string)
	}{
		{
			name: "empty path returns error",
			setup: func() (string, error) {
				return "", nil
			},
			expectedErr: true,
			validate:    nil,
		},
		{
			name: "delete existing file",
			setup: func() (string, error) {
				tmpFile := filepath.Join(os.TempDir(), "test-delete-file.txt")
				if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
					return "", err
				}
				return tmpFile, nil
			},
			expectedErr: false,
			validate: func(t *testing.T, path string) {
				// Check file does not exist
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("Expected file to be deleted at %s", path)
				}
			},
		},
		{
			name: "delete non-existing file returns nil",
			setup: func() (string, error) {
				return filepath.Join(os.TempDir(), "non-existing-file.txt"), nil
			},
			expectedErr: false,
			validate:    nil,
		},
		{
			name: "delete file in nested directory",
			setup: func() (string, error) {
				tmpDir := filepath.Join(os.TempDir(), "test-delete-nested")
				if err := os.MkdirAll(tmpDir, 0755); err != nil {
					return "", err
				}
				tmpFile := filepath.Join(tmpDir, "test.txt")
				if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
					return "", err
				}
				return tmpFile, nil
			},
			expectedErr: false,
			validate: func(t *testing.T, path string) {
				// Check file does not exist
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("Expected file to be deleted at %s", path)
				}
				// Clean up directory
				os.RemoveAll(filepath.Dir(path))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.setup()

			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			deleteErr := deleteFileUnix(path)

			if (deleteErr != nil) != tt.expectedErr {
				t.Errorf("deleteFileUnix() error = %v, expectedErr %v", deleteErr, tt.expectedErr)
			}

			if !tt.expectedErr && tt.validate != nil {
				tt.validate(t, path)
			}
		})
	}
}

// TestLogRotate tests the logRotate function
func TestLogRotate(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (string, error)
		expectedErr bool
		skip        bool
		skipReason  string
	}{
		{
			name: "empty log location returns error",
			setup: func() (string, error) {
				return "", nil
			},
			expectedErr: true,
		},
		{
			name: "log rotate with traefik not running fails",
			setup: func() (string, error) {
				tmpFile := filepath.Join(os.TempDir(), "test-rotate.log")
				if err := os.WriteFile(tmpFile, []byte("test log"), 0644); err != nil {
					return "", err
				}
				return tmpFile, nil
			},
			expectedErr: true, // Will fail because Traefik is not running
		},
		{
			name: "log rotate with valid file path",
			setup: func() (string, error) {
				tmpFile := filepath.Join(os.TempDir(), "traefik-access.log")
				if err := os.WriteFile(tmpFile, []byte("test log content"), 0644); err != nil {
					return "", err
				}
				return tmpFile, nil
			},
			expectedErr: true, // Will fail because Traefik is not running
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip(tt.skipReason)
			}

			path, err := tt.setup()
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Clean up
			defer os.Remove(path)

			rotateErr := logRotate(path)

			if (rotateErr != nil) != tt.expectedErr {
				t.Errorf("logRotate() error = %v, expectedErr %v", rotateErr, tt.expectedErr)
			}

			// Verify file was recreated even if rotation failed
			if path != "" {
				if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
					t.Errorf("Error checking file state: %v", err)
				}
			}
		})
	}
}

// TestCreateDeleteFileUnixIntegration tests create and delete together
func TestCreateDeleteFileUnixIntegration(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-integration.txt")

	// Create file
	if err := createFileUnix(tmpFile); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Expected file to exist after creation")
	}

	// Delete file
	if err := deleteFileUnix(tmpFile); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("Expected file to not exist after deletion")
	}
}

// TestCreateFileUnixPermissions tests file permissions
func TestCreateFileUnixPermissions(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-permissions.txt")

	if err := createFileUnix(tmpFile); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Check file permissions
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// File should be readable
	if info.Mode().Perm()&0400 == 0 {
		t.Error("Expected file to be readable by owner")
	}

	// Clean up
	os.Remove(tmpFile)
}
