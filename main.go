package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/AndrewVos/colour"
	"gopkg.in/yaml.v2"
)

type Server struct {
	User         string `json:"user"`
	Address      string `json:"address"`
	Port         int    `json:"port"`
	IdentityFile string `json:"identity_file"`
}

type ServerConfiguration struct {
	Name    string   `json:"name"`
	Tasks   []Task   `json:"tasks"`
	Servers []Server `json:"servers"`
}

type Task struct {
	Name         string            `json:"name"`
	Script       string            `json:"script"`
	Parallel     bool              `json:"parallel"`
	Environment  map[string]string `json:"environment"`
	Parameters   []string          `json:"parameters"`
	MergedOutput bool              `json:"merged_output"`
}

func (t *Task) ExecuteOnServers(servers []Server) []error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	parallel := t.Parallel && len(servers) > 1

	var allErrors []error

	for _, server := range servers {
		fmt.Printf(colour.Blue("Executing %q on %q\n"), t.Script, server.Address)
		if parallel {
			wg.Add(1)
			var out io.Writer
			if t.MergedOutput {
				out = os.Stdout
			} else {
				out = &DelayedStdWriter{Out: os.Stdout}
			}
			go func(server Server, task *Task, out io.Writer) {
				err := task.Execute(server, out)
				if delayedWriter, ok := out.(*DelayedStdWriter); ok {
					delayedWriter.Flush()
				}

				if err != nil {
					mu.Lock()
					allErrors = append(allErrors, err)
					mu.Unlock()
				}
				wg.Done()
			}(server, t, out)
		} else {
			err := t.Execute(server, os.Stdout)
			if err != nil {
				mu.Lock()
				allErrors = append(allErrors, err)
				mu.Unlock()
			}
		}
	}

	wg.Wait()
	return allErrors
}

func (t *Task) Execute(server Server, out io.Writer) error {
	script, err := ioutil.ReadFile(t.Script)
	if err != nil {
		return errors.New(fmt.Sprintf("I couldn't read from your script %q:\n%v", t.Script, err))
	}

	port := "22"
	if server.Port != 0 {
		port = strconv.Itoa(server.Port)
	}

	arguments := []string{
		"-T",
		"-o", "StrictHostKeyChecking=no",
		"-p", port,
	}
	if server.IdentityFile != "" {
		arguments = append(arguments, "-i", server.IdentityFile)
	}
	arguments = append(arguments, fmt.Sprintf("%v@%v", server.User, server.Address))

	cmd := exec.Command("ssh", arguments...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.New(fmt.Sprintf("I couldn't connect to stdin of ssh:\n%v", err))
	}
	for name, value := range t.Environment {
		fmt.Fprintf(stdin, "export %v=\"%v\"\n", name, value)
	}

	for _, parameter := range t.Parameters {
		value := os.Getenv(parameter)
		if value == "" {
			return errors.New(fmt.Sprintf("Couldn't find environment variable %q. This is required.", parameter))
		}
		fmt.Fprintf(stdin, "export %v=\"%v\"\n", parameter, value)
	}
	stdin.Write(script)
	stdin.Close()

	cmd.Stdout = out
	cmd.Stderr = out

	err = cmd.Start()
	if err != nil {
		return errors.New(fmt.Sprintf("I couldn't launch ssh:\n%v", err))
	}

	err = cmd.Wait()
	if err != nil {
		return errors.New(fmt.Sprintf("I had some problems running %q on %q:\n%v", t.Script, server.Address, err))
	}
	return nil
}

func main() {
	errors := garrison()
	if len(errors) > 0 {
		for _, err := range errors {
			Redf("%v\n", err)
		}
		os.Exit(len(errors))
	}
}

func readConfiguration() ([]ServerConfiguration, error) {
	var serverConfigurations []ServerConfiguration
	var decodeErrorFormat = "I couldn't decode your %v\n%v\n"

	if _, err := os.Stat("garrison.json"); err == nil {
		b, err := ioutil.ReadFile("garrison.json")
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, &serverConfigurations)
		if err != nil {
			return nil, errors.New(fmt.Sprintf(decodeErrorFormat, "garrison.json", err))
		}
		return serverConfigurations, nil
	}

	if _, err := os.Stat("garrison.yml"); err == nil {
		b, err := ioutil.ReadFile("garrison.yml")
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(b, &serverConfigurations)
		if err != nil {
			return nil, errors.New(fmt.Sprintf(decodeErrorFormat, "garrison.yml", err))
		}
		return serverConfigurations, nil
	}

	return nil, errors.New("Couldn't find a garrison.[json|yml|] configuration file. You need one of these.")
}

func garrison() []error {
	serverConfigurations, err := readConfiguration()
	if err != nil {
		return []error{err}
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "--completion-help" {
			printCompletionCommands(serverConfigurations)
		} else {
			command := os.Args[1]
			return executeCommand(command, serverConfigurations)
		}
	} else {
		printCommands(serverConfigurations)
	}

	return nil
}

func printCommands(serverConfigurations []ServerConfiguration) {
	fmt.Printf("Usage: %v <command>\n\n", os.Args[0])
	fmt.Println("Commands:")
	for _, serverConfiguration := range serverConfigurations {
		var addresses []string
		for i, s := range serverConfiguration.Servers {
			addresses = append(addresses, fmt.Sprintf("%d: %v", i, s.Address))
		}
		fmt.Printf("%v: {%v}\n", serverConfiguration.Name, strings.Join(addresses, ", "))

		for _, task := range serverConfiguration.Tasks {
			fmt.Printf("  %v:[index|address:]%v\n", serverConfiguration.Name, task.Name)
		}

	}
}

func printCompletionCommands(serverConfigurations []ServerConfiguration) {
	for _, serverConfiguration := range serverConfigurations {
		for _, task := range serverConfiguration.Tasks {
			fmt.Printf("%v:%v\n", serverConfiguration.Name, task.Name)
			for i, _ := range serverConfiguration.Servers {
				fmt.Printf("%v:%v:%v\n", serverConfiguration.Name, i, task.Name)
			}
			for _, server := range serverConfiguration.Servers {
				fmt.Printf("%v:%v:%v\n", serverConfiguration.Name, server.Address, task.Name)
			}
		}
	}
}

func executeCommand(command string, serverConfigurations []ServerConfiguration) []error {
	for _, serverConfiguration := range serverConfigurations {
		for _, task := range serverConfiguration.Tasks {
			var matchedServers []Server
			if fmt.Sprintf("%v:%v", serverConfiguration.Name, task.Name) == command {
				for _, server := range serverConfiguration.Servers {
					matchedServers = append(matchedServers, server)
				}
			}

			for i, server := range serverConfiguration.Servers {
				commandWithServer := fmt.Sprintf("%v:%v:%v", serverConfiguration.Name, server.Address, task.Name)
				commandWithIndex := fmt.Sprintf("%v:%v:%v", serverConfiguration.Name, i, task.Name)
				if command == commandWithServer || command == commandWithIndex {
					matchedServers = append(matchedServers, server)
					break
				}
			}

			if len(matchedServers) > 0 {
				errors := task.ExecuteOnServers(matchedServers)
				return errors
			}
		}
	}
	return []error{errors.New(fmt.Sprintf("I couldn't find the command %q\n", command))}
}

func Redf(format string, v ...interface{}) {
	fmt.Printf(colour.Red(format), v...)
}

func FatalRedf(format string, v ...interface{}) {
	fmt.Printf(colour.Red(format), v...)
	os.Exit(1)
}

func Bluef(format string, v ...interface{}) {
	fmt.Printf(colour.Blue(format), v...)
}

type DelayedStdWriter struct {
	Out    io.Writer
	buffer []byte
}

func (s *DelayedStdWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		s.buffer = append(s.buffer, b)
	}
	return len(p), nil
}

func (s *DelayedStdWriter) Flush() {
	s.Out.Write(s.buffer)
}
