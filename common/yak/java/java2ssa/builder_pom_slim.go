//go:build ssa_only

package java2ssa

import (
	fi "github.com/yaklang/yaklang/common/utils/filesys/filesys_interface"
	"github.com/yaklang/yaklang/common/utils/memedit"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

// handlePomDependencies is a no-op in the slim engine: SCA dependency
// extraction is an optional add-on, not part of SSA compilation.
func handlePomDependencies(prog *ssa.Program, fb *ssa.FunctionBuilder, filename string, editor *memedit.MemEditor, fileSystem fi.FileSystem) {
}
