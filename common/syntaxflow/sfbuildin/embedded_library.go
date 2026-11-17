//go:build !irify_exclude

package sfbuildin

import (
	"io/fs"
	"strings"
	"sync"

	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/syntaxflow/sfdb"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/filesys"
)

var embeddedLibraryCache sync.Map // library name -> *schema.SyntaxFlowRule

func init() {
	sfdb.RegisterEmbeddedLibraryLoader(GetEmbeddedLibrary)
}

// GetEmbeddedLibrary resolves one library rule (desc `lib: 'name'`) straight
// from the embedded rule snapshot. It exists for scans that skip the builtin
// database import (YAK_SKIP_BUILTIN_SYNC): <include('name')> must keep working
// without a populated profile database.
func GetEmbeddedLibrary(name string) (*schema.SyntaxFlowRule, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, utils.Error("library name is empty")
	}
	if cached, ok := embeddedLibraryCache.Load(name); ok {
		if rule, ok := cached.(*schema.SyntaxFlowRule); ok && rule != nil {
			return rule, nil
		}
	}
	if ruleFSWithHash == nil {
		InitEmbedFS()
	}
	if ruleFSWithHash == nil {
		return nil, utils.Error("embedded rule filesystem is not initialized")
	}

	var found *schema.SyntaxFlowRule
	err := filesys.Recursive(".", filesys.WithFileSystem(ruleFSWithHash), filesys.WithFileStat(func(path string, info fs.FileInfo) error {
		if found != nil || info == nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".sf") {
			return nil
		}
		raw, err := ruleFSWithHash.ReadFile(path)
		if err != nil {
			return nil
		}
		// Cheap textual prefilter: only parse files that mention the library
		// name at all (a handful), not all of the builtin snapshot.
		if !strings.Contains(string(raw), name) {
			return nil
		}
		rule, err := sfdb.CheckSyntaxFlowRuleContent(string(raw))
		if err != nil || rule == nil {
			return nil
		}
		if rule.AllowIncluded && (rule.IncludedName == name || rule.Title == name) {
			found = rule
		}
		return nil
	}))
	if err != nil {
		return nil, utils.Wrapf(err, "walk embedded rules for library %q", name)
	}
	if found == nil {
		return nil, utils.Errorf("embedded library %q not found", name)
	}
	embeddedLibraryCache.Store(name, found)
	log.Infof("resolved embedded library %q without database sync", name)
	return found, nil
}
