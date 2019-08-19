package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

func TestCLIInvalidCommand(t *testing.T) {
	// Run the crashing code when FLAG is set
	if os.Getenv("FLAG") == "1" {
		msg, display, err := startCLI([]string{"sn-dotfiles", "lemon"})
		fmt.Println(msg, display, err)
		return
	}
	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestCLIInvalidCommand")
	cmd.Env = append(os.Environ(), "FLAG=1")
	err := cmd.Run()

	// Cast the error as *exec.ExitError and compare the result
	e, ok := err.(*exec.ExitError)
	expectedErrorString := "exit status 1"
	assert.Equal(t, true, ok)
	assert.Equal(t, expectedErrorString, e.Error())
}
