package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Fanatics/toast/collector"
	"github.com/Fanatics/toast/plugin"
)

func main() {
	plugin.New("toast-plugin").Init(func(data *collector.Data) error {
		var files []string
		for _, pkg := range data.Packages {
			for _, file := range pkg.Files {
				/*
					within each package is a set of files. a collector.File is:

					type File struct {
						Name             string            `json:"name,omitempty"`
						Package          string            `json:"package,omitempty"`
						Imports          []Import          `json:"imports,omitempty"`
						TypeDefs         []TypeDefinition  `json:"type_defs,omitempty"`
						Structs          []Struct          `json:"structs,omitempty"`
						Interfaces       []Interface       `json:"interfaces,omitempty"`
						Funcs            []Func            `json:"funcs,omitempty"`
						Consts           []Const           `json:"consts,omitempty"`
						Vars             []Var             `json:"vars,omitempty"`
						Comments         []Comment         `json:"comments,omitempty"`
						MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
						GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
						BuildTags        []Constraint      `json:"build_tags,omitempty"`
					}

					using those fields, you can generate code based on the Go code
					which was parsed to a simplified AST.
				*/

				// accumulate the file names, as a basic example
				files = append(files, file.Name)
			}
		}

		f, err := os.Create(filepath.Join(data.OutputBase, "my-file.txt"))
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write([]byte(
			strings.Join(files, "\n"),
		))
		if err != nil {
			return err
		}

		return nil
	})
}
