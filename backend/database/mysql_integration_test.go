package database

import (
	"crane-system/config"
	"crane-system/models"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openMySQLIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open mysql integration db: %v", err)
	}

	return db
}

func resetMySQLIntegrationSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	models := migrationModels()
	for i := len(models) - 1; i >= 0; i-- {
		_ = db.Migrator().DropTable(models[i])
	}
}

func TestMySQLAutoMigrateFreshAndRepeat(t *testing.T) {
	DB = openMySQLIntegrationDB(t)
	resetMySQLIntegrationSchema(t, DB)
	t.Cleanup(func() { resetMySQLIntegrationSchema(t, DB) })

	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := AutoMigrate(); err != nil {
		t.Fatalf("first auto migrate: %v", err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}

	var admin models.User
	if err := DB.Where("employee_id = ?", "admin001").First(&admin).Error; err != nil {
		t.Fatalf("find seeded admin: %v", err)
	}

	var migrationCount int64
	if err := DB.Model(&SchemaMigration{}).Where("name IN ?", []string{migrationStepSchemaBootstrap, migrationStepBaseSeed, migrationStepPermissionBackfill}).Count(&migrationCount).Error; err != nil {
		t.Fatalf("count migration records: %v", err)
	}
	if migrationCount != 3 {
		t.Fatalf("expected three migration records, got %d", migrationCount)
	}
}

func TestMySQLAutoMigratePartialVehicleModelTable(t *testing.T) {
	DB = openMySQLIntegrationDB(t)
	resetMySQLIntegrationSchema(t, DB)
	t.Cleanup(func() { resetMySQLIntegrationSchema(t, DB) })

	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := DB.Exec("CREATE TABLE vehicle_models (id bigint unsigned AUTO_INCREMENT PRIMARY KEY, code varchar(50))").Error; err != nil {
		t.Fatalf("create partial vehicle_models table: %v", err)
	}
	if err := DB.Exec("CREATE UNIQUE INDEX uni_vehicle_models_code ON vehicle_models (code)").Error; err != nil {
		t.Fatalf("create legacy unique index: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate partial vehicle_models: %v", err)
	}
	if !DB.Migrator().HasTable(&models.Program{}) {
		t.Fatal("expected later tables to be created")
	}
}

func TestMySQLMigrationLockAcquireRelease(t *testing.T) {
	DB = openMySQLIntegrationDB(t)

	if err := DB.Connection(func(db *gorm.DB) error {
		if err := acquireMigrationLock(db); err != nil {
			return err
		}
		if err := releaseMigrationLock(db); err != nil {
			return err
		}
		if err := acquireMigrationLock(db); err != nil {
			return err
		}
		return releaseMigrationLock(db)
	}); err != nil {
		t.Fatalf("migration lock acquire/release: %v", err)
	}
}
