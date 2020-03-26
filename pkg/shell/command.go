package shell

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// Command is a simpler struct for defining commands than Go's built-in Cmd.
type Command struct {
	Command           string            // The command to run
	Args              []string          // The args to pass to the command
	WorkingDir        string            // The working directory
	Env               map[string]string // Additional environment variables to set
	OutputMaxLineSize int               // The max line size of stdout and stderr (in bytes)
	Logger            *logrus.Entry
	NonInteractive    bool
	SensitiveArgs     bool // If true, will not log the arguments to the command
}

// RunCommand runs a shell command and redirects its stdout and stderr to the stdout of the atomic script itself.
func RunCommand(command Command) error {
	_, err := RunCommandAndGetOutput(command)
	return err
}

// RunCommandAndGetOutput runs a shell command and returns its stdout and stderr as a string. The stdout and stderr of that command will also
// be printed to the stdout and stderr of this Go program to make debugging easier.
func RunCommandAndGetOutput(command Command) (string, error) {
	allOutput := []string{}
	err := runCommandAndStoreOutput(command, &allOutput, &allOutput)

	output := strings.Join(allOutput, "\n")
	return output, err
}

// RunCommandAndGetStdOut runs a shell command and returns solely its stdout (but not stderr) as a string. The stdout
// and stderr of that command will also be printed to the stdout and stderr of this Go program to make debugging easier.
func RunCommandAndGetStdOut(command Command) (string, error) {
	stdout := []string{}
	stderr := []string{}
	err := runCommandAndStoreOutput(command, &stdout, &stderr)

	output := strings.Join(stdout, "\n")
	return output, err
}

// runCommandAndStoreOutput runs a shell command and stores each line from stdout and stderr in the given
// storedStdout and storedStderr variables, respectively. The stdout and stderr of that command will also
// be printed to the stdout and stderr of this Go program to make debugging easier.
func runCommandAndStoreOutput(command Command, storedStdout *[]string, storedStderr *[]string) error {

	cmd := exec.Command(command.Command, command.Args...)
	cmd.Dir = command.WorkingDir
	cmd.Stdin = os.Stdin
	cmd.Env = formatEnvVars(command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	if err := readStdoutAndStderr2(command.Logger, stdout, stderr, storedStdout, storedStderr, command.OutputMaxLineSize); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// This function captures stdout and stderr into the given variables while still printing it to the stdout and stderr
// of this Go program
func readStdoutAndStderr2(logger *logrus.Entry, stdout io.ReadCloser, stderr io.ReadCloser, storedStdout *[]string, storedStderr *[]string, maxLineSize int) error {
	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	if maxLineSize > 0 {
		stdoutScanner.Buffer(make([]byte, maxLineSize), maxLineSize)
		stderrScanner.Buffer(make([]byte, maxLineSize), maxLineSize)
	}

	wg := &sync.WaitGroup{}
	mutex := &sync.Mutex{}
	wg.Add(2)
	go readData(logger, stdoutScanner, wg, mutex, storedStdout)
	go readData(logger, stderrScanner, wg, mutex, storedStderr)
	wg.Wait()

	if err := stdoutScanner.Err(); err != nil {
		return err
	}

	if err := stderrScanner.Err(); err != nil {
		return err
	}

	return nil
}

func readData(logger *logrus.Entry, scanner *bufio.Scanner, wg *sync.WaitGroup, mutex *sync.Mutex, allOutput *[]string) {
	defer wg.Done()
	for scanner.Scan() {
		logTextAndAppendToOutput(logger, mutex, scanner.Text(), allOutput)
	}
}

func logTextAndAppendToOutput(logger *logrus.Entry, mutex *sync.Mutex, text string, allOutput *[]string) {
	defer mutex.Unlock()
	logger.Println(text)
	mutex.Lock()
	*allOutput = append(*allOutput, text)
}

// GetExitCodeForRunCommandError tries to read the exit code for the error object returned from running a shell command. This is a bit tricky to do
// in a way that works across platforms.
func GetExitCodeForRunCommandError(err error) (int, error) {
	// http://stackoverflow.com/a/10385867/483528
	if exitErr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), nil
		}
		return 1, errors.New("could not determine exit code")
	}

	return 0, nil
}

func formatEnvVars(command Command) []string {
	env := os.Environ()
	for key, value := range command.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
