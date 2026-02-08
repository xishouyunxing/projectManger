# 🚀 部署指南

本指南详细说明如何在不同环境中部署起重机生产线程序管理系统。

## 📋 环境要求

### 🖥️ 系统要求
- **操作系统**: Linux (推荐 Ubuntu 20.04+) / Windows Server 2019+ / macOS
- **内存**: 最低 4GB，推荐 8GB+
- **存储**: 最低 20GB 可用空间
- **网络**: 稳定的互联网连接

### 🔧 后端环境
- **Go 1.23+** - 后端开发语言
- **MySQL 8.0+** - 数据库服务
- **Git** - 版本控制

### 🎨 前端环境
- **Node.js 18+** - JavaScript运行时
- **npm 9+** - 包管理器

### 🐳 容器化部署（可选）
- **Docker 20.10+**
- **Docker Compose 2.0+**

## 💻 本地开发部署

### 🗄️ 1. 数据库准备

**步骤一：创建数据库和用户**
```sql
-- 连接MySQL
mysql -u root -p

-- 创建数据库
CREATE DATABASE zlzk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建专用用户（推荐）
CREATE USER 'zlzk'@'localhost' IDENTIFIED BY 'zlzk.12345678';
GRANT ALL PRIVILEGES ON zlzk.* TO 'zlzk'@'localhost';
FLUSH PRIVILEGES;
```

### 🔧 2. 后端部署

```bash
# 进入项目根目录
cd crane-system

# 复制环境配置文件
cp .env.example .env

# 编辑 .env 文件（位于根目录），配置数据库连接
nano .env
```

**.env 配置示例：**
```env
# 数据库配置
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=zlzk
DB_PASSWORD=zlzk.12345678
DB_NAME=zlzk

# JWT配置
JWT_SECRET=crane-system-jwt-secret-key-2024

# 服务器配置
SERVER_PORT=8080

```

**启动后端服务：**
```bash
# 进入后端目录
cd backend

# 安装依赖
go mod tidy

# 开发模式启动
go run main.go

# 或构建后运行
go build -o crane-system.exe    # Windows
go build -o crane-system       # Linux/macOS
./crane-system.exe             # Windows
./crane-system                 # Linux/macOS
```

后端服务将在 `http://localhost:8080` 启动

### 🎨 3. 前端部署

```bash
# 新开终端，进入前端目录
cd frontend

# 安装依赖
npm install

# 开发模式启动
npm run dev

# 构建生产版本
npm run build
```

前端开发服务器将在 `http://localhost:3000` 启动

### 👤 4. 创建管理员账号

```bash
# 在backend目录下运行初始化脚本
cd backend
go run init_all.go
```

**默认管理员账号：**
- 工号：`admin001`
- 密码：`admin123456`

## 🏭 生产环境部署

### 🔧 1. 后端部署

#### 🐧 使用 systemd (Linux推荐)

**创建服务文件：**
```bash
sudo nano /etc/systemd/system/crane-system.service
```

**服务配置：**
```ini
[Unit]
Description=Crane Production Line Management System
After=network.target mysql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/crane-system
Environment=GIN_MODE=release
ExecStart=/opt/crane-system/crane-system
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=crane-system

# 安全设置
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/crane-system/uploads /opt/crane-system/backups

[Install]
WantedBy=multi-user.target
```

**启动和管理服务：**
```bash
# 重新加载systemd配置
sudo systemctl daemon-reload

# 启用服务（开机自启）
sudo systemctl enable crane-system

# 启动服务
sudo systemctl start crane-system

# 查看服务状态
sudo systemctl status crane-system

# 查看日志
sudo journalctl -u crane-system -f
```

#### 🐳 使用 Docker

**后端 Dockerfile：**
```dockerfile
# 多阶段构建
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装git（用于go mod）
RUN apk add --no-cache git

# 复制go mod文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o crane-system

# 运行阶段
FROM alpine:latest

# 安装ca证书和时区数据
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN adduser -D -s /bin/sh crane

# 创建必要目录
WORKDIR /app
RUN mkdir -p uploads backups && chown -R crane:crane /app

# 复制构建的二进制文件
COPY --from=builder /app/crane-system .
COPY --from=builder /app/.env.example .env

# 切换到非root用户
USER crane

# 暴露端口
EXPOSE 3000

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/api/health || exit 1

# 启动命令
CMD ["./crane-system"]
```

**构建和运行：**
```bash
# 构建镜像
docker build -t crane-backend:latest .

# 运行容器
docker run -d \
  --name crane-backend \
  --restart unless-stopped \
  -p 3000:3000 \
  -v /opt/crane-data/uploads:/app/uploads \
  -v /opt/crane-data/backups:/app/backups \
  -v /opt/crane-data/.env:/app/.env \
  crane-backend:latest
```

### 🎨 2. 前端部署

#### 🌐 使用 Nginx（推荐生产方案）

**构建前端：**
```bash
cd frontend
npm ci --production
npm run build
```

**部署到服务器：**
```bash
# 创建部署目录
sudo mkdir -p /var/www/crane-system

# 复制构建文件
sudo cp -r dist/* /var/www/crane-system/

# 设置权限
sudo chown -R www-data:www-data /var/www/crane-system
sudo chmod -R 755 /var/www/crane-system
```

**Nginx 配置：**
```bash
sudo nano /etc/nginx/sites-available/crane-system
```

**完整的 Nginx 配置：**
```nginx
server {
    listen 80;
    server_name your-domain.com;  # 替换为实际域名
    
    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;  # 替换为实际域名
    
    # SSL证书配置
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
    
    # 前端静态文件
    root /var/www/crane-system;
    index index.html;
    
    # 安全头部
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
    
    # Gzip压缩
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;
    
    # 前端路由（支持React Router）
    location / {
        try_files $uri $uri/ /index.html;
        
        # 缓存设置
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }
    
    # API代理到后端
    location /api {
        proxy_pass http://127.0.0.1:3000;  # 后端服务地址
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # 文件上传大小限制
    client_max_body_size 100M;
    
    # 访问日志
    access_log /var/log/nginx/crane-system.access.log;
    error_log /var/log/nginx/crane-system.error.log;
}
```

**启用配置：**
```bash
# 启用站点
sudo ln -s /etc/nginx/sites-available/crane-system /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载配置
sudo systemctl reload nginx
```

#### 🐳 使用 Docker 容器化前端

**前端 Dockerfile：**
```dockerfile
# 构建阶段
FROM node:18-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制package文件
COPY package*.json ./

# 安装依赖
RUN npm ci --production

# 复制源代码
COPY . .

# 构建应用
RUN npm run build

# 生产阶段
FROM nginx:alpine

# 安装必要工具
RUN apk add --no-cache curl

# 复制构建文件
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制Nginx配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost/ || exit 1

# 暴露端口
EXPOSE 80

# 启动Nginx
CMD ["nginx", "-g", "daemon off;"]
```

**nginx.conf：**
```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;
    
    # 前端路由
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # API代理
    location /api {
        proxy_pass http://backend:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # 文件上传限制
    client_max_body_size 100M;
}
```

### 🐳 3. 使用 Docker Compose（一体化部署 - 推荐）

**生产级 docker-compose.yml：**
```yaml
version: '3.9'

services:
  # 数据库服务
  mysql:
    image: mysql:8.0
    container_name: crane-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-crane_root_password}
      MYSQL_DATABASE: ${DB_NAME:-zlzk}
      MYSQL_USER: ${DB_USER:-zlzk}
      MYSQL_PASSWORD: ${DB_PASSWORD:-zlzk.12345678}
    volumes:
      - mysql_data:/var/lib/mysql
      - ./docker/mysql/my.cnf:/etc/mysql/conf.d/my.cnf:ro
      - ./backups:/backups
    networks:
      - crane-network
    command: --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      timeout: 20s
      retries: 10

  # 后端服务
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: crane-backend
    restart: unless-stopped
    environment:
      DB_HOST: mysql
      DB_PORT: 3306
      DB_USER: ${DB_USER:-zlzk}
      DB_PASSWORD: ${DB_PASSWORD:-zlzk.12345678}
      DB_NAME: ${DB_NAME:-zlzk}
      JWT_SECRET: ${JWT_SECRET:-crane-system-jwt-secret-key-2024}
      SERVER_PORT: 3000
      CORS_ORIGINS: ${CORS_ORIGINS:-http://localhost,https://your-domain.com}
    volumes:
      - ./uploads:/app/uploads
      - ./backups:/app/backups
      - ./.env:/app/.env:ro
    depends_on:
      mysql:
        condition: service_healthy
    networks:
      - crane-network
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # 前端服务
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: crane-frontend
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./docker/nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./docker/nginx/ssl:/etc/nginx/ssl:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      backend:
        condition: service_healthy
    networks:
      - crane-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Redis 缓存服务（可选）
  redis:
    image: redis:7-alpine
    container_name: crane-redis
    restart: unless-stopped
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD:-crane_redis_password}
    volumes:
      - redis_data:/data
    networks:
      - crane-network
    healthcheck:
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

# 数据卷
volumes:
  mysql_data:
    driver: local
  redis_data:
    driver: local

# 网络
networks:
  crane-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

**环境配置文件 .env：**
```env
# 数据库配置
MYSQL_ROOT_PASSWORD=your_secure_root_password
DB_USER=zlzk
DB_PASSWORD=zlzk.12345678
DB_NAME=zlzk

# JWT配置
JWT_SECRET=crane-system-jwt-secret-key-2024-change-this

# 服务器配置
CORS_ORIGINS=http://localhost,https://your-domain.com

# Redis配置（可选）
REDIS_PASSWORD=crane_redis_password
```

**启动和管理：**
```bash
# 构建并启动所有服务
docker-compose up -d --build

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f frontend

# 重启服务
docker-compose restart backend

# 停止所有服务
docker-compose down

# 完全清理（包括数据卷）
docker-compose down -v
```

**健康检查：**
```bash
# 检查所有服务健康状态
docker-compose exec backend wget --quiet --tries=1 --spider http://localhost:3000/api/health && echo "Backend OK"
docker-compose exec frontend curl --silent --fail http://localhost/ && echo "Frontend OK"
```

## 🔒 HTTPS 配置

### 📜 Let's Encrypt 自动证书（推荐）

```bash
# 安装 Certbot
sudo apt update
sudo apt install certbot python3-certbot-nginx

# 申请证书（自动配置Nginx）
sudo certbot --nginx -d your-domain.com

# 设置自动续期
sudo crontab -e
# 添加以下行：
# 0 12 * * * /usr/bin/certbot renew --quiet
```

### 🔐 手动SSL证书配置

**生成自签名证书（开发环境）：**
```bash
# 创建SSL目录
sudo mkdir -p /etc/nginx/ssl

# 生成私钥和证书
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/nginx/ssl/crane-system.key \
  -out /etc/nginx/ssl/crane-system.crt
```

## 👤 管理员账号初始化

### 🚀 方法一：使用初始化脚本（推荐）

```bash
# 在backend目录下运行
cd backend
go run init_all.go

# 或者使用Go工作空间
cd ..
go run backend/init_all.go
```

### 🔧 方法二：通过API注册

```bash
# 注册普通用户
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "employee_id": "admin001",
    "employee_no": "admin001",
    "name": "系统管理员",
    "department": "IT部门",
    "password": "admin123456"
  }'

# 登录获取token
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "employee_id": "admin001",
    "password": "admin123456"
  }'

# 使用管理员权限更新角色（如果需要）
# 或者直接在数据库中修改
mysql -u zlzk -p'zlzk.12345678' zlzk
UPDATE users SET role='admin', status='active' WHERE employee_id='admin001';
```

### 🗄️ 方法三：直接数据库插入

```sql
-- 连接数据库
mysql -u zlzk -p'zlzk.12345678' zlzk

-- 插入管理员用户
INSERT INTO users (employee_id, employee_no, name, department, role, password, status, created_at, updated_at) 
VALUES ('admin001', 'admin001', '系统管理员', 'IT部门', 'admin', 
        '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVEFDa', 
        'active', NOW(), NOW());

-- 验证插入
SELECT * FROM users WHERE employee_id='admin001';
```

**默认管理员账号：**
- 工号：`admin001`
- 密码：`admin123456`
- 角色：系统管理员
- 部门：IT部门

## 💾 备份和恢复策略

### 🔄 自动备份配置

**数据库自动备份脚本：**
```bash
#!/bin/bash
# /opt/crane-system/scripts/backup_db.sh

BACKUP_DIR="/opt/crane-data/backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="zlzk"
DB_USER="zlzk"
DB_PASS="zlzk.12345678"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 数据库备份
mysqldump -u $DB_USER -p$DB_PASS \
  --single-transaction \
  --routines \
  --triggers \
  --events \
  $DB_NAME > $BACKUP_DIR/database_backup_$DATE.sql

# 压缩备份文件
gzip $BACKUP_DIR/database_backup_$DATE.sql

# 删除30天前的备份
find $BACKUP_DIR -name "database_backup_*.sql.gz" -mtime +30 -delete

echo "数据库备份完成: database_backup_$DATE.sql.gz"
```

**文件备份脚本：**
```bash
#!/bin/bash
# /opt/crane-system/scripts/backup_files.sh

BACKUP_DIR="/opt/crane-data/backups"
SOURCE_DIR="/opt/crane-system/uploads"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 文件备份
tar -czf $BACKUP_DIR/files_backup_$DATE.tar.gz -C $(dirname $SOURCE_DIR) $(basename $SOURCE_DIR)

# 删除30天前的备份
find $BACKUP_DIR -name "files_backup_*.tar.gz" -mtime +30 -delete

echo "文件备份完成: files_backup_$DATE.tar.gz"
```

**设置定时备份：**
```bash
# 编辑crontab
sudo crontab -e

# 添加定时任务
# 每天凌晨2点备份数据库
0 2 * * * /opt/crane-system/scripts/backup_db.sh

# 每天凌晨3点备份文件
0 3 * * * /opt/crane-system/scripts/backup_files.sh

# 每周日凌晨4点完整备份
0 4 * * 0 /opt/crane-system/scripts/full_backup.sh
```

### 🛡️ 系统监控

#### 📊 应用性能监控

**使用系统服务监控：**
```bash
# 查看服务状态
sudo systemctl status crane-system

# 查看资源使用
htop
df -h
free -h
```

#### 🔍 日志管理

**配置日志轮转：**
```bash
sudo nano /etc/logrotate.d/crane-system
```

```
/opt/crane-data/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 www-data www-data
    postrotate
        systemctl reload crane-system
    endscript
}
```

## 🚨 故障排除

### 🔌 数据库连接问题

**症状：** 后端无法连接到数据库

**排查步骤：**
```bash
# 1. 检查MySQL服务状态
sudo systemctl status mysql

# 2. 检查数据库配置
mysql -u zlzk -p'zlzk.12345678' -h 127.0.0.1 zlzk

# 3. 检查网络连接
telnet 127.0.0.1 3306

# 4. 检查环境变量配置
cat /opt/crane-system/.env

# 5. 查看应用日志
sudo journalctl -u crane-system -f
```

**常见解决方案：**
- 重启MySQL服务：`sudo systemctl restart mysql`
- 检查防火墙：`sudo ufw status`
- 验证用户权限：`mysql -u root -p -e "SHOW GRANTS FOR 'zlzk'@'localhost';"`

### 📁 文件上传问题

**症状：** 文件上传失败或无法访问

**排查步骤：**
```bash
# 1. 检查目录权限
ls -la /opt/crane-system/uploads/

# 2. 检查磁盘空间
df -h /opt/crane-system/

# 3. 检查Nginx配置
sudo nginx -t

# 4. 查看Nginx日志
sudo tail -f /var/log/nginx/error.log
```

**解决方案：**
```bash
# 修复权限
sudo chown -R www-data:www-data /opt/crane-system/uploads/
sudo chmod -R 755 /opt/crane-system/uploads/

# 增加上传限制（在Nginx配置中）
client_max_body_size 200M;
```

### 🌐 前端访问问题

**症状：** 前端页面无法访问或API请求失败

**排查步骤：**
```bash
# 1. 检查Nginx状态
sudo systemctl status nginx

# 2. 检查端口占用
sudo netstat -tlnp | grep :80
sudo netstat -tlnp | grep :443

# 3. 测试后端API
curl -I http://localhost:3000/api/health

# 4. 检查DNS解析
nslookup your-domain.com
```

## 🔒 安全加固

### 🛡️ 系统安全配置

**1. 防火墙配置：**
```bash
# 启用UFW防火墙
sudo ufw enable

# 允许SSH
sudo ufw allow 22

# 允许HTTP/HTTPS
sudo ufw allow 80
sudo ufw allow 443

# 拒绝其他端口
sudo ufw default deny incoming

# 查看状态
sudo ufw status
```

**2. Fail2Ban防暴力破解：**
```bash
# 安装Fail2Ban
sudo apt install fail2ban

# 配置Nginx防护
sudo nano /etc/fail2ban/jail.local
```

```ini
[nginx-http-auth]
enabled = true
filter = nginx-http-auth
logpath = /var/log/nginx/error.log

[nginx-limit-req]
enabled = true
filter = nginx-limit-req
logpath = /var/log/nginx/error.log
maxretry = 10
```

**3. 定期安全更新：**
```bash
# 自动安全更新
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

### 🔐 应用安全最佳实践

**1. 环境变量安全：**
```bash
# 设置适当的文件权限
chmod 600 /opt/crane-system/.env
chown www-data:www-data /opt/crane-system/.env
```

**2. 定期密码轮换：**
- 每90天更换数据库密码
- 每60天更换JWT密钥
- 管理员密码复杂度要求

**3. SSL/TLS配置：**
```nginx
# 在Nginx配置中添加
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
```

## 📈 性能优化

### ⚡ 数据库优化

```sql
-- MySQL配置优化
SET GLOBAL innodb_buffer_pool_size = 1G;
SET GLOBAL query_cache_size = 256M;
SET GLOBAL max_connections = 200;
```

### 🚀 应用优化

**Go应用优化：**
```bash
# 启用Go编译优化
go build -ldflags="-s -w" -o crane-system

# 设置环境变量
export GIN_MODE=release
export GOGC=100
```

**Nginx优化：**
```nginx
# 启用HTTP/2
listen 443 ssl http2;

# 启用Brotli压缩
brotli on;
brotli_comp_level 6;
brotli_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
```

**重要提醒：在生产环境中，请确保：**
- ✅ 已修改所有默认密码
- ✅ 已启用HTTPS加密
- ✅ 已配置定期备份
- ✅ 已设置监控系统
- ✅ 已制定应急响应计划
