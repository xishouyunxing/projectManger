# 起重机生产线程序管理系统

<p align="center">
  <strong>面向离线编程与产线程序交付的企业级管理平台</strong>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=20232A">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white">
  <img alt="Vite" src="https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite&logoColor=white">
  <img alt="MySQL" src="https://img.shields.io/badge/MySQL-8.0-4479A1?style=flat-square&logo=mysql&logoColor=white">
</p>

<p align="center">
  <a href="#功能亮点">功能亮点</a> ·
  <a href="#快速开始">快速开始</a> ·
  <a href="#权限模型">权限模型</a> ·
  <a href="#部署方式">部署方式</a> ·
  <a href="#项目结构">项目结构</a>
</p>

---

## 项目简介

本项目用于管理起重机生产线的离线编程程序、程序文件、车型、生产线、部门、用户与权限。系统以“程序版本可追溯、文件交付可管控、产线授权可精确配置”为核心目标，支持从开发调试到内网部署的完整工作流。

适合以下场景：

- 多条生产线共用程序库，需要按产线、车型、工序组织程序。
- 程序文件需要上传、下载、版本追踪、批量导入和完整性治理。
- 管理员需要精确控制用户、部门、角色默认权限和部门默认权限。
- 系统需要在企业内网中以 Go 后端托管前端静态资源，或以前后端分离方式运行。

## 功能亮点

### 程序与文件管理

- 程序基础信息维护，支持按产线、车型、状态等维度检索。
- 程序文件上传、下载、删除与历史版本记录。
- 最新版本下载、指定版本下载、程序关联关系维护。
- Excel 导出、批量导入、批量上传和长任务状态查询。
- 文件忽略列表、文件迁移、文件完整性检查等运维能力。

### 主数据管理

- 生产线、工序、车型的统一维护。
- 生产线自定义字段模板，支持程序扩展字段值。
- 部门与用户管理，支持管理员重置密码和用户修改密码。
- 首页 Dashboard 汇总程序矩阵、完成率和关键业务状态。

### 权限与安全

- JWT Bearer Token 认证，受保护接口统一挂载在 `/api` 下。
- 管理员接口使用额外中间件限制访问。
- 用户权限、部门权限、角色默认权限、部门默认权限分层管理。
- 权限矩阵支持“继承 / 显式覆盖 / 显式拒绝”三态语义。
- CORS 来源通过配置白名单控制，不允许使用 `*`。

### 部署与运维

- 支持 MySQL 8.0。
- 支持 Docker Compose 一键启动 MySQL、后端和前端。
- 支持 Go 后端直接托管 `frontend/dist`，便于内网单服务部署。
- 支持数据库备份、文件备份、全量备份和恢复接口。
- 支持自动迁移开关，开发环境默认开启，生产环境建议关闭。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | React 18, TypeScript, Vite 5, Ant Design 5, React Router 6, Axios, Day.js |
| 后端 | Go 1.25, Gin, GORM, JWT, MySQL Driver, Excelize |
| 数据库 | MySQL 8.0 |
| 测试 | Vitest, Testing Library, Go test |
| 部署 | Docker, Docker Compose, Nginx, Go static file serving |

## 快速开始

### 1. 准备环境

建议版本：

- Go 1.25 或兼容版本
- Node.js 18+
- MySQL 8.0+
- npm 9+

### 2. 初始化配置

复制环境变量模板：

```bash
cp .env.example .env
```

关键配置说明：

| 变量 | 说明 |
| --- | --- |
| `APP_ENV` | 运行环境，建议使用 `development`、`test` 或 `production` |
| `SERVER_PORT` | 后端服务端口，默认 `8080` |
| `AUTO_MIGRATE` | 是否启动时自动执行迁移，生产环境建议设为 `false` |
| `FRONTEND_DIST` | Go 后端托管前端时使用的 `dist` 目录 |
| `DB_HOST` / `DB_PORT` | 数据库地址与端口 |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | 数据库账号、密码和库名 |
| `JWT_SECRET` | JWT 密钥，至少 32 个字符 |
| `DEFAULT_PASSWORD` | 初始化用户默认密码，至少 8 个字符 |
| `CORS_ALLOWED_ORIGINS` | 允许访问后端的前端来源，逗号分隔 |
| `UPLOADS_DIR` | 程序文件上传目录 |
| `BACKUPS_DIR` | 备份文件目录 |

### 3. 准备数据库

```sql
CREATE DATABASE crane_system CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE USER 'crane_user'@'localhost' IDENTIFIED BY 'zlzk.12345678';
GRANT ALL PRIVILEGES ON crane_system.* TO 'crane_user'@'localhost';
FLUSH PRIVILEGES;
```

### 4. 安装依赖

```bash
make install
```

也可以分别安装：

```bash
cd backend
go mod download

cd ../frontend
npm install
```

### 5. 初始化系统数据

```bash
make init-data
```

等价命令：

```bash
cd backend
go run -tags initcmd ./init_main.go ./init_all.go
```

初始化会创建基础数据和管理员账号。默认管理员工号为：

```text
admin001
```

密码来自 `.env` 中的 `DEFAULT_PASSWORD`。

### 6. 启动开发环境

后端：

```bash
make backend-run
```

前端：

```bash
make frontend-run
```

默认访问地址：

| 服务 | 地址 |
| --- | --- |
| 前端开发服务 | `http://localhost:3000` |
| 后端 API | `http://localhost:8080/api` |

Vite 开发服务器会把 `/api` 代理到 `http://localhost:8080`。

## 常用命令

| 命令 | 说明 |
| --- | --- |
| `make install` | 安装前后端依赖 |
| `make init-data` | 初始化系统基础数据 |
| `make backend-run` | 启动后端服务 |
| `make frontend-run` | 启动前端开发服务 |
| `make build` | 构建后端二进制与前端产物 |
| `make docker-up` | 使用 Docker Compose 启动服务 |
| `make docker-down` | 停止 Docker Compose 服务 |
| `make docker-logs` | 查看容器日志 |
| `npm run build` | 构建前端生产产物 |
| `npm run test -- --run` | 运行前端测试 |
| `go test ./...` | 运行后端测试 |

## 权限模型

系统按生产线粒度控制权限，权限位包括：

- `can_view`：查看
- `can_download`：下载
- `can_upload`：上传
- `can_manage`：管理

有效权限按以下优先级解析：

```text
用户显式权限
  > 部门显式权限
  > 角色默认权限
  > 部门默认权限
  > 无权限
```

用户和部门权限矩阵支持三态：

| 状态 | 含义 |
| --- | --- |
| 继承 | 当前层不保存显式配置，继续向下级默认权限回落 |
| 显式允许 | 当前层保存权限位，优先于继承来源 |
| 显式拒绝 | 当前层保存全 false 权限位，用于阻断继承权限 |

注意：全 false 的显式覆盖不是空配置。管理员在用户或部门矩阵中关闭所有权限并保存时，系统会保留该覆盖记录，用来明确拒绝访问。

## API 概览

公开接口：

```text
POST /api/login
```

受保护接口需要携带 JWT：

```http
Authorization: Bearer <token>
```

主要接口分组：

| 分组 | 说明 |
| --- | --- |
| `/api/users` | 用户管理、修改密码、重置密码 |
| `/api/departments` | 部门管理 |
| `/api/production-lines` | 生产线与自定义字段管理 |
| `/api/processes` | 工序管理 |
| `/api/vehicle-models` | 车型管理 |
| `/api/programs` | 程序管理、导出、批量导入、按车型查询 |
| `/api/files` | 文件上传、下载、版本文件查询与删除 |
| `/api/versions` | 程序版本管理与版本激活 |
| `/api/permissions` | 用户权限、用户权限矩阵、有效权限查询 |
| `/api/department-permissions` | 部门权限与部门权限矩阵 |
| `/api/permission-defaults` | 角色默认权限与部门默认权限 |
| `/api/program-mappings` | 程序上下级映射关系 |
| `/api/backup` | 数据库、文件、全量备份与恢复 |
| `/api/migration` | 迁移状态、启动与回滚 |
| `/api/tasks` | 长任务状态查询 |

## 部署方式

### Docker Compose

```bash
docker-compose up -d --build
```

默认服务：

| 服务 | 端口 |
| --- | --- |
| 前端 Nginx | `80` |
| 后端 API | `8080` |
| MySQL | `3306` |

生产部署前请修改：

- `JWT_SECRET`
- `DEFAULT_PASSWORD`
- `DB_PASSWORD`
- `CORS_ALLOWED_ORIGINS`
- 是否开启 `AUTO_MIGRATE`

### Go 后端托管前端

先构建前端：

```bash
cd frontend
npm install
npm run build
```

设置 `.env`：

```env
FRONTEND_DIST=../frontend/dist
```

启动后端：

```bash
cd backend
go run .
```

当 `FRONTEND_DIST/index.html` 存在时，Go 服务会对非 `/api`、非 `/uploads` 的 GET 请求回退到前端 SPA。

## 项目结构

```text
projectManger/
├── backend/                 # Gin + GORM 后端服务
│   ├── app/                 # 启动装配、运行目录和服务构建
│   ├── config/              # 环境变量与配置校验
│   ├── controllers/         # HTTP 控制器
│   ├── database/            # 数据库连接与迁移
│   ├── middleware/          # 认证与管理员中间件
│   ├── models/              # GORM 数据模型
│   └── router/              # 路由注册与前端托管
├── frontend/                # React + Vite 前端
│   ├── src/contexts/        # 登录态与全局上下文
│   ├── src/pages/           # 业务页面
│   ├── src/services/        # API 请求封装
│   └── src/components/      # 公共组件
├── docs/                    # 项目文档
├── deploy/                  # 部署相关文件
├── uploads/                 # 运行时上传目录
├── backups/                 # 运行时备份目录
├── docker-compose.yml       # 本地容器编排
├── Makefile                 # 常用开发命令
└── README.md
```

## 数据模型概览

| 模型 | 说明 |
| --- | --- |
| `User` | 系统登录账号与用户基础信息 |
| `Department` | 部门信息 |
| `UserPermission` | 用户对生产线的显式权限覆盖 |
| `DepartmentPermission` | 部门对生产线的显式权限覆盖 |
| `RoleDefaultPermission` | 角色默认权限 |
| `DepartmentDefaultPermission` | 部门默认权限 |
| `Process` | 工序 |
| `ProductionLine` | 生产线 |
| `ProductionLineCustomField` | 生产线自定义字段模板 |
| `VehicleModel` | 车型 |
| `Program` | 程序主数据 |
| `ProgramFile` | 程序文件记录 |
| `ProgramVersion` | 程序版本记录 |
| `ProgramRelation` | 程序关联关系 |
| `ProgramMapping` | 程序父子映射关系 |
| `ProgramCustomFieldValue` | 程序自定义字段值 |

## 开发规范

- 后端接口统一挂载在 `/api` 下。
- 前端业务请求统一通过 `frontend/src/services/api.ts` 发起。
- 用户与部门权限矩阵保存时只提交脏行，避免把继承权限固化为显式覆盖。
- 用户/部门矩阵中的全 false 覆盖表示显式拒绝，不能按空配置删除。
- 生产环境建议关闭 `AUTO_MIGRATE`，改用受控迁移流程。
- `uploads/`、`backups/`、`.perf-logs/` 等运行或测量产物不应提交到版本库。

## 测试与质量检查

后端：

```bash
cd backend
go test ./...
```

前端：

```bash
cd frontend
npm run build
npm run test -- --run
```

格式与静态检查：

```bash
cd frontend
npm run lint
npm run format:check
```

## 常见问题

### 登录失败或提示配置错误

检查 `.env` 中的 `JWT_SECRET` 和 `DEFAULT_PASSWORD` 是否存在，并确认长度满足要求。

### 前端请求后端失败

开发环境确认：

- 后端运行在 `http://localhost:8080`
- 前端运行在 `http://localhost:3000`
- `.env` 中 `CORS_ALLOWED_ORIGINS` 包含前端地址
- Vite 代理配置仍指向 `http://localhost:8080`

### 权限关闭后用户仍然可访问

确认是在用户或部门权限矩阵中启用了“覆盖”后再关闭权限位。单纯继承状态下关闭显示值不会阻断角色默认或部门默认权限。

### Docker 启动后不能直接用于生产

`docker-compose.yml` 中包含示例密码和示例 JWT 密钥。上线前必须替换所有默认凭据，并按部署环境调整 CORS、数据库账号和自动迁移策略。

## 相关文档

- [项目结构说明](docs/project-structure.md)
- [部署说明](DEPLOYMENT.md)
- [审查修复任务](docs/REVIEW-REMEDIATION-TODO.md)

## License

当前仓库未声明开源许可证。对外发布或分发前，请先补充 LICENSE 文件并明确使用范围。
