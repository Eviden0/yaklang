//go:build !irify_exclude && !ssa_only

package sfbuildin

import (
	"github.com/yaklang/yaklang/common/consts"
	"github.com/yaklang/yaklang/common/yakgrpc/yakit"
)

func getEmbedRuleHash() string {
	return yakit.Get(consts.EmbedSfBuildInRuleKey)
}

func setEmbedRuleHash(value string) {
	yakit.Set(consts.EmbedSfBuildInRuleKey, value)
}
