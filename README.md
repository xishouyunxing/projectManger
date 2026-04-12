# 🏗️ 离线编程程序管理系统

适用于离线编程组生产线程序管理系统，提供程序版本管理、用户与部门权限管理、车型与生产线管理、文件备份恢复等功能。系统支持上车/下车结构管理、程序文件版本追踪，以及按生产线粒度的权限控制。

## 🚀 技术栈

### 后端技术栈
- **Go 1.24**
- **Gin**
- **GORM**
- **MySQL 8.0**
- **JWT**
- **Gin-Cors**

### 前端技术栈
- **React 18**
- **TypeScript**
- **Vite 5**
- **Ant Design 5**
- **React Router 6**
- **Axios**
- **Day.js**

## 🎨 UI 框架说明

本项目前端使用 **Ant Design 5** 作为主要 UI 组件库，提供统一的设计语言、响应式布局、中文本地化支持和较完整的企业后台组件能力。

## 🎯 功能特性

### 🔧 程序管理
- 程序基础信息维护
- 程序文件上传、下载与版本追踪
- 程序关联关系管理
- 按车型查看程序
- 支持批量导入、文件迁移与文件完整性治理

### 🏗️ 生产线与车型管理
- 上车 / 下车生产线管理
- 工序与生产线关联
- 车型信息维护
- 生产线自定义字段模板配置

### 👥 用户与权限系统
- 用户、部门、角色管理
- 用户级权限分配
- 部门级权限分配
- 管理员重置密码、用户自行修改密码
- 基于 JWT 的认证机制

### 📁 文件与系统管理
- 程序文件存储与版本记录
- 文件忽略列表与完整性检查
- 数据库 / 文件 / 全量备份与恢复
- 长任务管理（如批量导入）

## 🚀 快速开始

### 1. 数据库准备

```sql
-- 连接 MySQL
mysql -u root -p

-- 创建数据库
CREATE DATABASE zlzk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建专用用户（推荐）
CREATE USER 'zlzk'@'localhost' IDENTIFIED BY 'zlzk.12345678';
GRANT ALL PRIVILEGES ON zlzk.* TO 'zlzk'@'localhost';
FLUSH PRIVILEGES;
```

### 2. 配置环境变量

在项目根目录创建 `.env`：

```env
# 数据库配置
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=zlzk
DB_PASSWORD=zlzk.12345678
DB_NAME=zlzk

# JWT 配置（必须至少 32 个字符）
JWT_SECRET=replace-with-a-random-secret-at-least-32-characters

# 服务配置
SERVER_PORT=8080
DEFAULT_PASSWORD=zlzk.12345678
FRONTEND_DIST=../frontend/dist
```

说明：
- 后端会优先从项目根目录加载 `.env`
- `JWT_SECRET` 未配置或长度不足时，后端会直接启动失败
- 前端开发服务器默认运行在 `http://localhost:3000`
- 前端开发环境会将 `/api` 代理到 `http://localhost:8080`

### 3. 初始化系统数据

```bash
# 在项目根目录执行
go run ./init_all.go
```

初始化会自动创建：
- 部门
- 管理员账号
- 车型基础数据
- 生产线基础数据

默认管理员账号：
- 工号：`admin001`
- 密码：`admin123456`

### 4. 启动后端服务

```bash
cd backend
go mod download
go run main.go
```

后端默认监听：`http://localhost:8080`

### 5. 启动前端服务

```bash
# 方法 1：使用便捷脚本（推荐）
fnpm install
fnpm run dev

# 方法 2：传统方式
cd frontend
npm install
npm run dev
```

前端默认地址：`http://localhost:3000`

## 📁 项目结构

```text
projectManger/
├── backend/        # Gin + GORM 后端服务
├── frontend/       # React + Vite 前端应用
├── backups/        # 备份文件目录
├── uploads/        # 上传文件目录
├── init_all.go     # 根目录初始化脚本
├── Makefile        # 常用开发命令
├── docker-compose.yml
└── README.md
```

结构说明：
- `backend/main.go` 负责加载配置、连接数据库、自动迁移、初始化默认管理员、检查未完成任务并启动 Gin
- `backend/router/router.go` 统一注册 `/api` 路由
- `backend/models/` 定义核心数据模型与关联关系
- `backend/task/` 提供长任务管理能力，用于批量导入等异步流程
- `frontend/src/App.tsx` 定义应用路由与全局 Provider
- `frontend/src/services/api.ts` 统一处理 API 请求、JWT 注入与 `401` 跳转

## 🗄️ 核心数据模型

### 主要实体

```text
Department
  部门信息

User
  用户基础信息（employee_id, employee_no, name, department_id, role, status）

UserPermission
  用户对生产线的权限（can_view, can_download, can_upload, can_manage）

DepartmentPermission
  部门对生产线的权限（can_view, can_download, can_upload, can_manage）

Process
  工序信息（name, code, type, sort_order）

ProductionLine
  生产线信息（name, code, type, process_id, status）

ProductionLineCustomField
  生产线自定义字段模板（name, field_type, options_json, sort_order, enabled）

VehicleModel
  车型信息（name, code, series, status）

Program
  程序主实体（name, code, production_line_id, vehicle_model_id, version, status, editing_by）

ProgramFile
  程序文件记录（file_name, file_path, file_size, file_type, version, uploaded_by）

ProgramVersion
  程序版本记录（program_id, version, file_id, uploaded_by, change_log, is_current）

ProgramRelation
  程序关联关系（source_program_id, related_program_id, relation_type）

ProgramCustomFieldValue
  程序在自定义字段上的取值
```

### 关系概览
- `Department` ← `User`（一对多）
- `Process` ← `ProductionLine`（一对多）
- `ProductionLine` ← `Program`（一对多）
- `VehicleModel` ← `Program`（一对多）
- `User` ← `UserPermission`（一对多）
- `Department` ← `DepartmentPermission`（一对多）
- `ProductionLine` ← `UserPermission` / `DepartmentPermission`（一对多）
- `ProductionLine` ← `ProductionLineCustomField`（一对多）
- `Program` ← `ProgramFile` / `ProgramVersion`（一对多）
- `Program` ← `ProgramRelation`（程序自关联）
- `Program` ← `ProgramCustomFieldValue`（一对多）

## 🔒 认证与权限

- 所有受保护接口统一挂在 `/api` 下，并通过 JWT Bearer Token 认证
- 管理员接口额外使用管理员中间件限制访问
- 前端将 token 与用户信息保存在 `localStorage`
- 前端包含 4 小时无交互自动登出机制

## 🌐 API 示例

### 公开接口
```text
POST /api/login                 # 登录
POST /api/register              # 注册
```

### 用户与部门
```text
GET    /api/users
GET    /api/users/:id
POST   /api/users
PUT    /api/users/:id
DELETE /api/users/:id
PUT    /api/users/:id/password
PUT    /api/users/:id/reset-password

GET    /api/departments
GET    /api/departments/:id
POST   /api/departments
PUT    /api/departments/:id
DELETE /api/departments/:id
```

### 工序、生产线、车型
```text
GET    /api/processes
POST   /api/processes
PUT    /api/processes/:id
DELETE /api/processes/:id

GET    /api/production-lines
GET    /api/production-lines/:id
GET    /api/production-lines/:id/custom-fields
POST   /api/production-lines
PUT    /api/production-lines/:id
DELETE /api/production-lines/:id
POST   /api/production-lines/:id/custom-fields
PUT    /api/production-lines/:id/custom-fields/:fieldId
DELETE /api/production-lines/:id/custom-fields/:fieldId

GET    /api/vehicle-models
POST   /api/vehicle-models
PUT    /api/vehicle-models/:id
DELETE /api/vehicle-models/:id
```

### 程序、文件、版本、关联
```text
GET    /api/programs
GET    /api/programs/:id
POST   /api/programs
PUT    /api/programs/:id
PUT    /api/programs/:id/custom-field-values
DELETE /api/programs/:id
GET    /api/programs/by-vehicle/:vehicle_id

POST   /api/files/upload
GET    /api/files/program/:program_id
GET    /api/files/download/:id
GET    /api/files/download/program/:program_id/latest
GET    /api/files/download/version/:version
DELETE /api/files/:id

GET    /api/versions/program/:program_id
POST   /api/versions
PUT    /api/versions/:id/activate

GET    /api/relations/program/:program_id
POST   /api/relations
DELETE /api/relations/:id
```

### 权限、备份、迁移、任务
```text
GET    /api/permissions
POST   /api/permissions
PUT    /api/permissions/:id
DELETE /api/permissions/:id
GET    /api/permissions/user/:user_id
GET    /api/permissions/user/:user_id/effective

GET    /api/department-permissions
POST   /api/department-permissions
PUT    /api/department-permissions/:id
DELETE /api/department-permissions/:id

POST   /api/backup/database
POST   /api/backup/files
POST   /api/backup/full
GET    /api/backup
GET    /api/backup/download/:name
DELETE /api/backup/:name
POST   /api/backup/restore/database/:name
POST   /api/backup/restore/files/:name

GET    /api/migration/status
POST   /api/migration/start
POST   /api/migration/rollback
```

## 🧪 开发命令

```bash
make install          # 安装依赖
make dev              # 查看开发启动提示
make build            # 构建前后端
make test             # 运行全部测试
make lint             # 运行全部 lint

cd backend && go test ./... -v -cover
cd frontend && npm test -- --run
cd frontend && npm run lint
```

## 📌 说明

- 当前仓库的初始化脚本入口位于根目录 `init_all.go`
- 后端在存在 `frontend/dist` 时可直接托管已构建的前端静态资源
- README 仅展示当前代码中可确认的主路径；更完整的调用细节请以 `backend/router/router.go` 和前端页面实现为准
