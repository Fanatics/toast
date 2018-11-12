package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type plugin struct {
	cmd       *exec.Cmd
	outputDir string
}

type runner struct {
	p    *plugin
	data io.Reader
}

const (
	outPrefix       = "out="
	pluginErrPrefix = "[toast:plugin]"
)

var pluginList []plugin

func (p *plugin) String() string {
	var all []string
	for _, plug := range pluginList {
		all = append(all, fmt.Sprintf(
			"plugin command: %s, output: [%s]",
			plug.cmd.Args, plug.outputDir,
		))
	}

	return strings.Join(all, "\n")
}

func (p *plugin) Set(value string) error {
	// --plugin="amdm_gen_db subcmd -flag1 val --flag2=val2:out=./internal/db"
	//                                                     ^
	pluginParts := strings.Split(value, ":")
	if len(pluginParts) < 2 {
		return fmt.Errorf("invalid plugin flag value: %s", value)
	}

	var pluginCmd, pluginOutput string
	var pluginOptions []string

	pluginCmdVals := strings.Split(pluginParts[0], " ")
	if len(pluginCmdVals) == 0 {
		return fmt.Errorf("invalid plugin flag value (bad command): %s", value)
	}

	pluginCmd = pluginCmdVals[0]
	if strings.HasPrefix(pluginParts[1], outPrefix) {
		// plugin was passed no options, and output path is second part
		pluginOutput = pluginParts[1]
	} else {
		return fmt.Errorf("invalid plugin flag value (bad out): %s", value)
	}
	if len(pluginCmdVals) > 1 {
		// plugin was passed options as second part, output is third
		pluginOptions = pluginCmdVals[1:]
	}

	outputVals := strings.Split(pluginOutput, "=")
	if !strings.HasPrefix(pluginOutput, outPrefix) || len(outputVals) < 2 {
		return fmt.Errorf("invalid plugin out value: %s", pluginOutput)
	}
	baseOutputDir := outputVals[1]

	pluginList = append(pluginList, plugin{
		cmd:       exec.Command(pluginCmd, pluginOptions...),
		outputDir: baseOutputDir,
	})

	return nil
}

func (p *plugin) each(fn func(idx int, plug *plugin) error) error {
	errChan := make(chan error)
	done := make(chan struct{})
	var errs []string
	go func() {
		for {
			select {
			case <-done:
				return

			case err := <-errChan:
				errs = append(errs, err.Error())
			}
		}
	}()

	for i := range pluginList {
		plug := &pluginList[i]
		err := fn(i, plug)
		if err != nil {
			errChan <- fmt.Errorf(
				"%s %s: %v (%s)",
				pluginErrPrefix, plug.cmd.Args[0], err, plug.cmd.Path,
			)
		}
	}
	done <- struct{}{}
	if errs != nil {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (r *runner) run() error {
	r.p.cmd.Stdin = r.data
	r.p.cmd.Stdout = os.Stdout
	r.p.cmd.Stderr = os.Stderr

	_, err := exec.LookPath(r.p.cmd.Args[0])
	if err != nil {
		return err
	}

	return r.p.cmd.Run()
}
