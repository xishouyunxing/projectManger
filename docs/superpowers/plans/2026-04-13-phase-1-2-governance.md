# Phase 1-2 Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clean up repository boundaries and refactor startup/config handling so the project has explicit runtime directories, a maintainable `.gitignore`, documented structure, and a clearer backend bootstrap path without changing core business behavior.

**Architecture:** The work is split into two tracks that meet in the bootstrap layer. Phase 1 standardizes repository boundaries and documentation at the root level; Phase 2 keeps the current backend packages but introduces focused bootstrap helpers so `main.go` becomes a thin composition entrypoint and config/runtime paths become explicit. Tests are expanded around config loading and startup policy so the refactor has coverage.

**Tech Stack:** Go 1.25, Gin, GORM, MySQL, React 18, Vite, Make, Docker Compose

---

## File Structure

### Files to modify
- `./.gitignore` — reduce duplicate ignore rules and clearly separate source, runtime, and local-workspace artifacts
- `./Makefile` — align clean/build/dev helper commands with the new runtime/build boundary
- `./docker-compose.yml` — mount explicit runtime directories and keep environment keys aligned with the new config structure
- `./README.md` — update quick-start and directory explanations to match the new runtime/config conventions
- `./backend/config/config.go` — replace the flat config layout with grouped config sections and explicit runtime path values
- `./backend/config/config_test.go` — update config tests to match the new structure and mode-specific validation behavior
- `./backend/main.go` — shrink to a composition root that calls bootstrap helpers
- `./backend/init_main.go` — reuse bootstrap/config loading helpers instead of duplicating startup logic
- `./backend/database/database.go` — read database settings from grouped config and split migration policy from connectivity concerns
- `./backend/router/router.go` — read static file paths and environment-specific server behavior from grouped config

### Files to create
- `./docs/project-structure.md` — document which directories are source-controlled versus runtime-generated
- `./.env.example` — provide the canonical configuration template for local development and deployment
- `./backend/app/bootstrap.go` — centralize config load, runtime directory preparation, DB connect, migration policy, and HTTP server startup wiring
- `./backend/app/runtime.go` — runtime-directory helper functions used by both server startup and initialization commands
- `./backend/app/bootstrap_test.go` — focused tests for migration-policy decisions and runtime directory preparation behavior

### Files to verify during implementation
- `./frontend/vite.config.ts` — confirm frontend dev server proxy/docs still match the documented backend port and API base
- `./deploy/**` — verify whether any deployment files depend on paths that must be updated after config/runtime cleanup

---

### Task 1: Inventory current runtime and repo-boundary assumptions

**Files:**
- Modify: `./docs/superpowers/specs/2026-04-13-phase-1-2-governance-design.md` (reference only, no edits expected)
- Test/Verify: `./.gitignore`, `./Makefile`, `./docker-compose.yml`, `./backend/config/config.go`, `./backend/main.go`, `./backend/init_main.go`, `./frontend/vite.config.ts`

- [ ] **Step 1: Capture the current working tree and runtime directories**

Run:
```bash
git status --short && ls && ls deploy
```
Expected:
- `git status --short` shows the existing backend edits and the new `docs/` directory
- root listing includes runtime-oriented directories such as `backups`, `uploads`, `deploy`
- `deploy` contents are visible for path review

- [ ] **Step 2: Inspect current ignore and bootstrap files before making changes**

Run:
```bash
git diff -- .gitignore Makefile docker-compose.yml backend/config/config.go backend/main.go backend/init_main.go backend/database/database.go backend/router/router.go README.md
```
Expected:
- current uncommitted backend changes are visible
- you understand whether local edits must be preserved while applying the plan

- [ ] **Step 3: Record the path and runtime assumptions in a scratch note or commit message draft**

Use this checklist while reviewing the files:
```text
[ ] Which directories are runtime-only?
[ ] Which files assume ../frontend/dist?
[ ] Which commands assume cd backend?
[ ] Which compose volumes mount uploads?
[ ] Which startup paths trigger AutoMigrate?
```
Expected:
- you can answer all five questions before editing code

- [ ] **Step 4: Commit the inventory checkpoint**

```bash
git add docs/superpowers/specs/2026-04-13-phase-1-2-governance-design.md
git commit -m "docs: add phase 1-2 governance spec"
```
Expected:
- commit succeeds and leaves the implementation work isolated from the spec commit

### Task 2: Rewrite repository boundary rules and document directory ownership

**Files:**
- Modify: `./.gitignore:1-194`, `./README.md:141-153`, `./Makefile:33-44`
- Create: `./docs/project-structure.md`
- Test/Verify: `./frontend/.gitignore:1-24`, `./backend/.gitignore:1-33`

- [ ] **Step 1: Write the failing repository-boundary verification by checking ignored paths**

Run:
```bash
git check-ignore -v frontend/node_modules backend/crane-system.exe uploads/example.txt backups/example.zip .planning/STATE.md
```
Expected before changes:
- output is inconsistent or overly dependent on duplicated root rules
- at least one path requires cleanup or clearer ownership

- [ ] **Step 2: Replace the root `.gitignore` with a grouped, deduplicated version**

Update `./.gitignore` to this content:
```gitignore
# Dependencies
node_modules/
frontend/node_modules/

# Frontend build output
frontend/dist/
frontend/build/
dist/
build/
.vite/
coverage/
*.lcov
.nyc_output/
*.tsbuildinfo

# Backend binaries and test artifacts
backend/*.exe
backend/*.exe~
backend/*.dll
backend/*.so
backend/*.dylib
backend/main
backend/*.test
backend/*.out
backend/vendor/
go.work.sum

# Environment and local config
.env
.env.*
!.env.example
*.local
config.local.js
config.local.json
config/*.local.js
config/*.local.json

# Logs and runtime files
logs/
*.log
pids/
*.pid
*.pid.lock
*.seed
*.tmp
*.temp
*.bak
*.backup
.cache/
.parcel-cache/
tmp/
temp/

# Runtime data
uploads/*
!uploads/.gitkeep
backups/*
!backups/.gitkeep
backend/backups/*.zip
backups/*.zip

# Local tooling workspaces
.vscode/
.idea/
.claude/
.worktrees/
.planning/
.agents/

# OS files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Deployment artifacts
deploy.zip
```
Expected:
- duplicate ignore sections are removed
- runtime paths are explicit and easy to scan

- [ ] **Step 3: Add a focused directory-ownership document**

Create `./docs/project-structure.md` with this content:
```markdown
# Project Structure

## Source-controlled directories
- `backend/` — Go backend source code and tests
- `frontend/` — React frontend source code and tests
- `docs/` — design, planning, and maintenance documentation
- `deploy/` — deployment configuration tracked in git when it is source, not packaged output

## Runtime directories
- `uploads/` — uploaded files generated at runtime
- `backups/` — backup archives generated at runtime
- `logs/` — runtime logs

These directories are environment data, not source code. They should be created by the runtime environment or deployment scripts and should not be committed.

## Local-only workspace directories
- `.planning/`
- `.worktrees/`
- `.agents/`
- `.claude/`

These directories store local planning, agent, and tooling state.

## Build output
- `frontend/dist/`
- `backend/*.exe`
- `backend/main`

Build output can be deleted and regenerated.
```
Expected:
- the repo has a single document explaining boundary ownership for future contributors

- [ ] **Step 4: Update README structure notes to point at the new boundary rules**

Replace the project-structure block in `./README.md` with:
```markdown
## 📁 项目结构

```text
projectManger/
├── backend/        # Gin + GORM 后端源码
├── frontend/       # React + Vite 前端源码
├── docs/           # 设计文档、维护文档
├── deploy/         # 部署配置（源码级）
├── backups/        # 运行时备份目录（不纳入版本控制）
├── uploads/        # 运行时上传目录（不纳入版本控制）
├── Makefile        # 常用开发命令
├── docker-compose.yml
└── README.md
```

更详细的目录职责请见 `docs/project-structure.md`。
```
Expected:
- README and the dedicated structure doc no longer disagree

- [ ] **Step 5: Align `make clean` with the new repository boundary**

Update the `clean` target in `./Makefile` to:
```make
clean:
	@echo "清理构建文件..."
	rm -f backend/crane-system
	rm -f backend/*.exe
	rm -f backend/*.test
	rm -f backend/*.out
	rm -rf frontend/dist
	@echo "清理完成!"
```
Expected:
- `make clean` removes build artifacts but does not delete dependency directories or runtime data

- [ ] **Step 6: Re-run ignore verification to prove the boundary rules now work**

Run:
```bash
git check-ignore -v frontend/node_modules backend/crane-system.exe uploads/example.txt backups/example.zip .planning/STATE.md && git diff -- .gitignore README.md Makefile docs/project-structure.md
```
Expected:
- all sample runtime/build/local-tooling paths are ignored by deliberate rules
- diff only shows the intended repository-boundary edits

- [ ] **Step 7: Commit the repository-boundary cleanup**

```bash
git add .gitignore README.md Makefile docs/project-structure.md
git commit -m "chore: clarify repository runtime boundaries"
```
Expected:
- commit succeeds with only Phase 1 boundary files staged

### Task 3: Introduce explicit grouped config with runtime paths

**Files:**
- Modify: `./backend/config/config.go:11-116`, `./backend/config/config_test.go:1-44`, `./docker-compose.yml:22-44`
- Create: `./.env.example`
- Test: `./backend/config/config_test.go`

- [ ] **Step 1: Write the failing config tests for grouped config and path fields**

Replace `./backend/config/config_test.go` with:
```go
package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfigDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("DEFAULT_PASSWORD", "12345678")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("FRONTEND_DIST", "")
	t.Setenv("UPLOAD_ROOT", "")
	t.Setenv("BACKUP_ROOT", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if AppConfig.App.Env != "development" {
		t.Fatalf("expected env development, got %s", AppConfig.App.Env)
	}

	if AppConfig.Database.Port != "3307" {
		t.Fatalf("expected default DB port 3307, got %s", AppConfig.Database.Port)
	}

	if filepath.Clean(AppConfig.Paths.FrontendDist) != filepath.Clean("../frontend/dist") {
		t.Fatalf("unexpected frontend dist: %s", AppConfig.Paths.FrontendDist)
	}

	if filepath.Clean(AppConfig.Paths.UploadRoot) != filepath.Clean("../uploads") {
		t.Fatalf("unexpected upload root: %s", AppConfig.Paths.UploadRoot)
	}

	if filepath.Clean(AppConfig.Paths.BackupRoot) != filepath.Clean("../backups") {
		t.Fatalf("unexpected backup root: %s", AppConfig.Paths.BackupRoot)
	}

	if len(AppConfig.CORS.AllowedOrigins) == 0 {
		t.Fatal("expected default CORS origins")
	}
}

func TestLoadConfigProductionRequiresExplicitSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DEFAULT_PASSWORD", "")
	t.Setenv("DB_PASSWORD", "")

	if err := LoadConfig(); err == nil {
		t.Fatal("expected LoadConfig to fail in production without explicit secrets")
	}
}
```
Expected:
- test file compiles against the planned config shape only after the config refactor is implemented

- [ ] **Step 2: Run the config tests to confirm they fail first**

Run:
```bash
cd backend && go test ./config -run TestLoadConfig -v
```
Expected:
- FAIL because `AppConfig.App`, `AppConfig.Database`, and `AppConfig.Paths` do not exist yet

- [ ] **Step 3: Refactor `backend/config/config.go` to grouped config with explicit runtime paths**

Replace `./backend/config/config.go` with:
```go
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type AppSettings struct {
	Env        string
	ServerPort string
}

type DatabaseSettings struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type AuthSettings struct {
	JWTSecret       string
	DefaultPassword string
}

type PathSettings struct {
	FrontendDist string
	UploadRoot   string
	BackupRoot   string
}

type CORSSettings struct {
	AllowedOrigins []string
}

type Config struct {
	App           AppSettings
	Database      DatabaseSettings
	Auth          AuthSettings
	Paths         PathSettings
	CORS          CORSSettings
	MaxUploadSize int64
}

var AppConfig *Config

func LoadConfig() error {
	envPaths := []string{"../.env", ".env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("成功加载.env文件: %s", path)
			break
		}
	}

	appEnv := getEnv("APP_ENV", "development")
	AppConfig = &Config{
		App: AppSettings{
			Env:        appEnv,
			ServerPort: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseSettings{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3307"),
			User:     getEnv("DB_USER", "crane_user"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "crane_system"),
		},
		Auth: AuthSettings{
			JWTSecret:       os.Getenv("JWT_SECRET"),
			DefaultPassword: os.Getenv("DEFAULT_PASSWORD"),
		},
		Paths: PathSettings{
			FrontendDist: getEnv("FRONTEND_DIST", "../frontend/dist"),
			UploadRoot:   getEnv("UPLOAD_ROOT", "../uploads"),
			BackupRoot:   getEnv("BACKUP_ROOT", "../backups"),
		},
		CORS: CORSSettings{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
		},
		MaxUploadSize: 100 * 1024 * 1024,
	}

	return validateConfig(AppConfig)
}

func validateConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET 未配置")
	}
	if len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET 长度不能少于 32 个字符")
	}
	if strings.TrimSpace(cfg.Auth.DefaultPassword) == "" {
		return fmt.Errorf("DEFAULT_PASSWORD 未配置")
	}
	if len(cfg.Auth.DefaultPassword) < 8 {
		return fmt.Errorf("DEFAULT_PASSWORD 长度不能少于 8 个字符")
	}
	if cfg.App.Env == "production" && strings.TrimSpace(cfg.Database.Password) == "" {
		return fmt.Errorf("DB_PASSWORD 未配置")
	}
	if len(cfg.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS 未配置")
	}
	for _, origin := range cfg.CORS.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS 不允许使用通配符 *")
		}
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
```
Expected:
- config fields are grouped by domain
- upload/backup/frontend paths are explicit
- production behavior is stricter than development

- [ ] **Step 4: Add the canonical env template**

Create `./.env.example` with:
```env
APP_ENV=development
DB_HOST=127.0.0.1
DB_PORT=3307
DB_USER=crane_user
DB_PASSWORD=change-me
DB_NAME=crane_system
JWT_SECRET=replace-with-a-random-secret-at-least-32-characters
DEFAULT_PASSWORD=change-me-too
SERVER_PORT=8080
FRONTEND_DIST=../frontend/dist
UPLOAD_ROOT=../uploads
BACKUP_ROOT=../backups
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```
Expected:
- local developers and deploy tooling have a single authoritative config template

- [ ] **Step 5: Update Docker Compose to use the new runtime-path configuration**

Modify the backend service block in `./docker-compose.yml` to:
```yaml
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: crane-backend
    ports:
      - "8080:8080"
    environment:
      APP_ENV: production
      DB_HOST: mysql
      DB_PORT: 3306
      DB_USER: root
      DB_PASSWORD: crane123456
      DB_NAME: crane_system
      JWT_SECRET: crane-secret-key-change-in-production-123456
      DEFAULT_PASSWORD: admin123456
      SERVER_PORT: 8080
      FRONTEND_DIST: /app/frontend-dist
      UPLOAD_ROOT: /app/uploads
      BACKUP_ROOT: /app/backups
      CORS_ALLOWED_ORIGINS: http://localhost
    depends_on:
      mysql:
        condition: service_healthy
    networks:
      - crane-network
    volumes:
      - ./uploads:/app/uploads
      - ./backups:/app/backups
```
Expected:
- compose reflects the new config keys and runtime-directory separation

- [ ] **Step 6: Run config tests again and confirm they pass**

Run:
```bash
cd backend && go test ./config -run TestLoadConfig -v
```
Expected:
- PASS for both config tests

- [ ] **Step 7: Commit the grouped config work**

```bash
git add .env.example docker-compose.yml backend/config/config.go backend/config/config_test.go
git commit -m "refactor: group runtime configuration"
```
Expected:
- commit contains only config/template/compose changes

### Task 4: Add bootstrap helpers and thin `main.go`

**Files:**
- Create: `./backend/app/bootstrap.go`, `./backend/app/runtime.go`, `./backend/app/bootstrap_test.go`
- Modify: `./backend/main.go:1-34`, `./backend/init_main.go:1-160`, `./backend/database/database.go:14-107`, `./backend/router/router.go:11-178`
- Test: `./backend/app/bootstrap_test.go`

- [ ] **Step 1: Write failing tests for runtime directory preparation and migration policy**

Create `./backend/app/bootstrap_test.go` with:
```go
package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRuntimeDirsCreatesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	uploads := filepath.Join(root, "uploads")
	backups := filepath.Join(root, "backups")

	if err := EnsureRuntimeDirs([]string{uploads, backups}); err != nil {
		t.Fatalf("EnsureRuntimeDirs returned error: %v", err)
	}

	for _, dir := range []string{uploads, backups} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}
}

func TestShouldAutoMigrateByEnv(t *testing.T) {
	cases := []struct {
		env      string
		expected bool
	}{
		{env: "development", expected: true},
		{env: "test", expected: true},
		{env: "production", expected: false},
	}

	for _, tc := range cases {
		if got := ShouldAutoMigrate(tc.env); got != tc.expected {
			t.Fatalf("env %s: expected %v, got %v", tc.env, tc.expected, got)
		}
	}
}
```
Expected:
- tests fail until the helper package exists

- [ ] **Step 2: Run the new bootstrap tests to prove they fail first**

Run:
```bash
cd backend && go test ./app -run Test -v
```
Expected:
- FAIL because `backend/app` does not exist yet

- [ ] **Step 3: Create runtime helper functions**

Create `./backend/app/runtime.go` with:
```go
package app

import "os"

func EnsureRuntimeDirs(paths []string) error {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ShouldAutoMigrate(env string) bool {
	switch env {
	case "development", "test":
		return true
	default:
		return false
	}
}
```
Expected:
- runtime directories and migration policy have a single focused implementation point

- [ ] **Step 4: Create the bootstrap composition layer**

Create `./backend/app/bootstrap.go` with:
```go
package app

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/router"
)

func LoadAndConnect() error {
	if err := config.LoadConfig(); err != nil {
		return err
	}

	if err := EnsureRuntimeDirs([]string{
		config.AppConfig.Paths.UploadRoot,
		config.AppConfig.Paths.BackupRoot,
	}); err != nil {
		return err
	}

	if err := database.Connect(); err != nil {
		return err
	}

	if ShouldAutoMigrate(config.AppConfig.App.Env) {
		if err := database.AutoMigrate(); err != nil {
			return err
		}
	}

	return nil
}

func BuildRouter() *router.RouterBuilder {
	return router.NewRouterBuilder()
}
```
Expected:
- startup logic is centralized instead of duplicated across `main.go` and `init_main.go`

- [ ] **Step 5: Update database access to use grouped config fields**

Change the DSN in `./backend/database/database.go` to:
```go
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Host,
		config.AppConfig.Database.Port,
		config.AppConfig.Database.Name,
	)
```
Expected:
- database package no longer depends on the removed flat fields

- [ ] **Step 6: Refactor router construction to a builder with config-backed paths**

Replace the top of `./backend/router/router.go` with:
```go
package router

import (
	"crane-system/config"
	"crane-system/controllers"
	"crane-system/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type RouterBuilder struct{}

func NewRouterBuilder() *RouterBuilder {
	return &RouterBuilder{}
}

func (b *RouterBuilder) Build() *gin.Engine {
	if config.AppConfig.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.Static("/uploads", config.AppConfig.Paths.UploadRoot)
```
Expected:
- router configuration is now environment-aware and uses explicit runtime paths

- [ ] **Step 7: Shrink `backend/main.go` to a thin composition root**

Replace `./backend/main.go` with:
```go
package main

import (
	"crane-system/app"
	"crane-system/config"
	"log"
)

func main() {
	if err := app.LoadAndConnect(); err != nil {
		log.Fatal("应用启动失败:", err)
	}

	r := app.BuildRouter().Build()
	log.Printf("服务器启动在端口 %s", config.AppConfig.App.ServerPort)
	if err := r.Run(":" + config.AppConfig.App.ServerPort); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
```
Expected:
- `main.go` becomes orchestration-only

- [ ] **Step 8: Reuse bootstrap logic from `init_main.go`**

Replace the startup portion of `./backend/init_main.go` with:
```go
import (
	"crane-system/app"
	"crane-system/database"
	"crane-system/models"
	"golang.org/x/crypto/bcrypt"
	"log"
)

func InitAll() {
	log.Println("🚀 开始初始化系统数据...")

	if err := app.LoadAndConnect(); err != nil {
		log.Fatal("初始化启动失败:", err)
	}

	createDepartments()
	createAdmin()
```
Expected:
- init command stops duplicating config/database/migration setup

- [ ] **Step 9: Run the new bootstrap tests and key package tests**

Run:
```bash
cd backend && go test ./app ./config ./database -v
```
Expected:
- PASS for the new `app` tests and updated config/database tests

- [ ] **Step 10: Run a backend smoke build**

Run:
```bash
cd backend && go test ./... && go build ./...
```
Expected:
- all backend packages compile after the bootstrap refactor

- [ ] **Step 11: Commit the bootstrap refactor**

```bash
git add backend/app/bootstrap.go backend/app/runtime.go backend/app/bootstrap_test.go backend/main.go backend/init_main.go backend/database/database.go backend/router/router.go
git commit -m "refactor: add backend bootstrap layer"
```
Expected:
- commit contains only startup/bootstrap-related changes

### Task 5: Update developer documentation and validate the new startup flow

**Files:**
- Modify: `./README.md:55-139`, `./Makefile:21-31`, `./docker-compose.yml:22-55`
- Test/Verify: `./backend/main.go`, `./backend/init_main.go`, `./frontend/vite.config.ts`

- [ ] **Step 1: Write the failing docs check by comparing README commands to the new config keys**

Run:
```bash
git diff -- README.md Makefile docker-compose.yml .env.example && grep -n "UPLOAD_ROOT\|BACKUP_ROOT\|APP_ENV" .env.example README.md docker-compose.yml
```
Expected:
- README likely does not yet mention the new runtime-path variables or startup behavior

- [ ] **Step 2: Update the README environment and startup section**

Edit `./README.md` so the `.env` example and startup text include:
```markdown
# 服务配置
APP_ENV=development
SERVER_PORT=8080
FRONTEND_DIST=../frontend/dist
UPLOAD_ROOT=../uploads
BACKUP_ROOT=../backups
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
```
And add this note below the config block:
```markdown
说明：
- `UPLOAD_ROOT` 和 `BACKUP_ROOT` 指向运行时目录，默认位于项目根目录外层的运行数据位置
- `APP_ENV=development` 时允许自动迁移，生产环境默认不隐式自动迁移
- 后端启动前会自动确保运行时目录存在
```
Expected:
- docs match the new config/runtime behavior

- [ ] **Step 3: Update Makefile dev guidance to reflect the canonical env template**

Change the `dev` target in `./Makefile` to:
```make
dev:
	@echo "启动开发环境..."
	@echo "请先复制 .env.example 为 .env 并按本地环境修改配置"
	@echo "请在不同终端窗口运行:"
	@echo "  终端1: cd backend && go run main.go"
	@echo "  终端2: cd frontend && npm run dev"
```
Expected:
- developers are pointed at the new config template before starting services

- [ ] **Step 4: Run config and smoke checks from the documented flow**

Run:
```bash
cp .env.example .env && cd backend && go test ./config ./app -v && go run main.go
```
Expected:
- tests pass
- server starts or fails only on known external dependencies such as the database not running

- [ ] **Step 5: Validate frontend dev proxy assumptions were not broken**

Run:
```bash
cd frontend && npm run build
```
Expected:
- PASS, confirming the frontend still builds against the documented API path assumptions

- [ ] **Step 6: Review the full diff before final handoff**

Run:
```bash
git diff -- .gitignore .env.example README.md Makefile docker-compose.yml backend/config/config.go backend/config/config_test.go backend/app/bootstrap.go backend/app/runtime.go backend/app/bootstrap_test.go backend/main.go backend/init_main.go backend/database/database.go backend/router/router.go docs/project-structure.md
```
Expected:
- all edits are limited to Phase 1-2 governance files
- no business-controller changes slipped in

- [ ] **Step 7: Commit the docs and validation updates**

```bash
git add README.md Makefile docker-compose.yml .env.example docs/project-structure.md
git commit -m "docs: align startup and runtime guidance"
```
Expected:
- final commit closes the loop between implementation and developer-facing docs

---

## Self-Review Checklist

### Spec coverage
- Repository/runtime boundary cleanup is covered in Task 2.
- Grouped configuration, `.env.example`, and runtime paths are covered in Task 3.
- Thin bootstrap flow and startup policy are covered in Task 4.
- Validation and updated docs are covered in Task 5.

### Placeholder scan
- No `TODO`, `TBD`, or “implement later” placeholders remain.
- Every code-changing step includes concrete code or exact replacement content.
- Every verification step includes an exact command and expected result.

### Type consistency
- Config access uses `AppConfig.App`, `AppConfig.Database`, `AppConfig.Auth`, `AppConfig.Paths`, and `AppConfig.CORS` consistently.
- Bootstrap helpers are named `EnsureRuntimeDirs`, `ShouldAutoMigrate`, `LoadAndConnect`, and `BuildRouter` consistently across tests and implementation.
- Router construction consistently uses `NewRouterBuilder().Build()`.
