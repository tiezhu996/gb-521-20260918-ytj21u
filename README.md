# VentLock 风网联锁推演台

面向矿井通风工程师、复核人员和管理员的离线网络建模、风机方案审批、确定性风量推演与联锁风险复核系统。系统不连接 PLC、风机、风门或其他现场控制设备，所有结果只作为决策支持证据，不能替代现场规程和人工批准。

## Docker 快速启动

```bash
cp .env.example .env
# 按部署环境修改 .env 中的密码和 JWT_SECRET
docker compose up -d --build
docker compose ps
```

访问地址：

- 前端工作台：<http://127.0.0.1:18521>
- 后端健康检查：<http://127.0.0.1:19521/healthz>
- 后端就绪检查：<http://127.0.0.1:19521/readyz>

初始账号的密码均为 `Ventilate!2026`：

| 角色 | 邮箱 | 主要权限 |
|---|---|---|
| 工程师 | `engineer@mine.local` | 维护网络、创建/提交方案、发起推演 |
| 复核员 | `reviewer@mine.local` | 批准或驳回方案、确认风险、查看审计 |
| 管理员 | `admin@mine.local` | 全部权限，包括归档方案 |

停止并删除本项目数据卷：

```bash
docker compose down -v --remove-orphans
```

## 主要功能

- 维护进风口、回风口、工作面、网络交点和有向巷道边，检测自环、孤立节点、边界缺失及不可达工作面。
- 风机方案固定执行 `draft -> pending_review -> approved -> archived`，驳回返回 `draft` 并保留原因；版本条件更新防止并发越级。
- 根据巷道阻力关系执行确定性迭代，保存输入快照、每轮最大残差、节点压力、边风量和历史运行，不使用随机数伪造结果。
- 计算风速超限、反向流、工作面需风缺口和关键路径中断四类规则证据，并要求复核员或管理员人工确认。
- JWT、RBAC、请求限流、request ID、结构化日志和不可变操作审计贯穿后端与前端权限表现。

## 技术栈

| 层级 | 技术 |
|---|---|
| 前端 | React 18、TypeScript、Vite、Ant Design、Zustand、React Router、Lucide Icons |
| 后端 | Go 1.22、Gin、GORM、JWT、bcrypt、slog |
| 正式数据库 | PostgreSQL 16 |
| 自包含验证 | GORM SQLite 内存数据库，仅用于测试和 `runtime_smoke.json` |
| 部署 | Docker Compose、Nginx 多阶段构建与 API 反向代理 |

## 项目结构

```text
.
├── backend/
│   ├── cmd/server/             # 服务入口与优雅停机
│   ├── internal/
│   │   ├── config/             # 配置、双数据库驱动、迁移与种子
│   │   ├── constants/          # 共享状态和规则编号
│   │   ├── model/              # GORM 实体与数据库约束
│   │   ├── dto/                # HTTP 输入和算法证据 DTO
│   │   ├── repository/         # 事务、条件更新和审计持久化
│   │   ├── service/            # 业务规则、状态机与求解器
│   │   ├── handler/            # HTTP 参数和响应适配
│   │   ├── middleware/         # request ID、日志、认证、RBAC、恢复、限流
│   │   └── router/             # 依赖组装与实体路由
│   └── pkg/api/                # 统一响应、分页和错误码
├── frontend/src/
│   ├── api/                    # 按实体拆分的真实 API 客户端
│   ├── stores/                 # 四个核心实体与认证 Zustand store
│   ├── types/                  # 前后端一致的领域类型
│   ├── components/common/      # 状态、证据和确认共享组件
│   ├── hooks/                  # useAuth、useSimulationPolling
│   ├── pages/                  # 网络、方案、推演、联锁、审计页面
│   ├── router/                 # 路由守卫
│   └── utils/                  # 格式化和统一错误展示
├── docker-compose.yml
├── go.work
├── runtime_smoke.json
└── output/                     # 实际验收报告与截图
```

## 本地开发

后端默认连接本机 PostgreSQL；快速开发可显式使用独立 SQLite 文件或内存库：

```bash
export DB_DRIVER=sqlite
export DB_DSN='file:local-dev.db?_foreign_keys=on'
export JWT_SECRET='local-development-secret-at-least-32-bytes'
export PORT=19521
go run ./backend/cmd/server
```

另开终端启动前端，Vite 会把 `/api` 代理到 `19521`：

```bash
npm --prefix frontend ci
npm --prefix frontend run dev
```

质量检查：

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
npm --prefix frontend test
npm --prefix frontend run build
```

## API 清单

所有业务 API 使用 `/api/v1`，除登录外均要求 Bearer Token。

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/v1/auth/login` | 登录，独立限流 |
| `GET` | `/api/v1/auth/me` | 当前身份与角色 |
| `GET/POST` | `/api/v1/nodes` | 节点列表与创建 |
| `GET/PUT` | `/api/v1/nodes/:id` | 节点详情与更新 |
| `GET` | `/api/v1/network/validate` | 有向网络校验 |
| `GET/POST` | `/api/v1/edges` | 巷道列表与创建 |
| `GET/PUT` | `/api/v1/edges/:id` | 巷道详情与乐观锁更新 |
| `GET/POST` | `/api/v1/scenarios` | 方案列表与草稿创建 |
| `POST` | `/api/v1/scenarios/:id/transition` | 提交、批准、驳回、归档 |
| `GET/POST` | `/api/v1/simulations` | 历史查询与批准方案推演，启动独立限流 |
| `GET` | `/api/v1/simulations/:id` | 完整结果和残差历史 |
| `POST` | `/api/v1/simulations/:id/confirm-risks` | 人工确认风险证据 |
| `GET` | `/api/v1/audits` | 按操作者、对象、状态和时间筛选审计 |

响应统一为 `{ data, request_id, meta? }` 或 `{ error: { code, message, details? }, request_id }`，时间使用 RFC 3339 UTC 字符串。

## 共享枚举出现位置

`ScenarioStatus = draft | pending_review | approved | archived`：

- 数据库约束与 model：`backend/internal/model/fan_scenario.go`
- 后端常量与状态机：`backend/internal/constants/scenario.go`
- DTO、repository、service、handler、router：`backend/internal/dto/fan_scenario.go`、`backend/internal/repository/fan_scenario.go`、`backend/internal/service/fan_scenario.go`、`backend/internal/handler/fan_scenario.go`、`backend/internal/router/fan_scenario.go`
- 前端类型、API、store、共享状态组件、页面：`frontend/src/types/scenario.ts`、`frontend/src/api/scenarios.ts`、`frontend/src/stores/scenarioStore.ts`、`frontend/src/components/common/StatusBadge.tsx`、`frontend/src/pages/ScenariosPage.tsx`

`SimulationStatus = queued | running | converged | not_converged | invalid_input | failed`：

- 数据库约束与 model：`backend/internal/model/simulation_run.go`
- 后端常量：`backend/internal/constants/simulation.go`
- DTO、repository、service、handler、router：`backend/internal/dto/simulation_run.go`、`backend/internal/repository/simulation_run.go`、`backend/internal/service/simulation_run.go`、`backend/internal/handler/simulation_run.go`、`backend/internal/router/simulation_run.go`
- 前端类型、API、store、轮询 hook、共享状态组件、页面：`frontend/src/types/simulation.ts`、`frontend/src/api/simulations.ts`、`frontend/src/stores/simulationStore.ts`、`frontend/src/hooks/useSimulationPolling.ts`、`frontend/src/components/common/StatusBadge.tsx`、`frontend/src/pages/SimulationsPage.tsx`

## 算法假设与安全边界

- 巷道按 `ΔP = R × Q²` 的方向关系计算风量，调节风门使用确定的阻力倍率；内部节点通过阻尼牛顿式压力修正迭代平衡。
- 进风和回风节点是压力边界；风机曲线使用分段线性插值；达到方案阈值为 `converged`，达到迭代上限为 `not_converged`，网络或曲线非法为 `invalid_input`。
- 计算是简化离线模型，没有替代经校核的专业仿真、瓦斯监测、现场巡检、应急预案或法定审批。
- 系统没有现场协议、控制寄存器、PLC 地址或设备下发接口；人工确认只形成审计事件。

## 环境变量与端口

| 变量 | 默认示例 | 说明 |
|---|---|---|
| `COMPOSE_PROJECT_NAME` | `mine-ventilation-network-simulator` | 固定英文 Compose 项目名 |
| `FRONTEND_PORT` | `18521` | 前端宿主端口 |
| `BACKEND_PORT` | `19521` | 后端宿主端口 |
| `DB_PORT` | `57521` | PostgreSQL 宿主端口 |
| `DB_NAME/DB_USER/DB_PASSWORD` | 见 `.env.example` | PostgreSQL 连接配置 |
| `JWT_SECRET` | 至少 32 字符 | JWT HMAC 密钥，部署前必须替换 |
| `CORS_ORIGINS` | 前端两个本地地址 | 逗号分隔的允许来源 |
| `LOG_LEVEL` | `info` | `debug/info/warn/error` |

数据库使用命名卷 `postgres_data`，不绑定中文宿主路径。服务间通过 Compose 服务名通信，前端代码只请求 `/api`，Nginx 使用无尾斜杠的 `proxy_pass http://backend:8080` 保留 `/api/v1` 路径。

## 故障排查

- `project name must not be empty`：确认根目录 `.env` 存在且 `COMPOSE_PROJECT_NAME` 未改为中文；Compose 文件也已提供固定 `name` 兜底。
- 后端未 healthy：执行 `docker compose logs backend`，重点检查 JWT 密钥长度、数据库密码和 PostgreSQL 健康状态。
- 页面登录后请求 401：清除当前标签页的 `sessionStorage` 后重新登录；令牌只保存在会话存储中。
- 方案不能推演：确认方案已由 `reviewer` 或 `admin` 迁移到 `approved`，工程师不能自行批准。
- 端口冲突：只能在确有冲突时同时修改 `.env` 与访问地址；项目规定端口用于批量验收时不要变更。

## License

MIT License。示例系统仅用于软件工程与离线模型演示，不构成矿山安全建议。
