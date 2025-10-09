package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var history = []string{}
var historyAppendIdx = 0

func getExecPath(command string) (string, bool) {
	envPath := os.Getenv("PATH")
	paths := strings.Split(envPath, string(os.PathListSeparator)) // : on Unix, ; on Windows

	found := false
	var execPath string
	for _, path := range paths {
		execPath = filepath.Join(path, command)
		stat, err := os.Stat(execPath)
		if os.IsNotExist(err) {
			continue
		} else if err == nil {
			// Owner 	Group	Others
			// rwx   	rwx   	rwx
			// 421		421		421
			if stat.Mode()&0111 != 0 {
				found = true
				break
			}
		} else {
			fmt.Printf("Stat error for %s: %s\n", execPath, err.Error())
		}
	}

	return execPath, found
}

func execute(pipeline [][]string, outFile *os.File, errFile *os.File) {
	var inFile io.ReadCloser
	var err error
	var cmds []*exec.Cmd

	handleBuiltinOutput := func(output string, isLastCommand bool) {
		if isLastCommand {
			fmt.Fprint(outFile, output)
		} else {
			reader, writer, err := os.Pipe()
			if err != nil {
				fmt.Fprintf(errFile, "Error creating pipe: %v\n", err)
				return
			}
			fmt.Fprint(writer, output)
			writer.Close()
			inFile = reader
		}
	}

	for i, args := range pipeline {
		command := args[0]
		isLastCommand := i == len(pipeline)-1

		switch command {
		case "exit": // assuming the tester will always pass in 0 as the argument
			os.Exit(0)

		case "echo":
			output := fmt.Sprintf("%s\n", strings.Join(args[1:], " ")) // "echo" + " "
			handleBuiltinOutput(output, isLastCommand)

		case "type":
			output := ""
			if slices.Contains(builtinCommands, args[1]) {
				output = fmt.Sprintf("%s is a shell builtin\n", args[1])
			} else {
				execPath, found := getExecPath(args[1])
				if found {
					output = fmt.Sprintf("%s is %s\n", args[1], execPath)
				} else {
					output = fmt.Sprintf("%s: not found\r\n", args[1])
				}
			}

			handleBuiltinOutput(output, isLastCommand)

		case "pwd":
			pwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(errFile, "Unable to get pwd\n")
				return
			}

			handleBuiltinOutput(pwd+"\n", isLastCommand)

		case "cd":
			if len(args) > 1 && args[1] != "~" {
				err = os.Chdir(args[1])
				if err != nil {
					fmt.Fprintf(errFile, "cd: %s: No such file or directory\r\n", args[1])
				}
			} else {
				home := os.Getenv("HOME")
				os.Chdir(home)
			}

		case "history":
			if len(args) == 3 {
				if args[1] == "-r" {
					file, err := os.Open(args[2])
					if err != nil {
						fmt.Fprintf(errFile, "Unable to open file: %s\r\n", args[2])
						continue
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						line := scanner.Text()
						history = append(history, line)
					}
				} else if args[1] == "-w" {
					file, err := os.Create(args[2])
					if err != nil {
						fmt.Fprintf(errFile, "Unable to open file: %s\r\n", args[2])
						continue
					}
					defer file.Close()

					for _, item := range history {
						_, err := fmt.Fprintf(file, "%s\n", item)
						if err != nil {
							fmt.Fprintf(errFile, "Unable to write to file: %s\r\n", args[2])
							continue
						}
					}
				} else if args[1] == "-a" {
					file, err := os.OpenFile(args[2], os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
					if err != nil {
						fmt.Fprintf(errFile, "Unable to open file: %s\r\n", args[2])
						continue
					}
					defer file.Close()

					for i := historyAppendIdx; i < len(history); i++ {
						item := history[i]
						_, err := fmt.Fprintf(file, "%s\n", item)
						if err != nil {
							fmt.Fprintf(errFile, "Unable to write to file: %s\r\n", args[2])
							continue
						}
					}
					historyAppendIdx = len(history)
				}
			} else {
				n := len(history)
				if len(args) == 2 {
					n, err = strconv.Atoi(args[1])
					if err != nil {
						fmt.Fprintf(errFile, "history %s: Invalid argument, not a number\r\n", args[1])
						continue
					}
				}
				var output strings.Builder

				for i := 0; i < n; i++ {
					idx := len(history) - n + i
					line := fmt.Sprintf("    %d  %s\r\n", idx+1, history[idx])
					output.WriteString(line)
				}

				handleBuiltinOutput(output.String(), isLastCommand)
			}

		default:
			// can use exec.LookPath(command) instead of custom getExecPath(command)
			_, found := getExecPath(command)
			if found {
				var cmd *exec.Cmd
				if len(args) > 1 {
					cmd = exec.Command(command, args[1:]...)
				} else {
					cmd = exec.Command(command)
				}

				if inFile == nil {
					cmd.Stdin = os.Stdin
				} else {
					cmd.Stdin = inFile
				}
				cmd.Stderr = errFile

				if isLastCommand {
					// Last command: output goes to outFile
					cmd.Stdout = outFile
				} else {
					// create pipe for next command
					inFile, err = cmd.StdoutPipe()
					if err != nil {
						log.Fatalf("Error creating stdout pipe for cmd: %v", err)
					}
				}
				cmd.Start()
				fmt.Print("\r")
				cmds = append(cmds, cmd)
			} else {
				fmt.Fprintf(outFile, "%s: command not found\n", command)
			}
		}
	}

	// Wait for all commands to finish
	for _, cmd := range cmds {
		cmd.Wait()
	}
}

func main() {
	histFile := os.Getenv("HISTFILE")
	if histFile != "" {
		file, err := os.Open(histFile)
		if err != nil {
			fmt.Println("unable to open history file")
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			history = append(history, line)
		}

	}

	reader := bufio.NewReader(os.Stdin)

	contCmd := ""
	for {
		// switch stdin to 'raw' mode for readline()
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}

		fmt.Fprintf(os.Stdout, "$ %s", contCmd)

		// Wait for user input
		input, shouldContinue, err := readline(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		history = append(history, input)
		if shouldContinue {
			contCmd = input
			continue
		}

		// Resume 'cooked' mode for executing commands
		term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Println()
		// fmt.Printf("len: %d, input: %s\n", len(input), input)

		tokens := tokenize(input)
		// fmt.Fprintf(os.Stderr, "tokens: len(%d) %v\n", len(tokens), tokens)

		pipeline, outFile, errFile := parse(tokens)

		execute(pipeline, outFile, errFile)

		// Close files immediately after use
		if outFile != os.Stdout {
			outFile.Close()
		}
		if errFile != os.Stderr {
			errFile.Close()
		}
	}
}
