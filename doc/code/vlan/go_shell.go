package command

import (
	"fmt"
	"github.com/progrium/go-shell"
	"strings"
)

func GoLinuxShell(cmd ...string) error {
	_, err := sh(cmd)
	return err
}

func GoLinuxShellWithResult(cmd ...string) (string, error) {
	defer handlerErr()
	process, err := sh(cmd)
	return process.String(), err
}

func sh(cmd []string) (*shell.Process, error) {
	shell.Panic = false
	shell.Trace = true

	cmdStr := strings.Join(cmd, " ")
	defer handlerErr()
	process := shell.Cmd(cmdStr).Run()
	code := process.ExitStatus
	fmt.Println(process, ",exitstatus:", code)
	var err error
	if code > 0 {
		err = process.Error()
	}
	return process, err
}

func handlerErr() {
	if err := recover(); err != nil {
		fmt.Println("GoLinuxShell error", err)
	}
}
