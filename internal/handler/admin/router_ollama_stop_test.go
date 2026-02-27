package admin

import "testing"

func TestGetOllamaStopCommand_SupportedOS(t *testing.T) {
	command, err := getOllamaStopCommand("darwin")
	if err != nil {
		t.Fatalf("darwin should be supported: %v", err)
	}
	if command == "" {
		t.Fatalf("darwin stop command should not be empty")
	}

	command, err = getOllamaStopCommand("linux")
	if err != nil {
		t.Fatalf("linux should be supported: %v", err)
	}
	if command == "" {
		t.Fatalf("linux stop command should not be empty")
	}
}

func TestGetOllamaStopCommand_UnsupportedOS(t *testing.T) {
	_, err := getOllamaStopCommand("windows")
	if err == nil {
		t.Fatalf("windows should be unsupported")
	}
}
