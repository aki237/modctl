package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func errf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func listModules(ctx *cli.Context) error {
	allowIndirect := ctx.Bool("allow-indirect")
	mf, err := loadModFile()
	if err != nil {
		return nil
	}

	for _, req := range mf.Require {
		v, err := parseVersion(req.Mod.Version)
		if err != nil {
			continue
		}
		if req.Indirect && !allowIndirect {
			continue
		}
		fmt.Println(
			strings.TrimSuffix(req.Mod.Path, fmt.Sprintf("/v%d", v.Major)),
		)
	}

	return nil
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
			{
				Name:   "list-modules",
				Action: listModules,
				Hidden: true,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "allow-indirect",
						Aliases: []string{"a"},
					},
				},
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
