package main

import (
	"os"

	"github.com/capitalrx/grafana/pkg/util/cmd"
)

func main() {
	os.Exit(cmd.RunGrafanaCmd("cli"))
}
