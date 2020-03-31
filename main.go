package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func errf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func main() {
	app := cli.App{
		Name:    "modctl",
		Version: "v0.0.0",
		Commands: []*cli.Command{
			{
				Name:   "upgrade",
				Action: (&Upgrader{}).Exec,
			},
		},
	}
	app.Name = "modctl"
	app.EnableBashCompletion = true
	app.Version = "v0.0.0"
	err := app.Run(os.Args)
	if err != nil {
		errf("Error: %s\n", err)
	}
}
