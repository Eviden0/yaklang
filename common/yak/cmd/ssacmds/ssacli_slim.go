//go:build !irify_exclude && ssa_only

package ssacmds

import cli "github.com/yaklang/yaklang/common/urfavecli"

// extraSSACommands is empty in the ssa_only profile: the AI completion
// commands are deliberately not linked into the slim engine.
var extraSSACommands []*cli.Command

// SSASlimCommands is the non-AI SSA CLI surface used by cmd/yak-ssa.
var SSASlimCommands = append([]*cli.Command{}, ssacBaseCommands...)
