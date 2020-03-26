package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gruntwork-io/gruntwork-cli/errors"
)

// Run the specified shell command with the specified arguments. Connect the command's stdin, stdout, and stderr to
// the currently running app.
func RunShellCommand(command Command) error {
	if command.SensitiveArgs {
		command.Logger.Infof("Running command: %s (args redacted)", command.Command)
	} else {
		command.Logger.Infof("Running command: %s %s", command.Command, strings.Join(command.Args, " "))
	}

	cmd := exec.Command(command.Command, command.Args...)

	// TODO: consider logging this via options.Logger
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Dir = command.WorkingDir

	if len(command.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range command.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return errors.WithStackTrace(cmd.Run())
}

// Run the specified shell command with the specified arguments. Return its stdout and stderr as a string
func RunShellCommandAndGetOutput(command Command) (string, error) {
	if command.SensitiveArgs {
		command.Logger.Infof("Running command: %s (args redacted)", command.Command)
	} else {
		command.Logger.Infof("Running command: %s %s", command.Command, strings.Join(command.Args, " "))
	}

	cmd := exec.Command(command.Command, command.Args...)

	cmd.Stdin = os.Stdin
	cmd.Dir = command.WorkingDir

	if len(command.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range command.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	out, err := cmd.CombinedOutput()
	return string(out), errors.WithStackTrace(err)
}

func KeysStringString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

// Run the specified shell command with the specified arguments. Return its stdout and stderr as a string and also
// stream stdout and stderr to the OS stdout/stderr
func RunShellCommandAndGetAndStreamOutput(command Command) (string, error) {
	if command.SensitiveArgs {
		command.Logger.Infof("Running command: %s (args redacted)", command.Command)
	} else {
		command.Logger.Infof("Running command: %s %s. EnvVars: %s", command.Command, strings.Join(command.Args, " "), KeysStringString(command.Env))
	}

	cmd := exec.Command(command.Command, command.Args...)

	if len(command.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range command.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmd.Dir = command.WorkingDir

	cmd.Stdin = os.Stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	if err := cmd.Start(); err != nil {
		return "", errors.WithStackTrace(err)
	}

	output, err := readStdoutAndStderr(stdout, stderr, command)
	if err != nil {
		return output, err
	}

	err = cmd.Wait()

	return output, errors.WithStackTrace(err)
}

// This function captures stdout and stderr while still printing it to the stdout and stderr of this Go program
func readStdoutAndStderr(stdout io.ReadCloser, stderr io.ReadCloser, command Command) (string, error) {
	allOutput := []string{}
	stderrOutput := []string{}

	// Ensure we can scan lines up to 1MB
	// This value is arbitrary at this point.
	// May want to look into a cleaner way for this.
	stdoutScanner := bufio.NewScanner(stdout)
	stdoutBuf := make([]byte, 0, 64*1024)
	stdoutScanner.Buffer(stdoutBuf, 1024*1024)
	stderrScanner := bufio.NewScanner(stderr)
	stderrBuf := make([]byte, 0, 64*1024)
	stderrScanner.Buffer(stderrBuf, 1024*1024)

	for {
		if stdoutScanner.Scan() {
			text := stdoutScanner.Text()
			command.Logger.Println(text)
			allOutput = append(allOutput, text)
		} else if stderrScanner.Scan() {
			text := stderrScanner.Text()
			command.Logger.Errorln(text)
			stderrOutput = append(stderrOutput, text)
			allOutput = append(allOutput, text)
		} else {
			break
		}
	}

	if err := stdoutScanner.Err(); err != nil {
		return "", errors.WithStackTrace(err)
	}

	if err := stderrScanner.Err(); err != nil {
		return "", errors.WithStackTrace(fmt.Errorf("%v: %s", err, stderrOutput))
	}

	return strings.Join(allOutput, "\n"), nil
}

// Return true if the OS has the given command installed
func CommandInstalled(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// CommandInstalledE returns an error if command is not installed
func CommandInstalledE(command string) error {
	if commandExists := CommandInstalled(command); !commandExists {
		err := fmt.Errorf("Command %s is not installed", command)
		return errors.WithStackTrace(err)
	}
	return nil
}
