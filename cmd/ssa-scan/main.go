// Command ssa-scan is the SSA-only build of the Yak CLI. It exposes the
// compile / SyntaxFlow / code-scan commands without importing the AI, MITM,
// crawler, plugin or gRPC server trees that the full yak binary carries.
package main

import (
	"fmt"
	"os"

	"github.com/yaklang/yaklang/common/consts"
	"github.com/yaklang/yaklang/common/log"
	cli "github.com/yaklang/yaklang/common/urfavecli"
	"github.com/yaklang/yaklang/common/yak/cmd/ssacmds"
	"github.com/yaklang/yaklang/common/yakgrpc/yakit"
)

func main() {
	app := cli.NewApp()
	app.Name = "ssa-scan"
	app.Usage = "Yak SSA compiler and SyntaxFlow code scanner"
	app.Version = consts.GetYakVersion()
	// SSACompilerCommands is []*cli.Command; urfavecli's App takes values.
	commands := make([]cli.Command, 0, len(ssacmds.SSACompilerCommands))
	for _, command := range ssacmds.SSACompilerCommands {
		if command != nil {
			commands = append(commands, *command)
		}
	}
	app.Commands = commands
	app.Before = func(c *cli.Context) error {
		if !needsDatabase(os.Args) {
			return nil
		}
		if err := consts.InitializeYakitDatabase("", "", ""); err != nil {
			return err
		}
		if err := yakit.CallPostInitDatabase(); err != nil {
			return fmt.Errorf("CallPostInitDatabase failed: %w", err)
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		log.Errorf("%v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// needsDatabase skips profile/project DB initialization for version and help,
// so `ssa-scan --help` stays instant in CI probes.
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
