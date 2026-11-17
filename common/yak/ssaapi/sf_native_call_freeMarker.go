package ssaapi

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yaklang/yaklang/common/syntaxflow/sfvm"
	"github.com/yaklang/yaklang/common/utils/memedit"
)

var (
	ftlVarExtractor    = regexp.MustCompile(`\$\{([^?}]+)\}!`)
	ftlSuffixExtractor = regexp.MustCompile(`spring.freemarker.suffix=(.*)`)
)

var nativeCallFreeMarker = sfvm.NativeCallFunc(func(v sfvm.Values, frame *sfvm.SFFrame, params *sfvm.NativeCallActualParams) (bool, sfvm.Values, error) {
	prog, err := fetchProgram(v)
	if err != nil {
		return false, nil, err
	}

	ftls := make(map[string][]string)   // key is fileName, value is unsafe param
	owners := make(map[string]*Program) // fileName -> layer that owns it
	var vals []sfvm.ValueOperator
	// get suffix name
	var suffix string
	collect := func(owner *Program, s string, me *memedit.MemEditor) bool {
		if owner == nil || me == nil {
			return true
		}
		if strings.HasSuffix(s, ".properties") {
			matchs := ftlSuffixExtractor.FindStringSubmatch(me.GetSourceCode())
			if len(matchs) > 1 {
				suffix = matchs[1]
			}
		}
		matchs := ftlVarExtractor.FindAllStringSubmatch(me.GetSourceCode(), -1)
		for _, match := range matchs {
			if len(match) > 1 {
				ftls[s] = append(ftls[s], match[1])
			}
		}
		owners[s] = owner
		return true
	}
	// Incremental scans must see FreeMarker sidecars from every visible layer,
	// not just the top diff program's extra files.
	if overlay := fetchProgramOverlay(v); overlay != nil {
		overlay.forEachVisibleFile(false, true, func(entry overlayFileEntry) bool {
			return collect(entry.Program, entry.Path, entry.Editor)
		})
	} else {
		prog.ForEachExtraFile(func(s string, me *memedit.MemEditor) bool {
			return collect(prog, s, me)
		})
	}

	v.Recursive(func(value sfvm.ValueOperator) error {
		valStr, err := strconv.Unquote(value.String())
		if err != nil {
			return nil
		}
		name := fmt.Sprintf("%s%s", valStr, suffix)
		for fileurl, match := range ftls {
			if strings.HasSuffix(fileurl, name) {
				owner := owners[fileurl]
				if owner == nil {
					owner = prog
				}
				for _, v := range match {
					vals = append(vals, owner.NewConstValue(v))
				}
			}
		}
		return nil
	})

	return true, sfvm.NewValues(vals), nil
})
