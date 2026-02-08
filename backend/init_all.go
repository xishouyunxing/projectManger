package main

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/models"
	"golang.org/x/crypto/bcrypt"
	"log"
)

func InitAll() {
	log.Println("🚀 开始初始化系统数据...")

	// 加载配置
	config.LoadConfig()

	// 连接数据库
	if err := database.Connect(); err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 自动迁移
	if err := database.AutoMigrate(); err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 1. 创建管理员账号
	createAdmin()

	// 2. 创建工序数据
	createProcesses()

	// 3. 创建车型数据
	createVehicleModels()

	// 4. 创建生产线数据
	createProductionLines()

	log.Println("🎉 系统数据初始化完成！")
	log.Println("🔑 默认登录信息:")
	log.Println("   工号: admin001")
	log.Println("   密码: admin123456")
}

func createAdmin() {
	// 生成管理员密码的哈希值
	password := "admin123456"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		return
	}

	// 创建管理员用户
	admin := models.User{
		EmployeeID: "admin001",
		EmployeeNo: "admin001",
		Name:       "系统管理员",
		Department: "IT部门",
		Role:       "admin",
		Password:   string(hashedPassword),
		Status:     "active",
	}

	// 检查是否已存在
	var existingUser models.User
	result := database.DB.Where("employee_id = ?", admin.EmployeeID).First(&existingUser)
	
	if result.Error == nil {
		log.Printf("✅ 管理员账号已存在")
		return
	}

	// 创建管理员
	if err := database.DB.Create(&admin).Error; err != nil {
		log.Printf("❌ 创建管理员失败: %v", err)
		return
	}

	log.Printf("✅ 创建管理员账号")
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
	log.Printf("✅ 创建工序数据")
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
	log.Printf("✅ 创建车型数据")
}

func createProductionLines() {
	var processes []models.Process
	database.DB.Find(&processes)

	processMap := make(map[string]uint)
	for _, p := range processes {
		processMap[p.Code] = p.ID
	}

	lines := []models.ProductionLine{
		{Name: "吊臂主臂生产线", Code: "UP_ARM_001", Type: "upper", ProcessID: processMap["UP001"], Description: "主要负责起重机吊臂主臂的制造和装配", Status: "active"},
		{Name: "吊臂副臂生产线", Code: "UP_ARM_002", Type: "upper", ProcessID: processMap["UP001"], Description: "主要负责起重机吊臂副臂的制造和装配", Status: "active"},
		{Name: "转台装配生产线", Code: "UP_TURN_001", Type: "upper", ProcessID: processMap["UP002"], Description: "负责起重机转台的整体装配", Status: "active"},
		{Name: "底盘焊接生产线", Code: "LOW_CHASSIS_001", Type: "lower", ProcessID: processMap["LOW001"], Description: "负责起重机底盘的焊接和初步成型", Status: "active"},
		{Name: "支腿液压生产线", Code: "LOW_LEG_001", Type: "lower", ProcessID: processMap["LOW002"], Description: "负责起重机支腿液压系统的制造", Status: "active"},
	}

	for _, line := range lines {
		var existing models.ProductionLine
		if database.DB.Where("code = ?", line.Code).First(&existing).Error == nil {
			continue
		}
		database.DB.Create(&line)
	}
	log.Printf("✅ 创建生产线数据")
}