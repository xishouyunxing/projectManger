package database

import (
	"crane-system/config"
	"crane-system/models"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SeedBaseData(cfg *config.Config) error {
	if DB == nil {
		return fmt.Errorf("database is not connected")
	}
	return SeedBaseDataWithDB(DB, cfg)
}

func SeedBaseDataWithDB(db *gorm.DB, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.AppConfig
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedDepartments(tx); err != nil {
			return err
		}
		if err := seedRoles(tx); err != nil {
			return err
		}
		if err := seedPermissions(tx); err != nil {
			return err
		}
		if err := seedDefaultRolePermissions(tx); err != nil {
			return err
		}
		if err := seedRoleDefaultRules(tx); err != nil {
			return err
		}
		if err := seedDepartmentDefaultRules(tx); err != nil {
			return err
		}
		return seedAdminUser(tx, cfg)
	})
}

func seedDepartments(tx *gorm.DB) error {
	departments := []models.Department{
		{Name: "IT部门", Description: "系统管理与平台维护", Status: "active"},
		{Name: "制造部", Description: "制造生产与执行管理", Status: "active"},
		{Name: "质量部", Description: "质量控制与检验管理", Status: "active"},
	}
	for _, department := range departments {
		if err := tx.Where(models.Department{Name: department.Name}).
			Attrs(models.Department{Description: department.Description, Status: department.Status}).
			FirstOrCreate(&models.Department{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedRoles(tx *gorm.DB) error {
	roles := []models.Role{
		{Name: "system_admin", Description: "系统管理员，全部权限", IsPreset: true, IsSystem: true, Status: "active", SortOrder: 1},
		{Name: "line_admin", Description: "产线管理员，可管理产线并编辑所有数据", IsPreset: true, Status: "active", SortOrder: 2},
		{Name: "offline_programmer", Description: "离线编程人员，可上传下载和编辑程序车型", IsPreset: true, Status: "active", SortOrder: 3},
		{Name: "field_operator", Description: "现场操作员，默认查看下载，可按产线指定权限", IsPreset: true, Status: "active", SortOrder: 4},
		{Name: "viewer", Description: "访客，产线只读", IsPreset: true, Status: "active", SortOrder: 5},
	}
	for _, role := range roles {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "name"}},
			DoUpdates: clause.Assignments(map[string]any{
				"description": role.Description,
				"is_preset":   role.IsPreset,
				"is_system":   role.IsSystem,
				"status":      role.Status,
				"sort_order":  role.SortOrder,
			}),
		}).Create(&role).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedPermissions(tx *gorm.DB) error {
	permissions := []models.Permission{
		{Code: "page:dashboard", Name: "仪表盘", Type: "page", Resource: "dashboard"},
		{Code: "page:programs", Name: "程序管理", Type: "page", Resource: "program"},
		{Code: "page:program_matrix", Name: "程序矩阵", Type: "page", Resource: "program"},
		{Code: "page:file_ignore_list", Name: "忽略文件列表", Type: "page", Resource: "file"},
		{Code: "page:user_management", Name: "用户管理", Type: "page", Resource: "user"},
		{Code: "page:production_lines", Name: "产线管理", Type: "page", Resource: "production_line"},
		{Code: "page:vehicle_models", Name: "车型管理", Type: "page", Resource: "vehicle_model"},
		{Code: "page:permissions", Name: "权限管理", Type: "page", Resource: "permission"},
		{Code: "page:system_management", Name: "系统管理", Type: "page", Resource: "system"},
		{Code: "op:program_create", Name: "创建程序", Type: "operation", Resource: "program"},
		{Code: "op:program_edit", Name: "编辑程序", Type: "operation", Resource: "program"},
		{Code: "op:program_delete", Name: "删除程序", Type: "operation", Resource: "program"},
		{Code: "op:program_export", Name: "导出Excel", Type: "operation", Resource: "program"},
		{Code: "op:file_upload", Name: "上传文件", Type: "operation", Resource: "file"},
		{Code: "op:file_download", Name: "下载文件", Type: "operation", Resource: "file"},
		{Code: "op:file_delete", Name: "删除文件", Type: "operation", Resource: "file"},
		{Code: "op:version_create", Name: "创建版本", Type: "operation", Resource: "version"},
		{Code: "op:version_manage", Name: "管理版本", Type: "operation", Resource: "version"},
		{Code: "op:user_create", Name: "创建用户", Type: "operation", Resource: "user"},
		{Code: "op:user_edit", Name: "编辑用户", Type: "operation", Resource: "user"},
		{Code: "op:user_delete", Name: "删除用户", Type: "operation", Resource: "user"},
		{Code: "op:password_reset", Name: "重置密码", Type: "operation", Resource: "user"},
		{Code: "op:backup_restore", Name: "备份恢复", Type: "operation", Resource: "system"},
		{Code: "op:line_permission_assign", Name: "分配产线权限", Type: "operation", Resource: "permission"},
	}
	for _, permission := range permissions {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "code"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "type", "resource", "description"}),
		}).Create(&permission).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDefaultRolePermissions(tx *gorm.DB) error {
	permIDs := map[string]uint{}
	var permissions []models.Permission
	if err := tx.Find(&permissions).Error; err != nil {
		return err
	}
	for _, permission := range permissions {
		permIDs[permission.Code] = permission.ID
	}

	rolePermDefs := map[string][]string{
		"system_admin": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
			"page:user_management", "page:production_lines", "page:vehicle_models",
			"page:permissions", "page:system_management",
			"op:program_create", "op:program_edit", "op:program_delete", "op:program_export",
			"op:file_upload", "op:file_download", "op:file_delete",
			"op:version_create", "op:version_manage",
			"op:user_create", "op:user_edit", "op:user_delete", "op:password_reset",
			"op:backup_restore", "op:line_permission_assign",
		},
		"line_admin": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
			"page:production_lines", "page:vehicle_models",
			"op:program_create", "op:program_edit", "op:program_delete", "op:program_export",
			"op:file_upload", "op:file_download", "op:file_delete",
			"op:version_create", "op:version_manage",
		},
		"offline_programmer": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
			"page:vehicle_models",
			"op:program_create", "op:program_edit", "op:program_export",
			"op:file_upload", "op:file_download",
			"op:version_create", "op:version_manage",
		},
		"field_operator": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
			"op:file_download",
		},
		"viewer": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
		},
	}

	for roleName, codes := range rolePermDefs {
		var role models.Role
		if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
			return err
		}
		for _, code := range codes {
			permID, ok := permIDs[code]
			if !ok {
				return fmt.Errorf("seed permission %s missing", code)
			}
			row := models.RolePermission{RoleID: role.ID, PermissionID: permID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedRoleDefaultRules(tx *gorm.DB) error {
	roleIDs := map[string]uint{}
	var roles []models.Role
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		roleIDs[role.Name] = role.ID
	}

	type seed struct {
		roleName string
		action   string
		decision string
	}
	seeds := []seed{
		{"field_operator", models.PermissionActionView, models.PermissionDecisionAllow},
		{"field_operator", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"offline_programmer", models.PermissionActionView, models.PermissionDecisionAllow},
		{"offline_programmer", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"offline_programmer", models.PermissionActionUpload, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionView, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionUpload, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionManage, models.PermissionDecisionAllow},
	}

	rules := make([]models.PermissionRule, 0, len(seeds))
	for _, item := range seeds {
		roleID, ok := roleIDs[item.roleName]
		if !ok {
			return fmt.Errorf("seed role %s missing", item.roleName)
		}
		rules = append(rules, models.PermissionRule{
			SubjectType:  "role_default",
			SubjectID:    roleID,
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   0,
			Action:       item.action,
			Decision:     item.decision,
		})
	}
	return insertMissingSeedPermissionRules(tx, rules)
}

func seedDepartmentDefaultRules(tx *gorm.DB) error {
	var dept models.Department
	if err := tx.Where("name = ?", "制造部").First(&dept).Error; err != nil {
		return err
	}
	rules := []models.PermissionRule{
		{
			SubjectType:  models.PermissionSubjectDepartmentDefault,
			SubjectID:    dept.ID,
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   0,
			Action:       models.PermissionActionView,
			Decision:     models.PermissionDecisionAllow,
		},
		{
			SubjectType:  models.PermissionSubjectDepartmentDefault,
			SubjectID:    dept.ID,
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   0,
			Action:       models.PermissionActionDownload,
			Decision:     models.PermissionDecisionAllow,
		},
	}
	return insertMissingSeedPermissionRules(tx, rules)
}

func upsertSeedPermissionRules(tx *gorm.DB, rules []models.PermissionRule) error {
	if len(rules) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
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

func insertMissingSeedPermissionRules(tx *gorm.DB, rules []models.PermissionRule) error {
	if len(rules) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "subject_type"},
			{Name: "subject_id"},
			{Name: "subject_key"},
			{Name: "resource_type"},
			{Name: "resource_id"},
			{Name: "action"},
		},
		DoNothing: true,
	}).Create(&rules).Error
}

func seedAdminUser(tx *gorm.DB, cfg *config.Config) error {
	if cfg == nil || cfg.Auth.DefaultPassword == "" {
		return nil
	}

	var role models.Role
	if err := tx.Where("name = ?", "system_admin").First(&role).Error; err != nil {
		return err
	}
	var department models.Department
	if err := tx.Where("name = ?", "IT部门").First(&department).Error; err != nil {
		return err
	}

	password, err := bcrypt.GenerateFromPassword([]byte(cfg.Auth.DefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admin := models.User{
		EmployeeID:   "admin001",
		EmployeeNo:   "admin001",
		Name:         "系统管理员",
		DepartmentID: &department.ID,
		Role:         "system_admin",
		RoleID:       &role.ID,
		Password:     string(password),
		Status:       "active",
	}

	var existing models.User
	if err := tx.Where("employee_id = ?", admin.EmployeeID).First(&existing).Error; err == nil {
		updates := map[string]any{
			"employee_no":   admin.EmployeeNo,
			"name":          admin.Name,
			"department_id": department.ID,
			"role":          admin.Role,
			"role_id":       role.ID,
			"status":        admin.Status,
		}
		if existing.Password == "" {
			updates["password"] = admin.Password
		}
		return tx.Model(&existing).Updates(updates).Error
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	return tx.Create(&admin).Error
}
