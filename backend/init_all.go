package main

import (
	"crane-system/app"
	"crane-system/config"
	"crane-system/database"
	"crane-system/models"
	"golang.org/x/crypto/bcrypt"
	"log"

	"gorm.io/gorm/clause"
)

func InitAll() {
	log.Println("开始初始化系统数据...")

	cfg, err := app.SetupInfrastructure()
	if err != nil {
		log.Fatal("初始化基础设施失败:", err)
	}

	if err := database.AutoMigrate(); err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	createDepartments()
	createRoles()
	createPermissions()
	seedRoleDefaults()
	seedDepartmentDefaults()
	createAdmin(cfg)

	log.Println("系统数据初始化完成")
	log.Println("已初始化：数据库结构、部门、角色、权限、管理员账号")
	log.Println("未初始化：工序、车型、生产线，请在系统中按业务需要手工录入")
	log.Println("默认登录信息:")
	log.Println("  工号: admin001")
	log.Printf("  密码: %s", cfg.Auth.DefaultPassword)
}

func createAdmin(cfg *config.Config) {
	var adminDepartment models.Department
	if err := database.DB.Where("name = ?", "IT部门").First(&adminDepartment).Error; err != nil {
		log.Printf("查询管理员部门失败: %v", err)
		return
	}

	// 查找 system_admin 角色
	var adminRole models.Role
	if err := database.DB.Where("name = ?", "system_admin").First(&adminRole).Error; err != nil {
		log.Printf("查询 system_admin 角色失败: %v", err)
		return
	}

	password := cfg.Auth.DefaultPassword
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		return
	}

	adminRoleID := adminRole.ID
	admin := models.User{
		EmployeeID:   "admin001",
		EmployeeNo:   "admin001",
		Name:         "系统管理员",
		DepartmentID: &adminDepartment.ID,
		Role:         "system_admin",
		RoleID:       &adminRoleID,
		Password:     string(hashedPassword),
		Status:       "active",
	}

	var existingUser models.User
	result := database.DB.Where("employee_id = ?", admin.EmployeeID).First(&existingUser)
	if result.Error == nil {
		// 已存在则更新 role 和 role_id
		database.DB.Model(&existingUser).Updates(map[string]interface{}{
			"role":    "system_admin",
			"role_id": adminRoleID,
		})
		log.Printf("管理员账号已存在，已更新角色关联")
		return
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Printf("创建管理员失败: %v", err)
		return
	}

	log.Printf("创建管理员账号成功")
}

// createRoles 创建预设角色。
func createRoles() {
	roles := []models.Role{
		{Name: "system_admin", Description: "系统管理员，全部权限", IsPreset: true, IsSystem: true, Status: "active", SortOrder: 1},
		{Name: "line_admin", Description: "产线管理员，可管理产线并编辑所有数据", IsPreset: true, IsSystem: false, Status: "active", SortOrder: 2},
		{Name: "offline_programmer", Description: "离线编程人员，可上传下载和编辑程序车型", IsPreset: true, IsSystem: false, Status: "active", SortOrder: 3},
		{Name: "field_operator", Description: "现场操作员，默认查看下载，可按产线指定权限", IsPreset: true, IsSystem: false, Status: "active", SortOrder: 4},
		// 保留旧角色用于兼容，新建用户不再使用
		{Name: "viewer", Description: "访客，产线只读", IsPreset: true, IsSystem: false, Status: "active", SortOrder: 5},
	}

	for _, role := range roles {
		var existing models.Role
		if database.DB.Where("name = ?", role.Name).First(&existing).Error == nil {
			// 更新已有预设角色的描述
			database.DB.Model(&existing).Update("description", role.Description)
			continue
		}
		database.DB.Create(&role)
	}
	log.Println("预设角色初始化完成")
}

// createPermissions 创建功能权限定义，并为预设角色分配默认权限。
func createPermissions() {
	// 定义所有功能权限
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

	for _, perm := range permissions {
		var existing models.Permission
		if database.DB.Where("code = ?", perm.Code).First(&existing).Error == nil {
			continue
		}
		database.DB.Create(&perm)
	}

	// 为预设角色分配默认功能权限
	assignDefaultRolePermissions()
	log.Println("功能权限初始化完成")
}

// assignDefaultRolePermissions 为预设角色分配默认功能权限。
func assignDefaultRolePermissions() {
	// 加载所有权限
	var allPerms []models.Permission
	database.DB.Find(&allPerms)
	permMap := map[string]uint{}
	for _, p := range allPerms {
		permMap[p.Code] = p.ID
	}

	// 定义每个角色的功能权限
	rolePermDefs := map[string][]string{
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
		// 旧角色保持兼容
		"engineer": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
			"op:program_create", "op:program_edit", "op:program_export",
			"op:file_upload", "op:file_download",
			"op:version_create", "op:version_manage",
		},
		"operator": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
			"op:file_download",
		},
		"viewer": {
			"page:dashboard", "page:programs", "page:program_matrix", "page:file_ignore_list",
		},
	}

	for roleName, permCodes := range rolePermDefs {
		var role models.Role
		if err := database.DB.Where("name = ?", roleName).First(&role).Error; err != nil {
			continue
		}

		for _, code := range permCodes {
			permID, ok := permMap[code]
			if !ok {
				continue
			}

			var existing models.RolePermission
			if database.DB.Where("role_id = ? AND permission_id = ?", role.ID, permID).First(&existing).Error == nil {
				continue
			}
			database.DB.Create(&models.RolePermission{
				RoleID:       role.ID,
				PermissionID: permID,
			})
		}
	}
}

func createDepartments() {
	departments := []models.Department{
		{Name: "IT部门", Description: "系统管理与平台维护", Status: "active"},
		{Name: "制造部", Description: "制造生产与执行管理", Status: "active"},
		{Name: "质量部", Description: "质量控制与检验管理", Status: "active"},
	}

	for _, department := range departments {
		var existing models.Department
		if database.DB.Where("name = ?", department.Name).First(&existing).Error == nil {
			continue
		}
		database.DB.Create(&department)
	}
	log.Printf("创建部门数据成功")
}

func createProcesses() {
	processes := []models.Process{
		{Name: "吊臂制造", Code: "UP001", Type: "upper", SortOrder: 1, Description: "上车吊臂制造工序"},
		{Name: "转台制造", Code: "UP002", Type: "upper", SortOrder: 2, Description: "上车转台制造工序"},
		{Name: "底盘制造", Code: "LOW001", Type: "lower", SortOrder: 1, Description: "下车底盘制造工序"},
		{Name: "支腿制造", Code: "LOW002", Type: "lower", SortOrder: 2, Description: "下车支腿制造工序"},
	}

	for _, process := range processes {
		var existing models.Process
		if database.DB.Where("code = ?", process.Code).First(&existing).Error == nil {
			continue
		}
		database.DB.Create(&process)
	}
	log.Printf("创建工序数据成功")
}

func createVehicleModels() {
	vehicles := []models.VehicleModel{
		{Name: "25吨汽车起重机", Code: "QC25", Series: "QC系列", Description: "25吨汽车起重机，适用于中小型建筑工地", Status: "active"},
		{Name: "50吨汽车起重机", Code: "QC50", Series: "QC系列", Description: "50吨汽车起重机，适用于中型工程项目", Status: "active"},
		{Name: "80吨汽车起重机", Code: "QC80", Series: "QC系列", Description: "80吨汽车起重机，适用于大型工程项目", Status: "active"},
		{Name: "100吨汽车起重机", Code: "QC100", Series: "QC系列", Description: "100吨汽车起重机，适用于重型工程项目", Status: "active"},
	}

	for _, vehicle := range vehicles {
		var existing models.VehicleModel
		if database.DB.Where("code = ?", vehicle.Code).First(&existing).Error == nil {
			continue
		}
		database.DB.Create(&vehicle)
	}
	log.Printf("创建车型数据成功")
}

func createProductionLines() {
	var processes []models.Process
	database.DB.Find(&processes)

	processMap := make(map[string]uint)
	for _, p := range processes {
		processMap[p.Code] = p.ID
	}

	optionalProcessID := func(code string) *uint {
		if id, ok := processMap[code]; ok {
			return &id
		}
		return nil
	}

	lines := []models.ProductionLine{
		{Name: "吊臂主臂生产线", Code: "UP_ARM_001", Type: "upper", ProcessID: optionalProcessID("UP001"), Description: "主要负责起重机吊臂主臂的制造和装配", Status: "active"},
		{Name: "吊臂副臂生产线", Code: "UP_ARM_002", Type: "upper", ProcessID: optionalProcessID("UP001"), Description: "主要负责起重机吊臂副臂的制造和装配", Status: "active"},
		{Name: "转台装配生产线", Code: "UP_TURN_001", Type: "upper", ProcessID: optionalProcessID("UP002"), Description: "负责起重机转台的整体装配", Status: "active"},
		{Name: "底盘焊接生产线", Code: "LOW_CHASSIS_001", Type: "lower", ProcessID: optionalProcessID("LOW001"), Description: "负责起重机底盘的焊接和初步成型", Status: "active"},
		{Name: "支腿液压生产线", Code: "LOW_LEG_001", Type: "lower", ProcessID: optionalProcessID("LOW002"), Description: "负责起重机支腿液压系统的制造", Status: "active"},
	}

	for _, line := range lines {
		var existing models.ProductionLine
		if database.DB.Where("code = ?", line.Code).First(&existing).Error == nil {
			continue
		}
		database.DB.Create(&line)
	}
	log.Printf("创建生产线数据成功")
}

// seedRoleDefaults 为预设角色创建全局默认产线权限（subject_type="role_default", resource_id=0）。
// 这些规则在权限解析链中介于角色规则和部门默认规则之间，作为角色的兜底默认值。
func seedRoleDefaults() {
	// 加载角色
	roleMap := map[string]uint{}
	var roles []models.Role
	database.DB.Find(&roles)
	for _, r := range roles {
		roleMap[r.Name] = r.ID
	}

	type ruleSeed struct {
		roleName string
		action   string
		decision string
	}

	seeds := []ruleSeed{
		// 现场操作员：默认 view + download
		{"field_operator", models.PermissionActionView, models.PermissionDecisionAllow},
		{"field_operator", models.PermissionActionDownload, models.PermissionDecisionAllow},
		// 离线编程人员：默认 view + download + upload
		{"offline_programmer", models.PermissionActionView, models.PermissionDecisionAllow},
		{"offline_programmer", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"offline_programmer", models.PermissionActionUpload, models.PermissionDecisionAllow},
		// 产线管理员：默认 view + download + upload + manage
		{"line_admin", models.PermissionActionView, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionUpload, models.PermissionDecisionAllow},
		{"line_admin", models.PermissionActionManage, models.PermissionDecisionAllow},
		// 旧角色兼容
		{"operator", models.PermissionActionView, models.PermissionDecisionAllow},
		{"operator", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"engineer", models.PermissionActionView, models.PermissionDecisionAllow},
		{"engineer", models.PermissionActionDownload, models.PermissionDecisionAllow},
		{"engineer", models.PermissionActionUpload, models.PermissionDecisionAllow},
	}

	rules := make([]models.PermissionRule, 0, len(seeds))
	for _, s := range seeds {
		roleID, ok := roleMap[s.roleName]
		if !ok {
			continue
		}
		rules = append(rules, models.PermissionRule{
			SubjectType:  "role_default",
			SubjectID:    roleID,
			SubjectKey:   "",
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   0,
			Action:       s.action,
			Decision:     s.decision,
		})
	}

	if len(rules) > 0 {
		if err := database.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "subject_type"}, {Name: "subject_id"}, {Name: "subject_key"},
				{Name: "resource_type"}, {Name: "resource_id"}, {Name: "action"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"decision", "updated_at"}),
		}).Create(&rules).Error; err != nil {
			log.Printf("种子角色默认权限失败: %v", err)
		} else {
			log.Println("角色默认产线权限种子完成")
		}
	}
}

// seedDepartmentDefaults 为"制造部"创建部门默认产线权限。
// 确保制造部用户无论什么角色，默认只能查看和下载。
func seedDepartmentDefaults() {
	var dept models.Department
	if err := database.DB.Where("name = ?", "制造部").First(&dept).Error; err != nil {
		// 制造部不存在，跳过
		return
	}

	type ruleSeed struct {
		action   string
		decision string
	}

	seeds := []ruleSeed{
		{models.PermissionActionView, models.PermissionDecisionAllow},
		{models.PermissionActionDownload, models.PermissionDecisionAllow},
	}

	rules := make([]models.PermissionRule, 0, len(seeds))
	for _, s := range seeds {
		rules = append(rules, models.PermissionRule{
			SubjectType:  models.PermissionSubjectDepartmentDefault,
			SubjectID:    dept.ID,
			SubjectKey:   "",
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   0,
			Action:       s.action,
			Decision:     s.decision,
		})
	}

	if len(rules) > 0 {
		if err := database.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "subject_type"}, {Name: "subject_id"}, {Name: "subject_key"},
				{Name: "resource_type"}, {Name: "resource_id"}, {Name: "action"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"decision", "updated_at"}),
		}).Create(&rules).Error; err != nil {
			log.Printf("种子制造部默认权限失败: %v", err)
		} else {
			log.Println("制造部默认产线权限种子完成")
		}
	}
}
