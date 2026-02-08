# 起重机生产线程序管理系统 - 前端

## 技术栈

- **React 18**
- **TypeScript**
- **Vite** - 构建工具
- **Ant Design** - UI组件库
- **React Router** - 路由管理
- **Axios** - HTTP客户端

## 快速开始

### 1. 安装依赖

```bash
cd frontend
npm install
```

### 2. 启动开发服务器

```bash
npm run dev
```

应用将在 `http://localhost:3000` 启动

### 3. 构建生产版本

```bash
npm run build
```

构建产物将生成在 `dist` 目录

## 功能模块

### 1. 登录认证
- 使用工号和密码登录
- JWT Token认证
- 自动登录保持

### 2. 仪表盘
- 系统数据概览
- 统计图表展示

### 3. 程序管理
- 程序列表查看
- 新建/编辑程序
- 文件上传下载
- 版本管理
- 按车型查看程序

### 4. 用户管理（管理员）
- 用户列表
- 新建/编辑用户
- 重置密码
- 角色权限管理

### 5. 生产线管理（管理员）
- 生产线列表
- 新建/编辑生产线
- 上车/下车分类

### 6. 车型管理
- 车型列表
- 新建/编辑车型
- 系列分类

### 7. 权限管理（管理员）
- 用户权限分配
- 生产线访问控制
- 查看/下载/上传/管理权限

## 项目结构

```
frontend/
├── public/              # 静态资源
├── src/
│   ├── components/      # 组件
│   │   ├── Layout.tsx   # 主布局
│   │   └── PrivateRoute.tsx  # 路由守卫
│   ├── contexts/        # Context
│   │   └── AuthContext.tsx   # 认证上下文
│   ├── pages/           # 页面
│   │   ├── Login.tsx
│   │   ├── Dashboard.tsx
│   │   ├── ProgramManagement.tsx
│   │   ├── UserManagement.tsx
│   │   ├── ProductionLineManagement.tsx
│   │   ├── VehicleModelManagement.tsx
│   │   └── PermissionManagement.tsx
│   ├── services/        # 服务
│   │   └── api.ts       # API封装
│   ├── App.tsx          # 根组件
│   ├── main.tsx         # 入口文件
│   └── index.css        # 全局样式
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## 环境变量

开发环境默认代理到 `http://localhost:8080`

生产环境需要配置 Nginx 代理或设置 API 地址。

## 注意事项

1. 确保后端服务已启动
2. 首次使用需要创建管理员账号
3. 文件上传大小限制可在后端配置
4. 建议使用现代浏览器（Chrome, Firefox, Edge）
