//go:build ssa_only

package php2ssa

import (
	"github.com/yaklang/yaklang/common/utils/memedit"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

// handleComposerDependencies is a no-op in the slim engine: SCA dependency
// extraction is optional and not part of SSA compilation.
func handleComposerDependencies(prog *ssa.Program, builder *ssa.FunctionBuilder, filename string, editor *memedit.MemEditor) {
}
