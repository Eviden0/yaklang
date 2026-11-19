//go:build !irify_exclude && ssa_only

package sfbuildin

import (
	"strconv"

	"github.com/yaklang/yaklang/common/consts"
	"github.com/yaklang/yaklang/common/schema"
)

// The embed-rule hash key is a small KV entry. The slim engine reads and
// writes it directly so it does not link the yakgrpc/yakit facade.
func getEmbedRuleHash() string {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return ""
	}
	key := strconv.Quote(consts.EmbedSfBuildInRuleKey)
	var kv schema.GeneralStorage
	if err := db.Where("key = ?", key).First(&kv).Error; err != nil || kv.Value == "" {
		return ""
	}
	value, err := strconv.Unquote(kv.Value)
	if err != nil {
		return kv.Value
	}
	return value
}

func setEmbedRuleHash(value string) {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return
	}
	key := strconv.Quote(consts.EmbedSfBuildInRuleKey)
	valueStr := strconv.Quote(value)
	_ = db.Model(&schema.GeneralStorage{}).
		Where("key = ?", key).
		Assign(map[string]interface{}{"key": key, "value": valueStr}).
		FirstOrCreate(&schema.GeneralStorage{}).Error
}
