package services

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const permCacheTTL = 5 * time.Minute

var permissionActions = []string{
	models.PermissionActionView,
	models.PermissionActionDownload,
	models.PermissionActionUpload,
	models.PermissionActionManage,
}

type CachedPermissions struct {
	FunctionCodes   []string
	LinePermissions map[uint]LinePerm
	ManagedLineIDs  []uint
	ExpiresAt       time.Time
}

type LinePerm struct {
	CanView     bool
	CanDownload bool
	CanUpload   bool
	CanManage   bool
	Source      string
}

type PermissionCell struct {
	Action         string `json:"action"`
	Setting        string `json:"setting"`
	SettingLabel   string `json:"setting_label"`
	Effective      string `json:"effective"`
	EffectiveLabel string `json:"effective_label"`
	Source         string `json:"source"`
	SourceLabel    string `json:"source_label"`
}

type PermissionMatrixLine struct {
	ResourceType string                    `json:"resource_type"`
	ResourceID   uint                      `json:"resource_id"`
	ResourceName string                    `json:"resource_name"`
	Actions      map[string]PermissionCell `json:"actions"`
}

type PermissionRuleChange struct {
	ResourceType string `json:"resource_type"`
	ResourceID   uint   `json:"resource_id"`
	Action       string `json:"action"`
	Decision     string `json:"decision"`
}

type LinePermissionBits struct {
	ProductionLineID uint
	CanView          bool
	CanDownload      bool
	CanUpload        bool
	CanManage        bool
	Source           string
}

type PermissionSubject struct {
	Type string
	ID   uint
	Key  string
}

type ResolvedLinePermission struct {
	ProductionLineID uint
	CanView          bool
	CanDownload      bool
	CanUpload        bool
	CanManage        bool
	Source           string
	Cells            map[string]PermissionCell
}

type rawDecision struct {
	Decision string
	Source   string
	Rank     int
}

var (
	permCache   sync.Map
	permCacheMu sync.Mutex
)

func decisionLabel(decision string) string {
	if decision == models.PermissionDecisionAllow {
		return "允许"
	}
	return "拒绝"
}

func sourceLabel(source string) string {
	switch source {
	case models.PermissionSubjectUser:
		return "单独设置"
	case models.PermissionSubjectDepartment:
		return "部门规则"
	case models.PermissionSubjectRole:
		return "角色规则"
	case "role_default":
		return "角色默认"
	case models.PermissionSubjectDepartmentDefault:
		return "部门默认规则"
	default:
		return "系统默认"
	}
}

func actionValid(action string) bool {
	for _, candidate := range permissionActions {
		if action == candidate {
			return true
		}
	}
	return false
}

func ruleKey(resourceID uint, action string) string {
	return fmt.Sprintf("%d:%s", resourceID, action)
}

func loadRuleMap(tx *gorm.DB, subjectType string, subjectID uint, subjectKey string, lineIDs []uint) (map[string]rawDecision, error) {
	result := map[string]rawDecision{}
	query := tx.Where("subject_type = ? AND resource_type = ? AND resource_id IN ?", subjectType, models.PermissionResourceProductionLine, lineIDs)
	if strings.TrimSpace(subjectKey) != "" {
		if subjectID > 0 {
			query = query.Where("(subject_id = ? AND subject_key IN (?, '')) OR (subject_id = 0 AND subject_key = ?)", subjectID, subjectKey, subjectKey)
		} else {
			query = query.Where("subject_id = 0 AND subject_key = ?", subjectKey)
		}
	} else {
		query = query.Where("subject_id = ? AND subject_key = ''", subjectID)
	}

	var rules []models.PermissionRule
	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}
	for _, rule := range rules {
		key := ruleKey(rule.ResourceID, rule.Action)
		source := rule.SubjectType
		if subjectType == models.PermissionSubjectRole && rule.SubjectID == 0 && strings.TrimSpace(rule.SubjectKey) != "" {
			source = "role_default"
		}
		decision := rawDecision{Decision: rule.Decision, Source: source, Rank: ruleScopeRank(rule, subjectID, subjectKey)}
		if existing, ok := result[key]; !ok || decision.Rank > existing.Rank {
			result[key] = decision
		}
	}
	return result, nil
}

// loadRoleDefaultRules 加载角色全局默认权限（resource_id=0），并展开到每个产线。
// role_default 规则存储时 resource_id=0 表示适用于所有产线。
func loadRoleDefaultRules(tx *gorm.DB, roleID uint, lineIDs []uint) (map[string]rawDecision, error) {
	result := map[string]rawDecision{}
	if roleID == 0 {
		return result, nil
	}

	var rules []models.PermissionRule
	if err := tx.Where("subject_type = ? AND subject_id = ? AND subject_key = '' AND resource_type = ? AND resource_id = 0",
		"role_default", roleID, models.PermissionResourceProductionLine).Find(&rules).Error; err != nil {
		return nil, err
	}

	// 展开到每个产线
	for _, rule := range rules {
		for _, lineID := range lineIDs {
			key := ruleKey(lineID, rule.Action)
			decision := rawDecision{Decision: rule.Decision, Source: "role_default", Rank: 0}
			if existing, ok := result[key]; !ok || decision.Rank > existing.Rank {
				result[key] = decision
			}
		}
	}
	return result, nil
}

func ruleScopeRank(rule models.PermissionRule, subjectID uint, subjectKey string) int {
	normalizedKey := strings.TrimSpace(subjectKey)
	if normalizedKey == "" {
		if rule.SubjectID == subjectID {
			return 1
		}
		return 0
	}
	if rule.SubjectID == subjectID && rule.SubjectKey == normalizedKey {
		return 3
	}
	if rule.SubjectID == subjectID && rule.SubjectKey == "" {
		return 2
	}
	if rule.SubjectID == 0 && rule.SubjectKey == normalizedKey {
		return 1
	}
	return 0
}

func permissionRuleScopeQuery(tx *gorm.DB, subject PermissionSubject, resourceType string, resourceID uint, action string) *gorm.DB {
	query := tx.Where("subject_type = ? AND resource_type = ? AND resource_id = ? AND action = ?", subject.Type, resourceType, resourceID, action)
	if subject.Type == models.PermissionSubjectRole && subject.ID > 0 && subject.Key != "" {
		return query.Where(
			"(subject_id = ? AND subject_key = ?) OR (subject_id = ? AND subject_key = '') OR (subject_id = 0 AND subject_key = ?)",
			subject.ID,
			subject.Key,
			subject.ID,
			subject.Key,
		)
	}
	return query.Where("subject_id = ? AND subject_key = ?", subject.ID, subject.Key)
}

func resolveCell(action string, resourceID uint, setting map[string]rawDecision, sources ...map[string]rawDecision) PermissionCell {
	key := ruleKey(resourceID, action)
	cell := PermissionCell{
		Action:         action,
		Setting:        "unset",
		SettingLabel:   "按规则",
		Effective:      models.PermissionDecisionDeny,
		EffectiveLabel: "拒绝",
		Source:         "system_default",
		SourceLabel:    "系统默认",
	}
	if own, ok := setting[key]; ok {
		cell.Setting = own.Decision
		cell.SettingLabel = decisionLabel(own.Decision)
	}
	for _, source := range sources {
		if decision, ok := source[key]; ok {
			cell.Effective = decision.Decision
			cell.EffectiveLabel = decisionLabel(decision.Decision)
			cell.Source = decision.Source
			cell.SourceLabel = sourceLabel(decision.Source)
			return cell
		}
	}
	return cell
}

func linePermFromCells(lineID uint, cells map[string]PermissionCell) ResolvedLinePermission {
	resolved := ResolvedLinePermission{ProductionLineID: lineID, Cells: cells, Source: "none"}
	resolved.CanView = cells[models.PermissionActionView].Effective == models.PermissionDecisionAllow
	resolved.CanDownload = cells[models.PermissionActionDownload].Effective == models.PermissionDecisionAllow
	resolved.CanUpload = cells[models.PermissionActionUpload].Effective == models.PermissionDecisionAllow
	resolved.CanManage = cells[models.PermissionActionManage].Effective == models.PermissionDecisionAllow
	for _, action := range permissionActions {
		if source := cells[action].Source; source != "system_default" {
			switch source {
			case models.PermissionSubjectUser:
				resolved.Source = "user"
			case models.PermissionSubjectDepartment:
				resolved.Source = "department"
			case models.PermissionSubjectRole:
				resolved.Source = "role"
			case "role_default":
				resolved.Source = "role_default"
			case models.PermissionSubjectDepartmentDefault:
				resolved.Source = "department_default"
			default:
				resolved.Source = source
			}
			return resolved
		}
	}
	return resolved
}

func ResolveUserProductionLinePermissions(user models.User, productionLines []models.ProductionLine) ([]ResolvedLinePermission, error) {
	lineIDs := make([]uint, 0, len(productionLines))
	for _, line := range productionLines {
		lineIDs = append(lineIDs, line.ID)
	}
	if len(lineIDs) == 0 {
		return []ResolvedLinePermission{}, nil
	}

	userRules, err := loadRuleMap(database.DB, models.PermissionSubjectUser, user.ID, "", lineIDs)
	if err != nil {
		return nil, err
	}

	departmentRules := map[string]rawDecision{}
	departmentDefaultRules := map[string]rawDecision{}
	if user.DepartmentID != nil {
		departmentRules, err = loadRuleMap(database.DB, models.PermissionSubjectDepartment, *user.DepartmentID, "", lineIDs)
		if err != nil {
			return nil, err
		}
		departmentDefaultRules, err = loadRuleMap(database.DB, models.PermissionSubjectDepartmentDefault, *user.DepartmentID, "", lineIDs)
		if err != nil {
			return nil, err
		}
	}

	roleID := uint(0)
	if user.RoleID != nil {
		roleID = *user.RoleID
	}
	roleRules, err := loadRuleMap(database.DB, models.PermissionSubjectRole, roleID, strings.TrimSpace(user.Role), lineIDs)
	if err != nil {
		return nil, err
	}

	// 加载角色全局默认权限（resource_id=0），展开到每个产线
	roleDefaultRules, err := loadRoleDefaultRules(database.DB, roleID, lineIDs)
	if err != nil {
		return nil, err
	}

	resolved := make([]ResolvedLinePermission, 0, len(productionLines))
	for _, line := range productionLines {
		cells := map[string]PermissionCell{}
		for _, action := range permissionActions {
			cells[action] = resolveCell(action, line.ID, userRules, userRules, departmentRules, roleRules, roleDefaultRules, departmentDefaultRules)
		}
		resolved = append(resolved, linePermFromCells(line.ID, cells))
	}
	return resolved, nil
}

func ResolveSubjectMatrix(user models.User, subject PermissionSubject, productionLines []models.ProductionLine) ([]PermissionMatrixLine, error) {
	resolved, err := ResolveUserProductionLinePermissions(user, productionLines)
	if err != nil {
		return nil, err
	}
	lineNames := map[uint]string{}
	for _, line := range productionLines {
		lineNames[line.ID] = line.Name
	}
	matrix := make([]PermissionMatrixLine, 0, len(resolved))
	for _, line := range resolved {
		matrix = append(matrix, PermissionMatrixLine{
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   line.ProductionLineID,
			ResourceName: lineNames[line.ProductionLineID],
			Actions:      line.Cells,
		})
	}
	return matrix, nil
}

func ResolveSubjectRuleMatrix(subject PermissionSubject, productionLines []models.ProductionLine) ([]PermissionMatrixLine, error) {
	subject = normalizeSubject(subject)
	lineIDs := make([]uint, 0, len(productionLines))
	lineNames := map[uint]string{}
	for _, line := range productionLines {
		lineIDs = append(lineIDs, line.ID)
		lineNames[line.ID] = line.Name
	}
	if len(lineIDs) == 0 {
		return []PermissionMatrixLine{}, nil
	}
	rules, err := loadRuleMap(database.DB, subject.Type, subject.ID, subject.Key, lineIDs)
	if err != nil {
		return nil, err
	}
	matrix := make([]PermissionMatrixLine, 0, len(productionLines))
	for _, line := range productionLines {
		cells := map[string]PermissionCell{}
		for _, action := range permissionActions {
			cells[action] = resolveCell(action, line.ID, rules, rules)
		}
		matrix = append(matrix, PermissionMatrixLine{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, ResourceName: lineNames[line.ID], Actions: cells})
	}
	return matrix, nil
}

func LoadSubjectLinePermissionBits(subject PermissionSubject, productionLines []models.ProductionLine) ([]LinePermissionBits, error) {
	subject = normalizeSubject(subject)
	lineIDs := make([]uint, 0, len(productionLines))
	for _, line := range productionLines {
		lineIDs = append(lineIDs, line.ID)
	}
	if len(lineIDs) == 0 {
		return []LinePermissionBits{}, nil
	}
	rules, err := loadRuleMap(database.DB, subject.Type, subject.ID, subject.Key, lineIDs)
	if err != nil {
		return nil, err
	}
	result := make([]LinePermissionBits, 0, len(productionLines))
	for _, line := range productionLines {
		bits := LinePermissionBits{ProductionLineID: line.ID, Source: "none"}
		for _, action := range permissionActions {
			decision, ok := rules[ruleKey(line.ID, action)]
			if !ok {
				continue
			}
			bits.Source = decision.Source
			allowed := decision.Decision == models.PermissionDecisionAllow
			switch action {
			case models.PermissionActionView:
				bits.CanView = allowed
			case models.PermissionActionDownload:
				bits.CanDownload = allowed
			case models.PermissionActionUpload:
				bits.CanUpload = allowed
			case models.PermissionActionManage:
				bits.CanManage = allowed
			}
		}
		if bits.Source != "none" {
			result = append(result, bits)
		}
	}
	return result, nil
}

func LoadSubjectLinePermissionBitsByLine(subject PermissionSubject, productionLineID uint) (LinePermissionBits, bool, error) {
	var lines []models.ProductionLine
	if err := database.DB.Select("id").Where("id = ?", productionLineID).Find(&lines).Error; err != nil {
		return LinePermissionBits{}, false, err
	}
	if len(lines) == 0 {
		return LinePermissionBits{}, false, nil
	}
	bits, err := LoadSubjectLinePermissionBits(subject, lines)
	if err != nil {
		return LinePermissionBits{}, false, err
	}
	if len(bits) == 0 {
		return LinePermissionBits{ProductionLineID: productionLineID, Source: "none"}, false, nil
	}
	return bits[0], true, nil
}

func normalizeSubject(subject PermissionSubject) PermissionSubject {
	subject.Key = strings.TrimSpace(subject.Key)
	if subject.Type != models.PermissionSubjectRole {
		subject.Key = ""
	}
	return subject
}

func SavePermissionRuleChangesTx(tx *gorm.DB, subject PermissionSubject, changes []PermissionRuleChange) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	subject = normalizeSubject(subject)
	for _, change := range changes {
		resourceType := strings.TrimSpace(change.ResourceType)
		if resourceType == "" {
			resourceType = models.PermissionResourceProductionLine
		}
		if resourceType != models.PermissionResourceProductionLine {
			return errors.New("invalid resource type")
		}
		if change.ResourceID == 0 {
			return errors.New("invalid resource id")
		}
		if !actionValid(change.Action) {
			return errors.New("invalid action")
		}
		decision := strings.TrimSpace(change.Decision)
		where := permissionRuleScopeQuery(tx, subject, resourceType, change.ResourceID, change.Action)
		if decision == "unset" || decision == "" {
			if err := where.Unscoped().Delete(&models.PermissionRule{}).Error; err != nil {
				return err
			}
			continue
		}
		if decision != models.PermissionDecisionAllow && decision != models.PermissionDecisionDeny {
			return errors.New("invalid decision")
		}
		if err := where.Unscoped().Delete(&models.PermissionRule{}).Error; err != nil {
			return err
		}
		rule := models.PermissionRule{SubjectType: subject.Type, SubjectID: subject.ID, SubjectKey: subject.Key, ResourceType: resourceType, ResourceID: change.ResourceID, Action: change.Action, Decision: decision}
		if err := tx.Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

func SavePermissionRuleChanges(subject PermissionSubject, changes []PermissionRuleChange) error {
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return SavePermissionRuleChangesTx(tx, subject, changes)
	}); err != nil {
		return err
	}
	InvalidateAllCache()
	return nil
}

func GetUserPermissions(userID uint) (*CachedPermissions, error) {
	if v, ok := permCache.Load(userID); ok {
		cached := v.(*CachedPermissions)
		if time.Now().Before(cached.ExpiresAt) {
			return cached, nil
		}
	}
	permCacheMu.Lock()
	defer permCacheMu.Unlock()
	if v, ok := permCache.Load(userID); ok {
		cached := v.(*CachedPermissions)
		if time.Now().Before(cached.ExpiresAt) {
			return cached, nil
		}
	}
	cached, err := loadUserPermissions(userID)
	if err != nil {
		return nil, err
	}
	permCache.Store(userID, cached)
	return cached, nil
}

func loadUserPermissions(userID uint) (*CachedPermissions, error) {
	var user models.User
	if err := database.DB.Select("id", "department_id", "role", "role_id").First(&user, userID).Error; err != nil {
		return nil, err
	}
	result := &CachedPermissions{FunctionCodes: []string{}, LinePermissions: map[uint]LinePerm{}, ManagedLineIDs: []uint{}, ExpiresAt: time.Now().Add(permCacheTTL)}
	if err := loadFunctionPermissions(userID, user.RoleID, result); err != nil {
		return nil, err
	}
	var lines []models.ProductionLine
	if err := database.DB.Select("id").Find(&lines).Error; err != nil {
		return nil, err
	}
	resolved, err := ResolveUserProductionLinePermissions(user, lines)
	if err != nil {
		return nil, err
	}
	for _, permission := range resolved {
		result.LinePermissions[permission.ProductionLineID] = LinePerm{CanView: permission.CanView, CanDownload: permission.CanDownload, CanUpload: permission.CanUpload, CanManage: permission.CanManage, Source: permission.Source}
	}
	if err := loadManagedLines(userID, result); err != nil {
		return nil, err
	}
	return result, nil
}

func loadFunctionPermissions(userID uint, roleID *uint, result *CachedPermissions) error {
	rolePermSet := map[uint]bool{}
	if roleID != nil {
		var rolePerms []models.RolePermission
		if err := database.DB.Where("role_id = ?", *roleID).Find(&rolePerms).Error; err != nil {
			return err
		}
		for _, rp := range rolePerms {
			rolePermSet[rp.PermissionID] = true
		}
	}
	overrides := map[uint]bool{}
	var userOverrides []models.UserPermissionOverride
	if err := database.DB.Where("user_id = ?", userID).Find(&userOverrides).Error; err != nil {
		return err
	}
	for _, uo := range userOverrides {
		overrides[uo.PermissionID] = uo.Granted
	}
	var allPerms []models.Permission
	if err := database.DB.Find(&allPerms).Error; err != nil {
		return err
	}
	permMap := make(map[uint]models.Permission, len(allPerms))
	for _, p := range allPerms {
		permMap[p.ID] = p
	}
	enabledIDs := map[uint]bool{}
	for pid := range rolePermSet {
		enabledIDs[pid] = true
	}
	for pid, granted := range overrides {
		if granted {
			enabledIDs[pid] = true
		} else {
			delete(enabledIDs, pid)
		}
	}
	for pid := range enabledIDs {
		if p, ok := permMap[pid]; ok {
			result.FunctionCodes = append(result.FunctionCodes, p.Code)
		}
	}
	return nil
}

func loadManagedLines(userID uint, result *CachedPermissions) error {
	var assignments []models.LineAdminAssignment
	if err := database.DB.Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		return err
	}
	for _, a := range assignments {
		result.ManagedLineIDs = append(result.ManagedLineIDs, a.ProductionLineID)
	}
	return nil
}

func InvalidateUserCache(userID uint) {
	permCache.Delete(userID)
}

func InvalidateAllCache() {
	permCache.Range(func(key, value any) bool {
		permCache.Delete(key)
		return true
	})
}

func UserHasPermission(userID uint, code string) bool {
	perms, err := GetUserPermissions(userID)
	if err != nil {
		return false
	}
	for _, c := range perms.FunctionCodes {
		if c == code {
			return true
		}
	}
	return false
}

func UserHasLinePermission(userID uint, lineID uint, action string) bool {
	perms, err := GetUserPermissions(userID)
	if err != nil {
		return false
	}
	lp, ok := perms.LinePermissions[lineID]
	if !ok {
		return false
	}
	switch action {
	case models.PermissionActionView:
		return lp.CanView || lp.CanManage
	case models.PermissionActionDownload:
		return lp.CanDownload || lp.CanManage
	case models.PermissionActionUpload:
		return lp.CanUpload || lp.CanManage
	case models.PermissionActionManage:
		return lp.CanManage
	default:
		return false
	}
}

func IsLineManager(userID uint, lineID uint) bool {
	perms, err := GetUserPermissions(userID)
	if err != nil {
		return false
	}
	for _, id := range perms.ManagedLineIDs {
		if id == lineID {
			return true
		}
	}
	return false
}
