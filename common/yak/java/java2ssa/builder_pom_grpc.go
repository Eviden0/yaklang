//go:build !ssa_only

package java2ssa

import (
	"strings"

	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/sca"
	"github.com/yaklang/yaklang/common/utils/filesys"
	fi "github.com/yaklang/yaklang/common/utils/filesys/filesys_interface"
	"github.com/yaklang/yaklang/common/utils/memedit"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

func handlePomDependencies(prog *ssa.Program, fb *ssa.FunctionBuilder, filename string, editor *memedit.MemEditor, fileSystem fi.FileSystem) {
	if strings.TrimLeft(filename, string(fileSystem.GetSeparators())) != "pom.xml" {
		return
	}
	fb.SetEditor(editor)
	vfs := filesys.NewVirtualFs()
	vfs.AddFile(filename, editor.GetSourceCode())
	pkgs, err := sca.ScanFilesystem(vfs)
	if err != nil {
		log.Warnf("scan pom.xml error: %v", err)
	}
	prog.AppendSCAPackages(pkgs...)
	fb.GenerateDependence(pkgs, filename)
}
