//go:build ssa_only

// Command yak-ssa is the slim SSA engine: compile / SyntaxFlow / code-scan
// without the AI, MITM, crawler, plugin or gRPC server trees.
package main

import (
	"fmt"
	"os"

	"github.com/yaklang/yaklang/common/consts"
	"github.com/yaklang/yaklang/common/log"
	cli "github.com/yaklang/yaklang/common/urfavecli"
	"github.com/yaklang/yaklang/common/yak/cmd/ssacmds"
)

func main() {
	app := cli.NewApp()
	app.Name = "yak-ssa"
	app.Usage = "Yak SSA compiler and SyntaxFlow code scanner (slim)"
	app.Version = consts.GetYakVersion()

	commands := make([]cli.Command, 0, len(ssacmds.SSASlimCommands))
	for _, command := range ssacmds.SSASlimCommands {
		if command != nil {
			commands = append(commands, *command)
		}
	}
	app.Commands = commands
	app.Before = func(c *cli.Context) error {
		if !needsDatabase(os.Args) {
			return nil
		}
		return consts.InitializeYakitDatabase("", "", "")
	}
	if err := app.Run(os.Args); err != nil {
		log.Errorf("%v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func needsDatabase(args []string) bool {
	if len(args) <= 1 {
		return false
	}
	switch args[1] {
	case "-v", "--version", "-version", "version", "help", "-h", "--help":
		return false
	}
	return true
}
