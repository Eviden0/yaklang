//go:build ssa_only

package syntaxflow_scan

import (
	"path/filepath"
	"strings"

	"github.com/yaklang/gorm"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/syntaxflow/sfdb"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/bizhelper"
	"github.com/yaklang/yaklang/common/yak/ssa/ssadb"
	"github.com/yaklang/yaklang/common/yak/ssaapi/ssaconfig"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

type ssaRiskLevelCount struct {
	Count    int64  `json:"count"`
	Severity string `json:"severity"`
}

func getMaxScanBatch(db *gorm.DB, programs []string) (uint64, error) {
	var result struct {
		MaxBatch uint64 `json:"max_batch"`
	}
	programsStr := strings.Join(programs, schema.SYNTAXFLOWSCAN_PROGRAM_SPLIT)
	err := db.Model(&schema.SyntaxFlowScanTask{}).
		Where("programs = ?", programsStr).
		Select("COALESCE(MAX(scan_batch), 0) as max_batch").
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.MaxBatch, nil
}

func getSSARiskLevelCount(db *gorm.DB, filter *ypb.SSARisksFilter) ([]*ssaRiskLevelCount, error) {
	query := db.Model(&schema.SSARisk{})
	if filter != nil {
		// The scan task count path only filters by runtime id. Keep the slim
		// helper focused; arbitrary report filters use the sfreport path.
		query = bizhelper.ExactOrQueryStringArrayOr(query, "runtime_id", filter.GetRuntimeID())
	}
	query = query.Select("severity as severity, COUNT(*) as count").Group("severity")
	var counts []*ssaRiskLevelCount
	if err := query.Scan(&counts).Error; err != nil {
		return nil, utils.Errorf("scan failed: %v", err)
	}
	return counts, nil
}

func queryLatestSSAProgramNameByProjectId(db *gorm.DB, projectID uint64) (string, error) {
	var names []string
	err := ssadb.GetDB().Model(&ssadb.IrProgram{}).
		Where("project_id = ?", projectID).
		Order("updated_at desc").
		Limit(1).
		Pluck("program_name", &names).Error
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", nil
	}
	return names[0], nil
}

func filterSyntaxFlowRule(db *gorm.DB, filter *ypb.SyntaxFlowRuleFilter) *gorm.DB {
	db = db.Model(&schema.SyntaxFlowRule{})
	if filter == nil {
		return db
	}
	if len(filter.GetGroupNames()) > 0 {
		db = db.Joins("JOIN syntax_flow_rule_and_group ON syntax_flow_rule_and_group.syntax_flow_rule_id = syntax_flow_rules.id").
			Joins("JOIN syntax_flow_groups ON syntax_flow_groups.id = syntax_flow_rule_and_group.syntax_flow_group_id").
			Where("syntax_flow_groups.group_name IN (?)", filter.GetGroupNames()).
			Group("syntax_flow_rules.id")
	}
	db = bizhelper.ExactQueryUInt64ArrayOr(db, "id", filter.GetIds())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "rule_id", filter.GetRuleIds())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "severity", filter.GetSeverity())
	if len(filter.GetRuleNames()) > 0 {
		db = bizhelper.ExactOrQueryStringArrayOr(db, "rule_name", filter.GetRuleNames())
	}
	db = bizhelper.ExactOrQueryStringArrayOr(db, "language", filter.GetLanguage())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "purpose", filter.GetPurpose())
	db = bizhelper.ExactOrQueryStringArrayOr(db, "syntax_flow_rules.tag", filter.GetTag())
	if keyword := strings.TrimSpace(filter.GetKeyword()); keyword != "" {
		db = bizhelper.FuzzSearchWithStringArrayOrEx(db, []string{
			"syntax_flow_rules.rule_name",
			"syntax_flow_rules.title",
			"syntax_flow_rules.title_zh",
			"syntax_flow_rules.description",
			"syntax_flow_rules.content",
		}, []string{keyword}, false)
	}
	if filter.GetAfterId() > 0 {
		db = db.Where("id > ?", filter.GetAfterId())
	}
	if filter.GetBeforeId() > 0 {
		db = db.Where("id < ?", filter.GetBeforeId())
	}
	switch filter.GetFilterRuleKind() {
	case "buildIn":
		db = bizhelper.QueryByBool(db, "is_build_in_rule", true)
	case "unBuildIn":
		db = bizhelper.QueryByBool(db, "is_build_in_rule", false)
	}
	switch filter.GetFilterLibRuleKind() {
	case "lib":
		db = bizhelper.QueryByBool(db, "allow_included", true)
	case "noLib":
		db = bizhelper.QueryByBool(db, "allow_included", false)
	}
	return db
}

func applySyntaxFlowRuleModeFilter(db *gorm.DB, modes []string) *gorm.DB {
	if db == nil || len(modes) == 0 {
		return db
	}
	normalized := make([]string, 0, len(modes))
	for _, mode := range modes {
		if trimmed := strings.TrimSpace(mode); trimmed != "" {
			normalized = append(normalized, string(schema.ValidRuleMode(trimmed)))
		}
	}
	if len(normalized) == 0 {
		return db
	}
	return bizhelper.ExactOrQueryStringArrayOr(db, "mode", normalized)
}

func applySyntaxFlowRuleInput(ruleInput *ypb.SyntaxFlowRuleInput, rule *schema.SyntaxFlowRule) (*schema.SyntaxFlowRule, error) {
	if ruleInput == nil {
		return nil, utils.Error("parse syntaxflow rule failed: rule input is nil")
	}
	if rule == nil {
		rule = &schema.SyntaxFlowRule{}
	}

	if strings.TrimSpace(ruleInput.Language) != "" {
		language, err := ssaconfig.ValidateLanguage(ruleInput.Language)
		if err != nil {
			return nil, err
		}
		rule.Language = language
	} else if rule.Language == "" {
		if inferred := inferLanguageFromRuleName(ruleInput.RuleName); inferred != "" {
			rule.Language = inferred
		} else {
			rule.Language = ssaconfig.General
		}
	}
	if strings.TrimSpace(ruleInput.RuleName) != "" {
		rule.RuleName = ruleInput.RuleName
		if strings.TrimSpace(rule.Title) == "" {
			rule.Title = ruleInput.RuleName
		}
	}
	if len(ruleInput.Tags) > 0 {
		rule.Tag = strings.Join(ruleInput.Tags, "|")
	}
	if strings.TrimSpace(ruleInput.Description) != "" {
		rule.Description = ruleInput.Description
	}
	if rule.Content == "" {
		rule.Content = ruleInput.Content
	}
	rule.Version = sfdb.UpdateVersion(rule.Version)
	if rule.AlertDesc == nil {
		rule.AlertDesc = schema.MapEx[string, *schema.SyntaxFlowDescInfo]{}
	}
	for key, message := range ruleInput.AlertMsg {
		rule.AlertDesc[key] = schema.ToSyntaxFlowAlertDesc(message)
	}
	return rule, nil
}

func inferLanguageFromRuleName(ruleName string) ssaconfig.Language {
	if strings.TrimSpace(ruleName) == "" {
		return ""
	}
	base := filepath.Base(ruleName)
	languageRaw, _, ok := strings.Cut(base, "-")
	if !ok {
		return ""
	}
	language, err := ssaconfig.ValidateLanguage(languageRaw)
	if err != nil {
		return ""
	}
	return language
}
