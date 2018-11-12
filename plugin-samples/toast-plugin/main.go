package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nilslice/toast/collector"
	"github.com/nilslice/toast/plugin"
)

func main() {
	plugin.New("toast-plugin").Init(func(data *collector.Data) error {
		var files []string
		for _, pkg := range data.Packages {
			for _, file := range pkg.Files {
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
