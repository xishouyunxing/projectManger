package database

import (
	"crane-system/config"
	"crane-system/models"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

const (
	migrationLockName               = "crane_system_migrations"
	migrationLockTimeoutSeconds     = 30
	migrationStepSchemaBootstrap    = "schema_bootstrap"
	migrationStepBaseSeed           = "base_seed"
	migrationStepPermissionBackfill = "legacy_permission_backfill"
)

type SchemaMigration struct {
	ID        uint      `gorm:"primarykey"`
	Name      string    `gorm:"size:100;not null;uniqueIndex"`
	AppliedAt time.Time `gorm:"not null"`
}

func (SchemaMigration) TableName() string {
	return "schema_migrations"
}

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
		&SchemaMigration{},
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
	return ensureTablesWithDB(DB)
}

func ensureTablesWithDB(db *gorm.DB) error {
	models := migrationModels()
	var failed []string

	for _, m := range models {
		if err := ensureModelSchema(db, m); err != nil {
			slog.Error("schema migration failed", "model", fmt.Sprintf("%T", m), "error", err)
			failed = append(failed, fmt.Sprintf("%T", m))
		}
	}

	var missing []string
	for _, m := range models {
		if !db.Migrator().HasTable(m) {
			missing = append(missing, fmt.Sprintf("%T", m))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("tables were not created for models: %v", missing)
	}

	if len(failed) > 0 {
		return fmt.Errorf("schema migrations failed for models: %v", failed)
	}

	return nil
}

func ensureModelSchema(db *gorm.DB, model any) error {
	if !db.Migrator().HasTable(model) {
		return db.Migrator().CreateTable(model)
	}

	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return err
	}

	for _, field := range stmt.Schema.Fields {
		if field.DBName == "" {
			continue
		}
		if !db.Migrator().HasColumn(model, field.DBName) {
			if err := db.Migrator().AddColumn(model, field.Name); err != nil {
				return fmt.Errorf("add column %s: %w", field.DBName, err)
			}
		}
	}

	for _, idx := range stmt.Schema.ParseIndexes() {
		if idx.Name == "" || db.Migrator().HasIndex(model, idx.Name) {
			continue
		}
		if err := db.Migrator().CreateIndex(model, idx.Name); err != nil {
			return fmt.Errorf("create index %s: %w", idx.Name, err)
		}
	}

	return nil
}

func runWithMigrationLock(fn func(*gorm.DB) error) error {
	if DB == nil {
		return errors.New("database is not connected")
	}
	if DB.Dialector.Name() != "mysql" {
		return fn(DB)
	}

	return DB.Connection(func(db *gorm.DB) error {
		if err := acquireMigrationLock(db); err != nil {
			return err
		}
		defer func() {
			if err := releaseMigrationLock(db); err != nil {
				slog.Error("release migration lock failed", "error", err)
			}
		}()

		return fn(db)
	})
}

func acquireMigrationLock(db *gorm.DB) error {
	var result int
	if err := db.Raw("SELECT GET_LOCK(?, ?)", migrationLockName, migrationLockTimeoutSeconds).Scan(&result).Error; err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("could not acquire migration lock %q", migrationLockName)
	}
	return nil
}

func releaseMigrationLock(db *gorm.DB) error {
	var result any
	return db.Raw("SELECT RELEASE_LOCK(?)", migrationLockName).Scan(&result).Error
}

func recordMigrationStep(db *gorm.DB, name string) error {
	step := SchemaMigration{Name: name, AppliedAt: time.Now()}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&step).Error
}

func migrationStepRecorded(db *gorm.DB, name string) (bool, error) {
	var count int64
	if err := db.Model(&SchemaMigration{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// backfillUserRoleIDs 在 Go 层完成 role 字符串 → role_id 的映射。
// 旧数据中用户只有 role 字段（如 "admin"），没有 role_id。
// 这里根据 role 名称查找 Role 表，自动回填 role_id。
// 旧角色名映射：admin → system_admin，其余保持原名。
func backfillUserRoleIDs() error {
	return backfillUserRoleIDsWithDB(DB)
}

func backfillUserRoleIDsWithDB(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&models.User{}, "role_id") {
		return nil
	}

	var roles []models.Role
	if err := db.Find(&roles).Error; err != nil {
		return fmt.Errorf("load roles for role_id backfill: %w", err)
	}
	roleMap := map[string]uint{}
	for _, r := range roles {
		roleMap[r.Name] = r.ID
	}

	legacyMap := map[string]string{
		"admin":    "system_admin",
		"user":     "viewer",
		"operator": "field_operator",
		"engineer": "offline_programmer",
	}

	var users []models.User
	if err := db.Where("role_id IS NULL").Find(&users).Error; err != nil {
		return fmt.Errorf("query users needing role_id backfill: %w", err)
	}

	for _, u := range users {
		roleName := u.Role
		if mapped, ok := legacyMap[roleName]; ok {
			roleName = mapped
		}
		roleID, ok := roleMap[roleName]
		if !ok {
			continue
		}
		if err := db.Model(&models.User{}).Where("id = ?", u.ID).Update("role_id", roleID).Error; err != nil {
			return err
		}
	}

	if len(users) > 0 {
		slog.Info("backfilled user role_id", "count", len(users))
	}
	return nil
}

func ValidateSchema() error {
	return ValidateSchemaWithDB(DB)
}

func ValidateSchemaWithDB(db *gorm.DB) error {
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
		if !db.Migrator().HasTable(check.table) {
			return fmt.Errorf("schema validation failed: table %s is missing", check.name)
		}
		if !db.Migrator().HasColumn(check.table, check.column) {
			return fmt.Errorf("schema validation failed: table %s missing column %s", check.name, check.column)
		}
	}

	if !db.Migrator().HasIndex(&models.ProductionLineCustomField{}, "idx_production_line_custom_fields_line_name") {
		return fmt.Errorf("schema validation failed: table production_line_custom_fields missing index idx_production_line_custom_fields_line_name")
	}

	if !db.Migrator().HasIndex(&models.ProgramCustomFieldValue{}, "idx_program_custom_field_values_program_field") {
		return fmt.Errorf("schema validation failed: table program_custom_field_values missing index idx_program_custom_field_values_program_field")
	}

	if !db.Migrator().HasIndex(&models.PermissionRule{}, "idx_permission_rule_scope") {
		return fmt.Errorf("schema validation failed: table permission_rules missing index idx_permission_rule_scope")
	}

	if err := validateCriticalColumnDefinitions(db); err != nil {
		return err
	}

	return nil
}

func validateCriticalColumnDefinitions(db *gorm.DB) error {
	notNull := false
	checks := []struct {
		table          any
		tableName      string
		column         string
		expectNullable *bool
	}{
		{&models.User{}, "users", "employee_id", nil},
		{&models.Role{}, "roles", "name", &notNull},
		{&models.Permission{}, "permissions", "code", &notNull},
		{&models.PermissionRule{}, "permission_rules", "subject_type", &notNull},
		{&models.PermissionRule{}, "permission_rules", "subject_id", &notNull},
		{&models.PermissionRule{}, "permission_rules", "subject_key", &notNull},
		{&models.PermissionRule{}, "permission_rules", "resource_type", &notNull},
		{&models.PermissionRule{}, "permission_rules", "resource_id", &notNull},
		{&models.PermissionRule{}, "permission_rules", "action", &notNull},
		{&models.Program{}, "programs", "production_line_id", &notNull},
		{&models.ProductionLineCustomField{}, "production_line_custom_fields", "production_line_id", &notNull},
		{&models.ProgramCustomFieldValue{}, "program_custom_field_values", "program_id", &notNull},
		{&models.ProgramCustomFieldValue{}, "program_custom_field_values", "production_line_custom_field_id", &notNull},
	}

	for _, check := range checks {
		types, err := db.Migrator().ColumnTypes(check.table)
		if err != nil {
			return fmt.Errorf("schema validation failed: inspect table %s: %w", check.tableName, err)
		}
		found := false
		for _, columnType := range types {
			if !strings.EqualFold(columnType.Name(), check.column) {
				continue
			}
			found = true
			nullable, ok := columnType.Nullable()
			if check.expectNullable != nil && ok && nullable != *check.expectNullable {
				return fmt.Errorf("schema validation failed: table %s column %s nullable=%v, want %v; run an explicit migration", check.tableName, check.column, nullable, *check.expectNullable)
			}
			break
		}
		if !found {
			return fmt.Errorf("schema validation failed: table %s missing column %s", check.tableName, check.column)
		}
	}
	return nil
}

func AutoMigrate() error {
	return runWithMigrationLock(func(db *gorm.DB) error {
		if err := ensureTablesWithDB(db); err != nil {
			return err
		}
		if err := recordMigrationStep(db, migrationStepSchemaBootstrap); err != nil {
			return err
		}

		if err := SeedBaseDataWithDB(db, config.AppConfig); err != nil {
			return err
		}
		if err := recordMigrationStep(db, migrationStepBaseSeed); err != nil {
			return err
		}

		if err := backfillUserRoleIDsWithDB(db); err != nil {
			return err
		}

		backfilled, err := migrationStepRecorded(db, migrationStepPermissionBackfill)
		if err != nil {
			return err
		}
		if !backfilled {
			if err := migrateLegacyPermissionRulesWithDB(db); err != nil {
				return err
			}
			if err := recordMigrationStep(db, migrationStepPermissionBackfill); err != nil {
				return err
			}
		}

		return ValidateSchemaWithDB(db)
	})
}

func migrateLegacyPermissionRules() error {
	return migrateLegacyPermissionRulesWithDB(DB)
}

func migrateLegacyPermissionRulesWithDB(db *gorm.DB) error {
	if err := migrateUserPermissionRules(db); err != nil {
		return err
	}
	if err := migrateDepartmentPermissionRules(db); err != nil {
		return err
	}
	if err := migrateRoleLinePermissionRules(db); err != nil {
		return err
	}
	if err := migrateRoleDefaultPermissionRules(db); err != nil {
		return err
	}
	return migrateDepartmentDefaultPermissionRules(db)
}

func migrateUserPermissionRules(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.UserPermission{}) {
		return nil
	}
	var rows []models.UserPermission
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSetWithDB(db, models.PermissionSubjectUser, row.UserID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateDepartmentPermissionRules(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.DepartmentPermission{}) {
		return nil
	}
	var rows []models.DepartmentPermission
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSetWithDB(db, models.PermissionSubjectDepartment, row.DepartmentID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateRoleLinePermissionRules(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.RoleLinePermission{}) {
		return nil
	}
	var rows []models.RoleLinePermission
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSetWithDB(db, models.PermissionSubjectRole, row.RoleID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateRoleDefaultPermissionRules(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.RoleDefaultPermission{}) {
		return nil
	}
	var rows []models.RoleDefaultPermission
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSetWithDB(db, models.PermissionSubjectRole, 0, row.Role, row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func migrateDepartmentDefaultPermissionRules(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.DepartmentDefaultPermission{}) {
		return nil
	}
	var rows []models.DepartmentDefaultPermission
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := upsertPermissionRuleSetWithDB(db, models.PermissionSubjectDepartmentDefault, row.DepartmentID, "", row.ProductionLineID, row.CanView, row.CanDownload, row.CanUpload, row.CanManage); err != nil {
			return err
		}
	}
	return nil
}

func upsertPermissionRuleSet(subjectType string, subjectID uint, subjectKey string, lineID uint, canView, canDownload, canUpload, canManage bool) error {
	return upsertPermissionRuleSetWithDB(DB, subjectType, subjectID, subjectKey, lineID, canView, canDownload, canUpload, canManage)
}

func upsertPermissionRuleSetWithDB(db *gorm.DB, subjectType string, subjectID uint, subjectKey string, lineID uint, canView, canDownload, canUpload, canManage bool) error {
	rules := []models.PermissionRule{
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionView, canView),
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionDownload, canDownload),
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionUpload, canUpload),
		buildPermissionRule(subjectType, subjectID, subjectKey, lineID, models.PermissionActionManage, canManage),
	}
	return db.Clauses(clause.OnConflict{
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
