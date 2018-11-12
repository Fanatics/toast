package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Fanatics/toast/collector"
)

// Func is a function which defines plugin behavior, and is provided a
// pointer to collector.Data.
type Func func(d *collector.Data) error

type plugin struct {
	name string
}

// New returns a plugin instance for a plugin to be initialized.
func New(name string) *plugin {
	return &plugin{
		name: name,
	}
}

// Init is called by plugin code and is provided a PluginFunc from the caller
// to handle the input Data (read from stdin).
func (p *plugin) Init(fn Func) {
	// read from stdin to get serialized bytes
	input := &bytes.Buffer{}
	_, err := io.Copy(input, os.Stdin)
	if err != nil {
		p.wrapErrAndLog(err)
		return
	}

	// deserialize bytes into *collector.Data
	inputData := &collector.Data{}
	err = json.Unmarshal(input.Bytes(), inputData)
	if err != nil {
		p.wrapErrAndLog(err)
		return
	}

	// execute "fn" and pass it the *collector.Data, where the plugin would use
	// the simplified AST to generate other code.
	p.wrapErrAndLog(fn(inputData))
}

func (p *plugin) wrapErrAndLog(err error) {
	if err != nil {
		fmt.Fprintf(os.Stdout, "[toast:plugin] %s: %v\n", p.name, err)
	}
}
