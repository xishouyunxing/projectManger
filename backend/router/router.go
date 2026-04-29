package router

import (
	"crane-system/config"
	"crane-system/controllers"
	"crane-system/middleware"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	public := r.Group("/api")
	{
		public.POST("/login", controllers.Login)
	}

	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		users := protected.Group("/users")
		{
			users.GET("", middleware.RequireAnyPermission("page:user_management", "page:permissions"), controllers.GetUsers)
			users.GET("/:id", controllers.GetUser)
			users.POST("", middleware.RequirePermission("op:user_create"), controllers.CreateUser)
			users.PUT("/:id", middleware.RequirePermission("op:user_edit"), controllers.UpdateUser)
			users.DELETE("/:id", middleware.RequirePermission("op:user_delete"), controllers.DeleteUser)
			users.PUT("/:id/password", controllers.ChangePassword)
			users.PUT("/:id/reset-password", middleware.RequirePermission("op:password_reset"), controllers.ResetPassword)
		}

		lines := protected.Group("/production-lines")
		{
			lines.GET("", controllers.GetProductionLines)
			lines.GET("/:id", controllers.GetProductionLine)
			lines.GET("/:id/custom-fields", controllers.GetProductionLineCustomFields)
			lines.POST("", middleware.RequirePermission("page:production_lines"), controllers.CreateProductionLine)
			lines.PUT("/:id", middleware.RequirePermission("page:production_lines"), controllers.UpdateProductionLine)
			lines.DELETE("/:id", middleware.RequirePermission("page:production_lines"), controllers.DeleteProductionLine)
			lines.POST("/:id/custom-fields", middleware.RequirePermission("page:production_lines"), controllers.CreateProductionLineCustomField)
			lines.PUT("/:id/custom-fields/:fieldId", middleware.RequirePermission("page:production_lines"), controllers.UpdateProductionLineCustomField)
			lines.DELETE("/:id/custom-fields/:fieldId", middleware.RequirePermission("page:production_lines"), controllers.DeleteProductionLineCustomField)
		}

		processes := protected.Group("/processes")
		{
			processes.GET("", controllers.GetProcesses)
			processes.GET("/:id", controllers.GetProcess)
			processes.POST("", middleware.RequirePermission("page:production_lines"), controllers.CreateProcess)
			processes.PUT("/:id", middleware.RequirePermission("page:production_lines"), controllers.UpdateProcess)
			processes.DELETE("/:id", middleware.RequirePermission("page:production_lines"), controllers.DeleteProcess)
		}

		models := protected.Group("/vehicle-models")
		{
			models.GET("", controllers.GetVehicleModels)
			models.GET("/:id", controllers.GetVehicleModel)
			models.POST("", middleware.RequirePermission("page:vehicle_models"), controllers.CreateVehicleModel)
			models.PUT("/:id", middleware.RequirePermission("page:vehicle_models"), controllers.UpdateVehicleModel)
			models.DELETE("/:id", middleware.RequirePermission("page:vehicle_models"), controllers.DeleteVehicleModel)
		}

		programs := protected.Group("/programs")
		{
			programs.GET("", controllers.GetPrograms)
			programs.GET("/export/excel", middleware.RequirePermission("op:program_export"), controllers.ExportProgramsExcel)
			programs.GET("/:id", controllers.GetProgram)
			programs.POST("", middleware.RequirePermission("op:program_create"), controllers.CreateProgram)
			programs.PUT("/:id", middleware.RequirePermission("op:program_edit"), controllers.UpdateProgram)
			programs.PUT("/:id/custom-field-values", controllers.SaveProgramCustomFieldValues)
			programs.DELETE("/:id", middleware.RequirePermission("op:program_delete"), controllers.DeleteProgram)
			programs.GET("/by-vehicle/:vehicle_id", controllers.GetProgramsByVehicle)
		}

		files := protected.Group("/files")
		{
			files.POST("/upload", middleware.RequirePermission("op:file_upload"), controllers.UploadFile)
			files.GET("/download/:id", middleware.RequirePermission("op:file_download"), controllers.DownloadFile)
			files.GET("/download/program/:program_id/latest", middleware.RequirePermission("op:file_download"), controllers.DownloadProgramLatestVersion)
			files.GET("/download/version/:version", middleware.RequirePermission("op:file_download"), controllers.DownloadVersionFiles)
			files.GET("/program/:program_id", controllers.GetProgramFiles)
			files.DELETE("/:id", middleware.RequirePermission("op:file_delete"), controllers.DeleteFile)
		}

		permissions := protected.Group("/permissions")
		{
			permissions.GET("", middleware.RequirePermission("page:permissions"), controllers.GetPermissions)
			permissions.POST("", middleware.RequirePermission("page:permissions"), controllers.CreatePermission)
			permissions.PUT("/:id", middleware.RequirePermission("page:permissions"), controllers.UpdatePermission)
			permissions.DELETE("/:id", middleware.RequirePermission("page:permissions"), controllers.DeletePermission)
			permissions.GET("/user/:user_id/matrix", middleware.RequirePermission("page:permissions"), controllers.GetUserPermissionMatrix)
			permissions.PUT("/user/:user_id/matrix", middleware.RequirePermission("page:permissions"), controllers.SaveUserPermissionMatrix)
			permissions.GET("/user/:user_id", controllers.GetUserPermissions)
			permissions.GET("/user/:user_id/effective", controllers.GetUserEffectivePermissions)
		}

		deptPermissions := protected.Group("/department-permissions")
		deptPermissions.Use(middleware.RequirePermission("page:permissions"))
		{
			deptPermissions.GET("", controllers.GetDepartmentPermissions)
			deptPermissions.POST("", controllers.CreateDepartmentPermission)
			deptPermissions.PUT("/:id", controllers.UpdateDepartmentPermission)
			deptPermissions.DELETE("/:id", controllers.DeleteDepartmentPermission)
			deptPermissions.GET("/department/:department_id/matrix", controllers.GetDepartmentPermissionMatrix)
			deptPermissions.PUT("/department/:department_id/matrix", controllers.SaveDepartmentPermissionMatrix)
		}

		permissionDefaults := protected.Group("/permission-defaults")
		permissionDefaults.Use(middleware.RequirePermission("page:permissions"))
		{
			permissionDefaults.GET("/roles/:role/matrix", controllers.GetRoleDefaultPermissionMatrix)
			permissionDefaults.PUT("/roles/:role/matrix", controllers.SaveRoleDefaultPermissionMatrix)
			permissionDefaults.GET("/departments/:department_id/matrix", controllers.GetDepartmentDefaultPermissionMatrix)
			permissionDefaults.PUT("/departments/:department_id/matrix", controllers.SaveDepartmentDefaultPermissionMatrix)
		}

		roles := protected.Group("/roles")
		{
			roles.GET("", controllers.GetRoles)
			roles.GET("/:id", controllers.GetRole)
			roles.POST("", middleware.RequirePermission("page:permissions"), controllers.CreateRole)
			roles.PUT("/:id", middleware.RequirePermission("page:permissions"), controllers.UpdateRole)
			roles.DELETE("/:id", middleware.RequirePermission("page:permissions"), controllers.DeleteRole)
			roles.GET("/:id/permissions", controllers.GetRoleLinePermissions)
			roles.PUT("/:id/permissions", middleware.RequirePermission("page:permissions"), controllers.SaveRoleLinePermissions)
			roles.PUT("/:id/function-permissions", middleware.RequirePermission("page:permissions"), controllers.SaveRolePermissions)
		}

		permissionDefs := protected.Group("/permission-definitions")
		{
			permissionDefs.GET("", controllers.GetAllPermissions)
		}

		lineAdmin := protected.Group("/line-admin")
		{
			lineAdmin.GET("/assignments", controllers.GetLineAdminAssignments)
			lineAdmin.POST("/assignments", middleware.RequirePermission("op:line_permission_assign"), controllers.CreateLineAdminAssignment)
			lineAdmin.DELETE("/assignments/:id", middleware.RequirePermission("op:line_permission_assign"), controllers.DeleteLineAdminAssignment)
			lineAdmin.GET("/lines/:id/permissions", controllers.GetLinePermissionsByLine)
			lineAdmin.PUT("/lines/:id/permissions", controllers.SaveLinePermissionByAdmin)
		}

		versions := protected.Group("/versions")
		{
			versions.GET("/program/:program_id", controllers.GetProgramVersions)
			versions.POST("", middleware.RequirePermission("op:version_create"), controllers.CreateVersion)
			versions.PUT("/:id", middleware.RequirePermission("op:version_manage"), controllers.UpdateVersion)
			versions.PUT("/:id/activate", middleware.RequirePermission("op:version_manage"), controllers.ActivateVersion)
		}

		programs.POST("/batch-upload", middleware.RequirePermission("op:program_create"), controllers.BatchUploadPrograms)
		programs.POST("/batch-import", middleware.RequirePermission("op:program_create"), controllers.BatchImportPrograms)

		tasks := protected.Group("/tasks")
		{
			tasks.GET("/:task_id/status", controllers.GetTaskStatus)
		}

		mappings := protected.Group("/program-mappings")
		{
			mappings.GET("/by-parent/:program_id", controllers.GetProgramMappingsByParent)
			mappings.GET("/by-child/:program_id", controllers.GetProgramMappingByChild)
			mappings.POST("", middleware.RequirePermission("op:program_create"), controllers.CreateProgramMappings)
			mappings.DELETE("/:id", middleware.RequirePermission("op:program_delete"), controllers.DeleteProgramMapping)
		}

		backup := protected.Group("/backup")
		backup.Use(middleware.RequirePermission("op:backup_restore"))
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

		migration := protected.Group("/migration")
		migration.Use(middleware.RequirePermission("page:system_management"))
		{
			migration.GET("/status", controllers.GetMigrationStatus)
			migration.POST("/start", controllers.StartMigration)
			migration.POST("/rollback", controllers.RollbackMigration)
		}

		departments := protected.Group("/departments")
		{
			departments.GET("", middleware.RequireAnyPermission("page:user_management", "page:permissions", "page:system_management"), controllers.GetDepartments)
			departments.GET("/:id", controllers.GetDepartment)
			departments.POST("", middleware.RequirePermission("page:system_management"), controllers.CreateDepartment)
			departments.PUT("/:id", middleware.RequirePermission("page:system_management"), controllers.UpdateDepartment)
			departments.DELETE("/:id", middleware.RequirePermission("page:system_management"), controllers.DeleteDepartment)
		}
	}

	registerFrontendRoutes(r)

	return r
}

func registerFrontendRoutes(r *gin.Engine) {
	frontendDist := config.AppConfig.App.FrontendDist
	indexPath := filepath.Join(frontendDist, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return
	}

	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Status(http.StatusNotFound)
			return
		}

		requestPath := c.Request.URL.Path
		if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		if requestPath == "/uploads" || strings.HasPrefix(requestPath, "/uploads/") {
			c.Status(http.StatusNotFound)
			return
		}

		frontendPath := filepath.Join(frontendDist, filepath.FromSlash(strings.TrimPrefix(path.Clean(requestPath), "/")))
		if info, err := os.Stat(frontendPath); err == nil && !info.IsDir() {
			c.File(frontendPath)
			return
		}

		c.File(indexPath)
	})
}
