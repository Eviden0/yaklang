//go:build !ssa_only

package ts2ssa

import (
	"strings"

	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/sca"
	"github.com/yaklang/yaklang/common/utils/filesys"
	fi "github.com/yaklang/yaklang/common/utils/filesys/filesys_interface"
	"github.com/yaklang/yaklang/common/utils/memedit"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

func handlePackageDependencies(prog *ssa.Program, fb *ssa.FunctionBuilder, filename string, editor *memedit.MemEditor, fileSystem fi.FileSystem) {
	if strings.TrimLeft(filename, string(fileSystem.GetSeparators())) != "package.json" {
		return
	}
	fb.SetEditor(editor)
	vfs := filesys.NewVirtualFs()
	vfs.AddFile(filename, editor.GetSourceCode())
	pkgs, err := sca.ScanFilesystem(vfs)
	if err != nil {
		log.Warnf("scan package.json error: %v", err)
	}
	prog.SCAPackages = append(prog.SCAPackages, pkgs...)
	fb.GenerateDependence(pkgs, filename)
}
