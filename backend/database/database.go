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
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBName,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	return err
}

func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.ProductionLine{},
		&models.Program{},
		&models.ProgramFile{},
		&models.ProgramVersion{},
		&models.ProgramRelation{},
		&models.UserPermission{},
		&models.VehicleModel{},
		&models.Process{},
	)
}
