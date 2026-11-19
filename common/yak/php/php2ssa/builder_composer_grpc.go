//go:build !ssa_only

package php2ssa

import (
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/sca"
	"github.com/yaklang/yaklang/common/utils/filesys"
	"github.com/yaklang/yaklang/common/utils/memedit"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

func handleComposerDependencies(prog *ssa.Program, builder *ssa.FunctionBuilder, filename string, editor *memedit.MemEditor) {
	builder.SetEditor(editor)
	vfs := filesys.NewVirtualFs()
	vfs.AddFile(filename, editor.GetSourceCode())
	pkgs, err := sca.ScanFilesystem(vfs)
	if err != nil {
		log.Warnf("scan composer.lock error: %v", err)
	}
	prog.SCAPackages = append(prog.SCAPackages, pkgs...)
	builder.GenerateDependence(pkgs, filename)
}
