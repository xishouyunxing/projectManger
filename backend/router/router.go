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

	// CORS 只允许配置中的前端来源，避免携带凭据的跨域请求被任意站点调用。
	r.Use(cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 公共接口：只放无需登录即可访问的能力。
	public := r.Group("/api")
	{
		public.POST("/login", controllers.Login)
	}

	// 受保护接口：先经过 JWT 鉴权，再按路由细分管理员权限或产线权限。
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		users := protected.Group("/users")
		{
			users.GET("", middleware.AdminMiddleware(), controllers.GetUsers)
			users.GET("/:id", controllers.GetUser)
			users.POST("", middleware.AdminMiddleware(), controllers.CreateUser)
			users.PUT("/:id", controllers.UpdateUser)
			users.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteUser)
			users.PUT("/:id/password", controllers.ChangePassword)
			users.PUT("/:id/reset-password", middleware.AdminMiddleware(), controllers.ResetPassword)
		}

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

		processes := protected.Group("/processes")
		{
			processes.GET("", controllers.GetProcesses)
			processes.GET("/:id", controllers.GetProcess)
			processes.POST("", middleware.AdminMiddleware(), controllers.CreateProcess)
			processes.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdateProcess)
			processes.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteProcess)
		}

		models := protected.Group("/vehicle-models")
		{
			models.GET("", controllers.GetVehicleModels)
			models.GET("/:id", controllers.GetVehicleModel)
			models.POST("", middleware.AdminMiddleware(), controllers.CreateVehicleModel)
			models.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdateVehicleModel)
			models.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteVehicleModel)
		}

		programs := protected.Group("/programs")
		{
			programs.GET("", controllers.GetPrograms)
			programs.GET("/export/excel", controllers.ExportProgramsExcel)
			programs.GET("/:id", controllers.GetProgram)
			programs.POST("", controllers.CreateProgram)
			programs.PUT("/:id", controllers.UpdateProgram)
			programs.PUT("/:id/custom-field-values", controllers.SaveProgramCustomFieldValues)
			programs.DELETE("/:id", controllers.DeleteProgram)
			programs.GET("/by-vehicle/:vehicle_id", controllers.GetProgramsByVehicle)
		}

		files := protected.Group("/files")
		{
			files.POST("/upload", controllers.UploadFile)
			files.GET("/download/:id", controllers.DownloadFile)
			files.GET("/download/program/:program_id/latest", controllers.DownloadProgramLatestVersion)
			files.GET("/download/version/:version", controllers.DownloadVersionFiles)
			files.GET("/program/:program_id", controllers.GetProgramFiles)
			files.DELETE("/:id", controllers.DeleteFile)
		}

		// 用户权限：包含传统明细接口和矩阵接口。
		// 矩阵接口支持“继承/显式覆盖”，用于精确配置用户在每条产线上的最终覆盖规则。
		permissions := protected.Group("/permissions")
		{
			permissions.GET("", middleware.AdminMiddleware(), controllers.GetPermissions)
			permissions.POST("", middleware.AdminMiddleware(), controllers.CreatePermission)
			permissions.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdatePermission)
			permissions.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeletePermission)
			permissions.GET("/user/:user_id/matrix", middleware.AdminMiddleware(), controllers.GetUserPermissionMatrix)
			permissions.PUT("/user/:user_id/matrix", middleware.AdminMiddleware(), controllers.SaveUserPermissionMatrix)
			permissions.GET("/user/:user_id", controllers.GetUserPermissions)
			permissions.GET("/user/:user_id/effective", controllers.GetUserEffectivePermissions)
		}

		// 部门权限：管理员维护部门对产线的显式授权，供部门成员继承。
		deptPermissions := protected.Group("/department-permissions")
		deptPermissions.Use(middleware.AdminMiddleware())
		{
			deptPermissions.GET("", controllers.GetDepartmentPermissions)
			deptPermissions.POST("", controllers.CreateDepartmentPermission)
			deptPermissions.PUT("/:id", controllers.UpdateDepartmentPermission)
			deptPermissions.DELETE("/:id", controllers.DeleteDepartmentPermission)
			deptPermissions.GET("/department/:department_id/matrix", controllers.GetDepartmentPermissionMatrix)
			deptPermissions.PUT("/department/:department_id/matrix", controllers.SaveDepartmentPermissionMatrix)
		}

		// 默认权限：角色默认和部门默认只作为兜底来源，低于用户/部门显式覆盖。
		permissionDefaults := protected.Group("/permission-defaults")
		permissionDefaults.Use(middleware.AdminMiddleware())
		{
			permissionDefaults.GET("/roles/:role/matrix", controllers.GetRoleDefaultPermissionMatrix)
			permissionDefaults.PUT("/roles/:role/matrix", controllers.SaveRoleDefaultPermissionMatrix)
			permissionDefaults.GET("/departments/:department_id/matrix", controllers.GetDepartmentDefaultPermissionMatrix)
			permissionDefaults.PUT("/departments/:department_id/matrix", controllers.SaveDepartmentDefaultPermissionMatrix)
		}

		// 角色管理：系统管理员可管理所有角色，产线管理员可查看。
		roles := protected.Group("/roles")
		{
			roles.GET("", controllers.GetRoles)
			roles.GET("/:id", controllers.GetRole)
			roles.POST("", middleware.AdminMiddleware(), controllers.CreateRole)
			roles.PUT("/:id", middleware.AdminMiddleware(), controllers.UpdateRole)
			roles.DELETE("/:id", middleware.AdminMiddleware(), controllers.DeleteRole)
			roles.GET("/:id/permissions", controllers.GetRoleLinePermissions)
			roles.PUT("/:id/permissions", middleware.AdminMiddleware(), controllers.SaveRoleLinePermissions)
			roles.PUT("/:id/function-permissions", middleware.AdminMiddleware(), controllers.SaveRolePermissions)
		}

		// 功能权限定义（只读）。
		permissionDefs := protected.Group("/permission-definitions")
		{
			permissionDefs.GET("", controllers.GetAllPermissions)
		}

		// 产线管理员分配。
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
			versions.POST("", controllers.CreateVersion)
			versions.PUT("/:id", controllers.UpdateVersion)
			versions.PUT("/:id/activate", controllers.ActivateVersion)
		}

		programs.POST("/batch-upload", controllers.BatchUploadPrograms)
		programs.POST("/batch-import", controllers.BatchImportPrograms)

		tasks := protected.Group("/tasks")
		{
			tasks.GET("/:task_id/status", controllers.GetTaskStatus)
		}

		mappings := protected.Group("/program-mappings")
		{
			mappings.GET("/by-parent/:program_id", controllers.GetProgramMappingsByParent)
			mappings.GET("/by-child/:program_id", controllers.GetProgramMappingByChild)
			mappings.POST("", controllers.CreateProgramMappings)
			mappings.DELETE("/:id", controllers.DeleteProgramMapping)
		}

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

		migration := protected.Group("/migration")
		migration.Use(middleware.AdminMiddleware())
		{
			migration.GET("/status", controllers.GetMigrationStatus)
			migration.POST("/start", controllers.StartMigration)
			migration.POST("/rollback", controllers.RollbackMigration)
		}

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

	registerFrontendRoutes(r)

	return r
}

func registerFrontendRoutes(r *gin.Engine) {
	frontendDist := config.AppConfig.App.FrontendDist
	indexPath := filepath.Join(frontendDist, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return
	}

	// 部署模式下由后端托管前端 dist：
	// API 和 uploads 仍返回 404，其他 GET 路径回退到 index.html 支持 SPA 路由刷新。
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
