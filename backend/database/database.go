package database

import (
	"crane-system/config"
	"crane-system/models"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Host,
		config.AppConfig.Database.Port,
		config.AppConfig.Database.Name,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(gormLogLevel()),
	})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	return nil
}

// gormLogLevel 根据 APP_ENV 返回合适的 GORM 日志级别。
func gormLogLevel() logger.LogLevel {
	if config.AppConfig != nil && config.AppConfig.App.Env == "production" {
		return logger.Warn
	}
	return logger.Info
}

// Close 关闭数据库连接池。
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func migrationModels() []any {
	return []any{
		&models.Department{},
		&models.Process{},
		&models.VehicleModel{},
		&models.User{},
		&models.ProductionLine{},
		&models.ProductionLineCustomField{},
		&models.Program{},
		&models.ProgramCustomFieldValue{},
		&models.ProgramFile{},
		&models.ProgramVersion{},
		&models.ProgramRelation{},
		&models.ProgramMapping{},
		&models.UserPermission{},
		&models.DepartmentPermission{},
		&models.RoleDefaultPermission{},
		&models.DepartmentDefaultPermission{},
		&models.PermissionRule{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserPermissionOverride{},
		&models.RoleLinePermission{},
		&models.LineAdminAssignment{},
	}
}

func ensureTables() error {
	models := migrationModels()
	var failed []string

	// 逐模型迁移，记录失败的模型但不阻塞后续建表
	for _, m := range models {
		if err := DB.AutoMigrate(m); err != nil {
			slog.Warn("AutoMigrate 模型失败，将重试", "model", fmt.Sprintf("%T", m), "error", err)
			// 重试一次：可能是 GORM 尝试 DROP 旧约束时先失败，但第二次表已部分创建
			if retryErr := DB.AutoMigrate(m); retryErr != nil {
				slog.Error("AutoMigrate 模型重试仍失败", "model", fmt.Sprintf("%T", m), "error", retryErr)
				failed = append(failed, fmt.Sprintf("%T", m))
			}
		}
	}

	// 验证所有表是否已创建
	var missing []string
	for _, m := range models {
		if !DB.Migrator().HasTable(m) {
			missing = append(missing, fmt.Sprintf("%T", m))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("以下模型的表未成功创建: %v", missing)
	}

	if len(failed) > 0 {
		slog.Warn("部分模型迁移时遇到错误（表已创建）", "models", failed)
	}

	return nil
}

// backfillUserRoleIDs 在 Go 层完成 role 字符串 → role_id 的映射。
// 旧数据中用户只有 role 字段（如 "admin"），没有 role_id。
// 这里根据 role 名称查找 Role 表，自动回填 role_id。
// 旧角色名映射：admin → system_admin，其余保持原名。
func backfillUserRoleIDs() error {
	// 跳过：role_id 列尚不存在（AutoMigrate 会加列，但回填应在加列之后）
	if !DB.Migrator().HasColumn(&models.User{}, "role_id") {
		return nil
	}

	// 加载所有角色到 map[name]ID
	var roles []models.Role
	if err := DB.Find(&roles).Error; err != nil {
		return fmt.Errorf("加载角色列表失败: %w", err)
	}
	roleMap := map[string]uint{}
	for _, r := range roles {
		roleMap[r.Name] = r.ID
	}

	// 旧角色名 → 新角色名
	legacyMap := map[string]string{
		"admin":    "system_admin",
		"user":     "viewer",
		"operator": "field_operator",
		"engineer": "offline_programmer",
	}

	// 查询所有 role_id 为空的用户
	var users []models.User
	if err := DB.Where("role_id IS NULL").Find(&users).Error; err != nil {
		return fmt.Errorf("查询待回填用户失败: %w", err)
	}

	for _, u := range users {
		roleName := u.Role
		if mapped, ok := legacyMap[roleName]; ok {
			roleName = mapped
		}
		roleID, ok := roleMap[roleName]
		if !ok {
			continue // 未知角色，跳过
		}
		DB.Model(&models.User{}).Where("id = ?", u.ID).Update("role_id", roleID)
	}

	if len(users) > 0 {
		slog.Info("已回填用户的 role_id", "count", len(users))
	}
	return nil
}

func ValidateSchema() error {
	checks := []struct {
		table  any
		name   string
		column string
	}{
		{&models.User{}, "users", "employee_id"},
		{&models.User{}, "users", "department_id"},
		{&models.Department{}, "departments", "name"},
		{&models.ProductionLine{}, "production_lines", "code"},
		{&models.VehicleModel{}, "vehicle_models", "code"},
		{&models.Program{}, "programs", "production_line_id"},
		{&models.ProductionLineCustomField{}, "production_line_custom_fields", "production_line_id"},
		{&models.ProductionLineCustomField{}, "production_line_custom_fields", "name"},
		{&models.ProgramCustomFieldValue{}, "program_custom_field_values", "program_id"},
		{&models.ProgramCustomFieldValue{}, "program_custom_field_values", "production_line_custom_field_id"},
	}

	for _, check := range checks {
		if !DB.Migrator().HasTable(check.table) {
			return fmt.Errorf("schema validation failed: table %s is missing", check.name)
		}
		if !DB.Migrator().HasColumn(check.table, check.column) {
			return fmt.Errorf("schema validation failed: table %s missing column %s", check.name, check.column)
		}
	}

	if !DB.Migrator().HasIndex(&models.ProductionLineCustomField{}, "idx_production_line_custom_fields_line_name") {
		return fmt.Errorf("schema validation failed: table production_line_custom_fields missing index idx_production_line_custom_fields_line_name")
	}

	if !DB.Migrator().HasIndex(&models.ProgramCustomFieldValue{}, "idx_program_custom_field_values_program_field") {
		return fmt.Errorf("schema validation failed: table program_custom_field_values missing index idx_program_custom_field_values_program_field")
	}

	if !DB.Migrator().HasIndex(&models.PermissionRule{}, "idx_permission_rule_scope") {
		return fmt.Errorf("schema validation failed: table permission_rules missing index idx_permission_rule_scope")
	}

	return nil
}

func AutoMigrate() error {
	if err := ensureTables(); err != nil {
		return err
	}

	if err := backfillUserRoleIDs(); err != nil {
		return err
	}

	if err := migrateLegacyPermissionRules(); err != nil {
		return err
	}

	return ValidateSchema()
}

func migrateLegacyPermissionRules() error {
	if err := migrateUserPermissionRules(); err != nil {
		return err
	}
	if err := migrateDepartmentPermissionRules(); err != nil {
		return err
	}
	if err := migrateRoleLinePermissionRules(); err != nil {
		return err
	}
	if err := migrateRoleDefaultPermissionRules(); err != nil {
		return err
	}
	return migrateDepartmentDefaultPermissionRules()
}

func migrateUserPermissionRules() error {
	if !DB.Migrator().HasTable(&models.UserPermission{}) {
		return nil
	}
	var rows []models.UserPermission
	if err := DB.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSet(models.PermissionSubjectUser, row.UserID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateDepartmentPermissionRules() error {
	if !DB.Migrator().HasTable(&models.DepartmentPermission{}) {
		return nil
	}
	var rows []models.DepartmentPermission
	if err := DB.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSet(models.PermissionSubjectDepartment, row.DepartmentID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateRoleLinePermissionRules() error {
	if !DB.Migrator().HasTable(&models.RoleLinePermission{}) {
		return nil
	}
	var rows []models.RoleLinePermission
	if err := DB.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSet(models.PermissionSubjectRole, row.RoleID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateRoleDefaultPermissionRules() error {
	if !DB.Migrator().HasTable(&models.RoleDefaultPermission{}) {
		return nil
	}
	var rows []models.RoleDefaultPermission
	if err := DB.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSet(models.PermissionSubjectRole, 0, row.Role, row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateDepartmentDefaultPermissionRules() error {
	if !DB.Migrator().HasTable(&models.DepartmentDefaultPermission{}) {
		return nil
	}
	var rows []models.DepartmentDefaultPermission
	if err := DB.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSet(models.PermissionSubjectDepartmentDefault, row.DepartmentID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func upsertPermissionRuleSet(subjectType string, subjectID uint, subjectKey string, lineID uint, canView, canDownload, canUpload, canManage bool) error {
	rules := []models.PermissionRule{
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionView, canView),
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionDownload, canDownload),
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionUpload, canUpload),
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionManage, canManage),
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "subject_type"},
			{Name: "subject_id"},
			{Name: "subject_key"},
			{Name: "resource_type"},
			{Name: "resource_id"},
			{Name: "action"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"decision", "updated_at"}),
	}).Create(&rules).Error
}

func buildPermissionRule(subjectType string, subjectID uint, subjectKey string, lineID uint, action string, allowed bool) models.PermissionRule {
	decision := models.PermissionDecisionDeny
	if allowed {
		decision = models.PermissionDecisionAllow
	}
	return models.PermissionRule{
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		SubjectKey:   subjectKey,
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   lineID,
		Action:       action,
		Decision:     decision,
	}
}
