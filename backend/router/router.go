package router

import (
	"crane-system/controllers"
	"crane-system/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 静态文件服务
	r.Static("/uploads", "./uploads")

	// 公开路由
	public := r.Group("/api")
	{
		public.POST("/login", controllers.Login)
		public.POST("/register", controllers.Register)
	}

	// 需要认证的路由
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// 用户管理
		users := protected.Group("/users")
		{
			users.GET("", controllers.GetUsers)
			users.GET("/:id", controllers.GetUser)
			users.POST("", middleware.AdminMiddleware(), controllers.CreateUser)
			users.PUT("/:id", controllers.UpdateUser)
			users.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteUser)
			users.PUT("/:id/password", controllers.ChangePassword)
			users.PUT("/:id/reset-password", middleware.AdminMiddleware(), controllers.ResetPassword)
		}

		// 生产线管理
		lines := protected.Group("/production-lines")
		{
			lines.GET("", controllers.GetProductionLines)
			lines.GET("/:id", controllers.GetProductionLine)
			lines.GET("/:id/custom-fields", controllers.GetProductionLineCustomFields)
			lines.POST("", middleware.AdminMiddleware(), controllers.CreateProductionLine)
			lines.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdateProductionLine)
			lines.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteProductionLine)
			lines.POST("/:id/custom-fields", middleware.AdminMiddleware(), controllers.CreateProductionLineCustomField)
			lines.PUT("/:id/custom-fields/:fieldId", middleware.AdminMiddleware(), controllers.UpdateProductionLineCustomField)
			lines.DELETE("/:id/custom-fields/:fieldId", middleware.AdminMiddleware(), controllers.DeleteProductionLineCustomField)
		}

		// 工序管理
		processes := protected.Group("/processes")
		{
			processes.GET("", controllers.GetProcesses)
			processes.GET("/:id", controllers.GetProcess)
			processes.POST("", middleware.AdminMiddleware(), controllers.CreateProcess)
			processes.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdateProcess)
			processes.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteProcess)
		}

		// 车型管理
		models := protected.Group("/vehicle-models")
		{
			models.GET("", controllers.GetVehicleModels)
			models.GET("/:id", controllers.GetVehicleModel)
			models.POST("", middleware.AdminMiddleware(), controllers.CreateVehicleModel)
			models.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdateVehicleModel)
			models.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteVehicleModel)
		}

		// 程序管理
		programs := protected.Group("/programs")
		{
			programs.GET("", controllers.GetPrograms)
			programs.GET("/:id", controllers.GetProgram)
			programs.POST("", controllers.CreateProgram)
			programs.PUT("/:id", controllers.UpdateProgram)
			programs.PUT("/:id/custom-field-values", controllers.SaveProgramCustomFieldValues)
			programs.DELETE("/:id", controllers.DeleteProgram)
			programs.GET("/by-vehicle/:vehicle_id", controllers.GetProgramsByVehicle)
		}

		// 文件管理
		files := protected.Group("/files")
		{
			files.POST("/upload", controllers.UploadFile)
			files.GET("/download/:id", controllers.DownloadFile)
			files.GET("/download/program/:program_id/latest", controllers.DownloadProgramLatestVersion)
			files.GET("/download/version/:version", controllers.DownloadVersionFiles)
			files.GET("/program/:program_id", controllers.GetProgramFiles)
			files.DELETE("/:id", controllers.DeleteFile)
		}

		// 权限管理
		permissions := protected.Group("/permissions")
		{
			permissions.GET("", middleware.AdminMiddleware(), controllers.GetPermissions)
			permissions.POST("", middleware.AdminMiddleware(), controllers.CreatePermission)
			permissions.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdatePermission)
			permissions.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeletePermission)
			permissions.GET("/user/:user_id", controllers.GetUserPermissions)
			permissions.GET("/user/:user_id/effective", controllers.GetUserEffectivePermissions)
		}

		// 部门权限管理 - 仅管理员可用
		deptPermissions := protected.Group("/department-permissions")
		deptPermissions.Use(middleware.AdminMiddleware())
		{
			deptPermissions.GET("", controllers.GetDepartmentPermissions)
			deptPermissions.POST("", controllers.CreateDepartmentPermission)
			deptPermissions.PUT("/:id", controllers.UpdateDepartmentPermission)
			deptPermissions.DELETE("/:id", controllers.DeleteDepartmentPermission)
		}

		// 程序版本
		versions := protected.Group("/versions")
		{
			versions.GET("/program/:program_id", controllers.GetProgramVersions)
			versions.POST("", controllers.CreateVersion)
			versions.PUT("/:id/activate", controllers.ActivateVersion)
		}

		// 程序关联
		relations := protected.Group("/relations")
		{
			relations.GET("/program/:program_id", controllers.GetProgramRelations)
			relations.POST("", controllers.CreateRelation)
			relations.DELETE("/:id", controllers.DeleteRelation)
		}

		// 备份恢复管理 - 仅管理员可用
		backup := protected.Group("/backup")
		backup.Use(middleware.AdminMiddleware())
		{
			backup.POST("/database", controllers.CreateDatabaseBackup)
			backup.POST("/files", controllers.CreateFilesBackup)
			backup.POST("/full", controllers.CreateFullBackup)
			backup.GET("", controllers.GetBackupList)
			backup.DELETE("/:name", controllers.DeleteBackup)
			backup.GET("/download/:name", controllers.DownloadBackup)
			backup.POST("/restore/database/:name", controllers.RestoreDatabase)
			backup.POST("/restore/files/:name", controllers.RestoreFiles)
		}

		// 文件迁移管理 - 仅管理员可用
		migration := protected.Group("/migration")
		migration.Use(middleware.AdminMiddleware())
		{
			migration.GET("/status", controllers.GetMigrationStatus)
			migration.POST("/start", controllers.StartMigration)
			migration.POST("/rollback", controllers.RollbackMigration)
		}

		// 部门管理 - 仅管理员可用
		departments := protected.Group("/departments")
		departments.Use(middleware.AdminMiddleware())
		{
			departments.GET("", controllers.GetDepartments)
			departments.GET("/:id", controllers.GetDepartment)
			departments.POST("", controllers.CreateDepartment)
			departments.PUT("/:id", controllers.UpdateDepartment)
			departments.DELETE("/:id", controllers.DeleteDepartment)
		}
	}

	return r
}
