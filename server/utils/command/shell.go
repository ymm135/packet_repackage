package command

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GoLinuxShell executes a shell command and returns error if any
func GoLinuxShell(cmdParts ...string) error {
	_, err := executeCommand(cmdParts)
	return err
}

// GoLinuxShellWithResult executes a shell command and returns output and error
func GoLinuxShellWithResult(cmdParts ...string) (string, error) {
	return executeCommand(cmdParts)
}

func executeCommand(cmdParts []string) (string, error) {
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmdStr := strings.Join(cmdParts, " ")
	cmd := exec.Command("sh", "-c", cmdStr)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	output := stdout.String()
	if err != nil {
		errOutput := stderr.String()
		if errOutput != "" {
			return output, fmt.Errorf("%s: %s", err.Error(), errOutput)
		}
		return output, err
	}

	return output, nil
}
