//go:build ssa_only

package sfreport

import (
	"time"

	"github.com/yaklang/gorm"
	sflog "github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/bizhelper"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

// filterSSARiskLocal mirrors yakit.FilterSSARisk so the slim engine keeps the
// report filters without linking the gRPC/DB facade.
func filterSSARiskLocal(db *gorm.DB, filter *ypb.SSARisksFilter) *gorm.DB {
	if filter == nil {
		return db
	}

	db = bizhelper.ExactQueryInt64ArrayOr(db, "id", filter.GetID())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "program_name", filter.GetProgramName())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "function_name", filter.GetFunctionName())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "code_source_url", filter.GetCodeSourceUrl())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "risk_type", filter.GetRiskType())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "severity", filter.GetSeverity())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "from_rule", filter.GetFromRule())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "runtime_id", filter.GetRuntimeID())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "hash", filter.GetHash())
	db = bizhelper.ExactQueryUint64ArrayOr(db, "result_id", filter.GetResultID())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "result_uuid", filter.GetResultUUID())
	db = bizhelper.FuzzSearchWithStringArrayOrEx(db, []string{"tags"}, filter.GetTags(), false)
	db = bizhelper.FuzzSearchEx(db, []string{"title", "title_verbose"}, filter.GetTitle(), false)
	db = bizhelper.FuzzSearchEx(db, []string{
		"id", "hash",
		"program_name", "code_source_url", "function_name",
		"risk_type", "severity", "from_rule", "tags",
		"title", "title_verbose",
	}, filter.GetSearch(), false)
	if filter.GetIsRead() != 0 {
		db = bizhelper.QueryByBool(db, "is_read", filter.GetIsRead() > 0)
	}
	if filter.GetAfterCreatedAt() > 0 {
		db = bizhelper.QueryByTimeRangeWithTimestamp(db, "created_at", filter.GetAfterCreatedAt(), time.Now().Add(10*time.Minute).Unix())
	}
	if filter.GetBeforeCreatedAt() > 0 {
		db = bizhelper.QueryByTimeRangeWithTimestamp(db, "created_at", 0, filter.GetBeforeCreatedAt())
	}
	if disposalStatuses := filter.GetLatestDisposalStatus(); len(disposalStatuses) > 0 {
		db = bizhelper.ExactOrQueryStringArrayOr(db, "latest_disposal_status", disposalStatuses)
	}
	if dr := filter.GetSSARiskDiffRequest(); dr != nil {
		baselineTaskID := dr.GetBaseLine().GetRiskRuntimeId()
		compareTaskID := dr.GetCompare().GetRiskRuntimeId()
		db = filterSSARiskByCompare(db, baselineTaskID, compareTaskID)
	}
	if filter.GetIncremental() {
		runtimeIDs := filter.GetRuntimeID()
		db = filterSSARiskByIncremental(db, runtimeIDs...)
	}
	return db
}

func filterSSARiskByIncremental(db *gorm.DB, runtimeId ...string) *gorm.DB {
	riskTable := schema.TableSSARisks
	disposalTable := schema.TableSSARiskDisposals
	taskTable := schema.TableSyntaxFlowScanTask
	if len(runtimeId) == 0 {
		var disposedFeatureHashes []string
		err := db.New().Model(&schema.SSARiskDisposals{}).
			Pluck("DISTINCT risk_feature_hash", &disposedFeatureHashes).Error
		if err != nil {
			sflog.Errorf("Failed to query disposed feature hashes: %v", err)
			return db
		}
		db = db.Where("risk_feature_hash NOT IN (?)", disposedFeatureHashes)
		return db
	} else if len(runtimeId) == 1 {
		var baseTask schema.SyntaxFlowScanTask
		if err := db.New().Where("task_id = ?", runtimeId).First(&baseTask).Error; err != nil {
			return db
		}
		var validFeatureHashes []string
		subWhere := `
		NOT EXISTS (
			SELECT 1
			FROM ` + disposalTable + `
			JOIN ` + taskTable + ` base_task ON ` + disposalTable + `.task_id = base_task.task_id
			WHERE ` + disposalTable + `.risk_feature_hash = ` + riskTable + `.risk_feature_hash
			  AND base_task.scan_batch <= ?
		)
	`
		err := db.New().
			Model(&schema.SSARisk{}).
			Where(riskTable+".runtime_id = ? AND "+riskTable+".risk_feature_hash != ''", runtimeId).
			Where(subWhere, baseTask.ScanBatch).
			Pluck("DISTINCT "+riskTable+".risk_feature_hash", &validFeatureHashes).Error
		if err != nil {
			sflog.Errorf("Failed to query valid feature hashes: %v", err)
			return db
		}
		return db.Where("risk_feature_hash IN (?)", validFeatureHashes)
	}
	return db
}

func filterSSARiskByCompare(db *gorm.DB, baselineTaskID, compareTaskID string) *gorm.DB {
	if baselineTaskID == "" || compareTaskID == "" {
		return db
	}
	var compareHash []string
	db.New().Model(&schema.SSARisk{}).
		Where("runtime_id = ?", compareTaskID).
		Where("risk_feature_hash != ''").
		Pluck("risk_feature_hash", &compareHash)
	if len(compareHash) > 0 {
		db = db.Where("risk_feature_hash NOT IN (?)", compareHash)
	}
	return db
}

func getSSARiskDisposalsOnlyLocal(db *gorm.DB, riskID int64) ([]schema.SSARiskDisposals, error) {
	db = db.Model(&schema.SSARiskDisposals{})
	var disposals []schema.SSARiskDisposals
	if err := db.Where("ssa_risk_id = ?", riskID).
		Order("updated_at DESC").
		Find(&disposals).Error; err != nil {
		return nil, utils.Errorf("GetSSARiskDisposalsOnly failed: %v", err)
	}
	return disposals, nil
}

func getSSARiskDisposalsWithInheritanceLocal(db *gorm.DB, riskID int64) ([]schema.SSARiskDisposals, error) {
	disposalTable := schema.TableSSARiskDisposals
	taskTable := schema.TableSyntaxFlowScanTask
	var risk schema.SSARisk
	if err := db.Where("id = ?", riskID).First(&risk).Error; err != nil {
		return nil, utils.Errorf("GetSSARiskDisposalsWithInheritance failed to query risk: %v", err)
	}
	if risk.RiskFeatureHash == "" {
		return getSSARiskDisposalsOnlyLocal(db, riskID)
	}
	var currentTask schema.SyntaxFlowScanTask
	if err := db.Where("task_id = ?", risk.RuntimeId).First(&currentTask).Error; err != nil {
		return getSSARiskDisposalsOnlyLocal(db, riskID)
	}
	var disposals []schema.SSARiskDisposals
	if err := db.Model(&schema.SSARiskDisposals{}).
		Joins("JOIN "+taskTable+" ON "+disposalTable+".task_id = "+taskTable+".task_id").
		Where(disposalTable+".risk_feature_hash = ? AND "+taskTable+".scan_batch <= ?",
			risk.RiskFeatureHash, currentTask.ScanBatch).
		Order(taskTable + ".scan_batch DESC, " + disposalTable + ".updated_at DESC").
		Find(&disposals).Error; err != nil {
		return nil, utils.Errorf("GetSSARiskDisposalsWithInheritance failed: %v", err)
	}
	return disposals, nil
}
