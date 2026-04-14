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
CREATE DATABASE crane_system CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建专用用户（推荐）
CREATE USER 'crane_user'@'localhost' IDENTIFIED BY 'zlzk.12345678';
GRANT ALL PRIVILEGES ON crane_system.* TO 'crane_user'@'localhost';
FLUSH PRIVILEGES;
```

### 2. 配置环境变量

复制模板并填写实际值：

```bash
cp .env.example .env
```

关键配置说明：
- `APP_ENV`：`development` / `test` / `production`
- `AUTO_MIGRATE`：开发环境默认允许自动迁移，生产环境默认关闭
- `JWT_SECRET`：必须至少 32 个字符
- `DEFAULT_PASSWORD`：初始化管理员账号时使用
- `UPLOADS_DIR` / `BACKUPS_DIR`：运行时目录
- `CORS_ALLOWED_ORIGINS`：逗号分隔，不允许 `*`
- `FRONTEND_DIST`：Go 进程直接托管前端时使用的构建产物目录

前端开发服务器默认运行在 `http://localhost:3000`，并将 `/api` 代理到 `http://localhost:8080`。

### 3. 启动方式

#### 方式 A：前后端分离开发（默认）

```bash
cd backend
go mod download
go run .
```

```bash
cd frontend
npm install
npm run dev
```

- 后端默认监听：`http://localhost:8080`
- 前端默认地址：`http://localhost:3000`

#### 方式 B：Go 直接托管前端

先构建前端：

```bash
cd frontend
npm install
npm run build
```

再启动后端：

```bash
cd backend
go mod download
go run .
```

此时需要保证 `FRONTEND_DIST` 指向前端构建目录，例如：

```env
FRONTEND_DIST=../frontend/dist
```

启动后由 Go 同时提供：
- 前端页面与静态资源
- `/api` 后端接口

### 4. 初始化系统数据

```bash
cd backend
go run -tags initcmd ./init_main.go ./init_all.go
```

初始化会自动创建：
- 数据库结构
- 部门
- 管理员账号

默认管理员账号：
- 工号：`admin001`
- 密码：`.env` 中的 `DEFAULT_PASSWORD`

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

结构说明：
- `backend/app/bootstrap.go` 负责配置加载、运行目录准备、数据库连接与启动装配
- `backend/config/config.go` 统一管理 `App / Database / Auth / Storage / Backup / CORS` 配置域
- `backend/router/router.go` 统一注册 `/api` 路由，并从配置读取 CORS 与静态资源目录；当 `FRONTEND_DIST` 可用时由 Go 直接托管前端构建产物
- `docs/project-structure.md` 说明源码目录、运行目录和本地工作目录边界

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

## 🔒 认证与权限

- 所有受保护接口统一挂在 `/api` 下，并通过 JWT Bearer Token 认证
- 管理员接口额外使用管理员中间件限制访问
- 前端将 token 与用户信息保存在 `localStorage`
- 前端包含 4 小时无交互自动登出机制

## 🌐 API 示例

### 公开接口
```text
POST /api/login                 # 登录
```
