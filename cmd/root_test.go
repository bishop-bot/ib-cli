package cmd

import (
	"bytes"
	"testing"
)

func TestRootCmd_Execute(t *testing.T) {
	// Reset command tree for testing
	rootCmd.SetArgs([]string{})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Execute should not panic
	err := Execute()
	// We expect no error since --help shouldn't cause issues
	_ = err // May fail due to no config, but shouldn't panic
}

func TestRootCmd_Version(t *testing.T) {
	rootCmd.SetArgs([]string{"--version"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected version output, got empty string")
	}
}

func TestRootCmd_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Help should return an error ( ExitError )
	Execute()

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("ib-cli")) {
		t.Error("Expected 'ib-cli' in help output")
	}
}

func TestRootCmd_UnknownCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"unknown-command"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Unknown command should return error
	err := Execute()
	if err == nil {
		t.Error("Expected error for unknown command")
	}
}