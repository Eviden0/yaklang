package ssaapi

import (
	"github.com/yaklang/yaklang/common/syntaxflow/sfvm"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/memedit"
)

// overlayFileEntry is one file the overlay view exposes to a native call,
// together with the layer program that finally owns it. Path is the stored
// name (it may carry the layer program-name prefix); Logical is the
// prefix-free name used for ownership matching.
type overlayFileEntry struct {
	Program *Program
	Path    string
	Logical string
	Editor  *memedit.MemEditor
}

// fetchProgramOverlay returns the incremental view behind the native-call
// input, or nil for a normal single-program query. A rule-level native call
// receives the overlay itself; a value-level call (e.g. $v<FilenameByContent()>)
// only carries a *Value, so the overlay is recovered from its parent program.
func fetchProgramOverlay(v sfvm.Values) *ProgramOverLay {
	var overlay *ProgramOverLay
	_ = v.Recursive(func(op sfvm.ValueOperator) error {
		switch ret := op.(type) {
		case *ProgramOverLay:
			if ret != nil {
				overlay = ret
				return utils.Error("normal abort")
			}
		}
		return nil
	})
	if overlay != nil {
		return overlay
	}
	_ = v.Recursive(func(op sfvm.ValueOperator) error {
		switch ret := op.(type) {
		case *Value:
			if ret != nil && ret.ParentProgram != nil {
				if o := ret.ParentProgram.GetOverlay(); o != nil {
					overlay = o
					return utils.Error("normal abort")
				}
			}
		case *Program:
			if ret != nil {
				if o := ret.GetOverlay(); o != nil {
					overlay = o
					return utils.Error("normal abort")
				}
			}
		}
		return nil
	})
	return overlay
}

// forEachVisibleFile walks the file set visible in the overlay view with
// newest-layer-wins ownership: newer diffs shadow older layers, and the base
// skips every path owned or deleted by a diff (ExcludeFile). fileList / extra
// select which per-program maps participate. The callback receives the layer
// that owns the file so values stay bound to the right program.
func (p *ProgramOverLay) forEachVisibleFile(includeFileList, includeExtra bool, cb func(overlayFileEntry) bool) {
	if p == nil || cb == nil {
		return
	}
	seen := make(map[string]struct{})
	visit := func(prog *Program, skipExcluded bool) bool {
		if prog == nil || prog.Program == nil {
			return true
		}
		maps := make([]map[string]string, 0, 2)
		if includeFileList && prog.Program.FileList != nil {
			maps = append(maps, prog.Program.FileList)
		}
		if includeExtra && prog.Program.ExtraFile != nil {
			maps = append(maps, prog.Program.ExtraFile)
		}
		for _, files := range maps {
			for raw, hash := range files {
				logical := normalizeOverlayFilePath(raw, prog.GetProgramName())
				if logical == "" {
					continue
				}
				if _, ok := seen[logical]; ok {
					continue
				}
				if skipExcluded && p.IsExcludedPath(logical) {
					continue
				}
				editor, err := prog.getEditor(raw, hash)
				if err != nil || editor == nil {
					continue
				}
				seen[logical] = struct{}{}
				if !cb(overlayFileEntry{Program: prog, Path: raw, Logical: logical, Editor: editor}) {
					return false
				}
			}
		}
		return true
	}
	for i := len(p.Diff) - 1; i >= 0; i-- {
		layer := p.Diff[i]
		if layer == nil || layer.Program == nil {
			continue
		}
		if !visit(layer.Program, false) {
			return
		}
	}
	if p.Base != nil {
		visit(p.Base, true)
	}
}

// visibleFileSet returns every logical path visible in the overlay view.
func (p *ProgramOverLay) visibleFileSet() map[string]struct{} {
	set := make(map[string]struct{})
	p.forEachVisibleFile(true, true, func(e overlayFileEntry) bool {
		set[e.Logical] = struct{}{}
		return true
	})
	return set
}

// isVisibleFile reports whether the value's file is part of the overlay view.
func (p *ProgramOverLay) isVisibleFile(filePath, programName string) bool {
	logical := normalizeOverlayFilePath(filePath, programName)
	if logical == "" {
		return false
	}
	visible := false
	p.forEachVisibleFile(true, true, func(e overlayFileEntry) bool {
		if e.Logical == logical {
			visible = true
			return false
		}
		return true
	})
	return visible
}
