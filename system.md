## 项目：植物养护日记

### 一、项目概述

这是一个面向家庭植物爱好者的养护记录工具。你可以记录家里每一盆植物的浇水、施肥、换盆、修剪等操作，系统根据植物品类自动推算下次养护时间并提醒你，还能通过照片记录植物的生长变化，形成每盆植物的专属成长相册。

**项目规模**：后端Go代码约2000-2100行，前端代码约700行，总计约22个文件。

### 二、功能模块

#### 1. 植物档案（350行）
- 录入植物：昵称、品种、入手日期、入手来源（花市/网购/朋友送的）
- 植物照片上传（头像图）
- 养护难度标注（好养/一般/需要细心）
- 植物状态：健康/黄叶/虫害/已送人/已枯萎
- 植物位置（阳台/客厅/卧室/书房）
- 编辑与删除

#### 2. 养护操作记录（400行）
- 记录操作类型：浇水、施肥、换盆、修剪、喷药、擦拭叶片
- 每次记录日期和备注（用量、时长等）
- 按时间线展示操作记录
- 支持上传对比照片（“今天长新叶了”）

#### 3. 养护周期与提醒（350行）
- 为每盆植物设置养护周期：
  - 浇水周期（如每3天一次 / 见干见湿）
  - 施肥周期（如每15天一次）
- 系统自动计算下次操作日期
- 首页展示“今天该做的事”列表
- 到期高亮提醒（已逾期标红）

#### 4. 植物生长时间线（250行）
- 按时间顺序展示某盆植物的所有操作记录
- 快速浏览从入手到现在的“成长史”
- 照片缩略图预览

#### 5. 多植物管理（150行）
- 所有植物列表展示（卡片模式）
- 按状态/位置筛选
- 快速查看上次浇水日期

#### 6. 统计看板（150行）
- 植物总数
- 各状态数量（健康/黄叶/虫害/已送人/已枯萎）
- 最近7天养护操作次数
- 本月新增植物数

### 三、技术栈

| 层级 | 技术选型 |
|------|----------|
| 后端框架 | Gin |
| ORM | GORM |
| 数据库 | SQLite / PostgreSQL |
| 文件存储 | 本地 / MinIO |
| 前端 | React + Tailwind CSS |
| 认证 | JWT |
| 配置管理 | Viper |
| 定时任务 | gocron（每日提醒计算） |

### 四、Go文件结构（22个文件）

```
plant-diary/
├── cmd/
│   └── server/
│       └── main.go                         // 入口 (90行)
├── config/
│   └── config.go                           // 配置 (60行)
├── internal/
│   ├── model/
│   │   ├── user.go                         // 用户 (40行)
│   │   ├── plant.go                        // 植物 (75行)
│   │   ├── care_log.go                     // 养护记录 (60行)
│   │   ├── care_cycle.go                   // 养护周期 (55行)
│   │   └── reminder.go                     // 提醒记录 (45行)
│   ├── repository/
│   │   ├── user_repo.go                    // (70行)
│   │   ├── plant_repo.go                   // (130行)
│   │   ├── care_log_repo.go                // (100行)
│   │   └── reminder_repo.go                // (80行)
│   ├── service/
│   │   ├── auth_service.go                 // 认证 (110行)
│   │   ├── plant_service.go                // 植物业务 (170行)
│   │   ├── care_service.go                 // 养护业务 (160行)
│   │   ├── reminder_service.go             // 提醒计算 (140行)
│   │   └── stats_service.go                // 统计 (120行)
│   ├── handler/
│   │   ├── auth_handler.go                 // 认证接口 (100行)
│   │   ├── plant_handler.go                // 植物接口 (160行)
│   │   ├── care_handler.go                 // 养护接口 (140行)
│   │   └── stats_handler.go                // 统计接口 (110行)
│   ├── middleware/
│   │   ├── auth.go                         // JWT验证 (70行)
│   │   └── cors.go                         // 跨域 (30行)
│   └── worker/
│       └── reminder_worker.go              // 每日提醒计算 (110行)
├── pkg/
│   ├── logger/
│   │   └── logger.go                       // 日志 (70行)
│   └── utils/
│       ├── time.go                         // 时间工具 (40行)
│       └── file.go                         // 文件处理 (50行)
├── api/
│   ├── router.go                           // 路由注册 (90行)
│   └── dto/
│       ├── plant_dto.go                    // (50行)
│       └── care_dto.go                     // (45行)
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_plants.sql
│   ├── 003_create_care_logs.sql
│   ├── 004_create_care_cycles.sql
│   └── 005_create_reminders.sql
├── frontend/
│   └── (React + Tailwind CSS)
├── go.mod
├── go.sum
└── docker-compose.yml
```

### 五、核心API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/register | 注册 |
| GET | /api/v1/plants | 植物列表 |
| POST | /api/v1/plants | 添加植物 |
| GET | /api/v1/plants/:id | 植物详情 |
| PUT | /api/v1/plants/:id | 更新植物 |
| DELETE | /api/v1/plants/:id | 删除植物 |
| GET | /api/v1/plants/:id/timeline | 生长时间线 |
| POST | /api/v1/plants/:id/care | 记录养护操作 |
| GET | /api/v1/today/reminders | 今日待办提醒 |
| GET | /api/v1/stats | 统计看板 |

### 六、数据库核心表

**plants（植物表）**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 所属用户 |
| name | VARCHAR(100) | 植物昵称 |
| species | VARCHAR(100) | 品种 |
| source | VARCHAR(50) | 入手来源 |
| acquired_date | DATE | 入手日期 |
| location | VARCHAR(50) | 位置 |
| status | VARCHAR(20) | healthy/yellowing/pests/gone/dead |
| difficulty | VARCHAR(20) | easy/medium/hard |
| avatar_url | VARCHAR(255) | 头像图 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**care_logs（养护记录表）**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| plant_id | UUID | 关联植物 |
| type | VARCHAR(20) | water/fertilizer/repot/prune/spray/clean |
| note | TEXT | 备注 |
| photo_url | VARCHAR(255) | 照片 |
| created_at | TIMESTAMP | 记录时间 |

**care_cycles（养护周期表）**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| plant_id | UUID | 关联植物 |
| type | VARCHAR(20) | water/fertilizer |
| interval_days | INT | 间隔天数 |
| last_date | DATE | 上次操作日期 |
| next_date | DATE | 下次操作日期 |

### 七、启动命令

```bash
# 构建
go build -o plant-diary ./cmd/server

# 运行
./plant-diary
```