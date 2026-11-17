//go:build ssa_only

package resources_monitor

import (
	"strconv"

	"github.com/yaklang/yaklang/common/consts"
	"github.com/yaklang/yaklang/common/schema"
)

func getResourceHash(resourceID string) string {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return ""
	}
	key := strconv.Quote(resourceID)
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

func setResourceHash(resourceID, hash string) {
	db := consts.GetGormProfileDatabase()
	if db == nil {
		return
	}
	key := strconv.Quote(resourceID)
	valueStr := strconv.Quote(hash)
	_ = db.Model(&schema.GeneralStorage{}).
		Where("key = ?", key).
		Assign(map[string]interface{}{"key": key, "value": valueStr}).
		FirstOrCreate(&schema.GeneralStorage{}).Error
}
