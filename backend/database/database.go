package database

import (
	"crane-system/config"
	"crane-system/models"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
		Logger: logger.Default.LogMode(logger.Info),
	})

	return err
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
	}
}

func ensureTables() error {
	for _, model := range migrationModels() {
		if DB.Migrator().HasTable(model) {
			continue
		}

		if err := DB.Migrator().CreateTable(model); err != nil {
			return err
		}
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

	return nil
}

func AutoMigrate() error {
	if err := ensureTables(); err != nil {
		return err
	}

	return ValidateSchema()
}
