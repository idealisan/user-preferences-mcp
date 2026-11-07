# UserPref MCP Server
## 技术设计说明书（Technical Design Specification）

**项目名称：** UserPref MCP Server  
**项目定位：** Universal User Preference Service for AI Agents  
**主要语言：** Go  
**数据库：** SQLite（纯 Go 实现）  
**通信协议：** MCP over Streamable HTTP  
**文档版本：** v0.1  
**项目阶段：** MVP

---

# 1. 项目背景与目标

## 1.1 项目背景

目前，不同 AI Agent 通常通过各自的配置文件来了解用户的偏好和工作习惯，例如：

- AGENTS.md
- CLAUDE.md
- Cursor Rules
- 其他 Agent Instructions 文件

这些文件通常包含用户的全局偏好，例如：

- 用户习惯使用什么编程语言。
- 用户喜欢什么样的代码风格。
- 用户希望 Agent 如何组织代码。
- 用户习惯将临时文件放在哪里。
- 用户希望使用什么语言交流。
- 用户喜欢怎样的工作流程。
- 用户希望 Agent 如何处理不确定的问题。

当用户同时使用多个 AI Agent 时，这些信息往往需要在不同工具中重复维护。

当用户有多个项目时，同样的全局偏好也可能被重复写入多个项目的 AGENTS.md。

这造成了以下问题：

1. 用户需要重复维护相同的偏好。
2. 不同 Agent 的偏好配置容易出现不一致。
3. 修改用户偏好时，需要同步更新多个文件。
4. 不同项目之间无法自然共享用户偏好。
5. 不同 AI Agent 无法共享统一的用户上下文。

## 1.2 项目目标

UserPref MCP Server 的目标是建立一个统一的用户偏好存储和访问服务。

用户只需要维护一份全局偏好，所有支持 MCP 的 AI Agent 都可以访问和使用这些偏好。

核心理念：

**One User. One Preference Profile. Every AI Agent.**

用户偏好应该属于用户，而不是某一个 AI Agent 或某一个项目。

系统通过 MCP 协议向不同 AI Agent 提供统一的偏好查询和修改能力。

## 1.3 核心设计原则

1. **Agent Agnostic：** 不依赖任何特定 AI Agent。
2. **Lightweight：** 服务本身尽可能简单，不引入不必要的基础设施。
3. **Easy Deployment：** 使用 Go 实现，提供独立可执行文件。
4. **Local First：** 第一阶段优先支持单机部署。
5. **SQLite First：** 使用 SQLite 作为唯一的数据存储。
6. **Explicit Preference First：** 用户明确指定的偏好优先于 Agent 推断的偏好。
7. **Minimal Dependencies：** 不依赖外部数据库、Redis、消息队列或其他外部服务。
8. **MCP Native：** 使用标准 MCP 协议，而不是自行设计一套 Agent 专用 API。

---

# 2. MVP 功能范围

## 2.1 必须实现的功能

| 功能 | 说明 |
|---|---|
| MCP Server | 提供标准 MCP 服务 |
| HTTP Transport | 使用 MCP Streamable HTTP |
| SQLite Storage | 使用 SQLite 持久化偏好 |
| Preference CRUD | 创建、查询、修改、删除偏好 |
| Global Scope | 支持用户全局偏好 |
| Project Scope | 支持项目级偏好 |
| Preference Resolver | 合并全局和项目级偏好 |
| User ID | 支持用户 ID 隔离 |
| JSON Value | 支持字符串、数字、布尔值、数组和对象 |
| Resource | 提供用户偏好的 MCP Resource |
| Standalone Binary | 编译为独立 Go 可执行文件 |

## 2.2 MVP 暂不实现的功能

以下功能明确不属于第一阶段：

- Web 管理界面。
- 用户认证系统。
- OAuth。
- API Key 管理。
- 数据加密。
- PostgreSQL。
- Redis。
- 外部服务集成。
- Agent 自动学习和推断用户偏好。
- Agent Scope。
- 偏好自动同步。
- 复杂的版本管理。
- Markdown / YAML 导入导出。
- JSON 导入导出。
- 多租户 SaaS 管理功能。
- 实时推送和通知。

SQLite 数据库文件本身就是系统的持久化数据载体。用户可以通过 SQLite 数据库管理工具直接查看和管理数据。

不需要额外实现导入导出功能。

---

# 3. 技术选型

## 3.1 编程语言

使用 Go。

主要考虑：

- 容易编译和部署。
- 可以提供单一可执行文件。
- 对 HTTP 服务支持良好。
- 适合实现轻量级 MCP Server。
- 具有良好的跨平台编译能力。

## 3.2 数据库

使用 SQLite。

SQLite Driver：

`modernc.org/sqlite`

要求：

- 使用纯 Go 实现。
- 不依赖 CGO。
- 不需要用户单独安装 SQLite。
- 支持 Windows、Linux 和 macOS。
- 使用标准 Go `database/sql` 接口。

数据库文件默认名称：

`preferences.db`

默认存放位置：

用户主目录（Home Directory）根目录，直接存放，不创建子目录。

例如：

```text
macOS / Linux:  ~/preferences.db
Windows:        C:\Users\<user>\preferences.db
```

实现时应通过 Go 标准库 `os.UserHomeDir()` 解析用户主目录，而不是硬编码平台路径。

用户可以通过启动参数 `--db` 指定其他数据库文件位置。

该参数主要用于开发调试场景，使开发者可以使用独立于本机实际使用数据库文件的临时数据库，避免污染真实用户偏好数据。

不带该参数时，始终使用用户主目录中的默认位置。

## 3.3 MCP SDK

使用 Go 生态中成熟、兼容标准 MCP 协议的 SDK。

优先考虑：

`github.com/modelcontextprotocol/go-sdk`

具体使用版本应在开发时选择稳定版本并固定依赖。

不建议自行实现 MCP JSON-RPC 协议。

## 3.4 HTTP Server

使用 Go 标准库：

`net/http`

不需要引入复杂的 Web Framework。

## 3.5 日志

使用 Go 标准库：

`log/slog`

日志应包括：

- 服务启动和停止。
- 数据库连接状态。
- MCP 请求错误。
- Tool 执行错误。
- 数据库操作错误。

默认不记录完整的用户偏好内容，避免不必要的信息泄露。

---

# 4. 系统架构

整体采用简单的分层架构。

```text
             AI Agents
                  |
                  |
             MCP Client
                  |
                  |
        MCP Streamable HTTP
                  |
                  |
       +---------------------+
       |                     |
       |  UserPref MCP       |
       |       Server        |
       |                     |
       +---------------------+
                  |
         +--------+--------+
         |                 |
         |                 |
     MCP Tools         MCP Resources
         |                 |
         +--------+--------+
                  |
                  |
         Preference Service
                  |
         +--------+--------+
         |                 |
         |                 |
    Preference CRUD   Preference Resolver
         |                 |
         +--------+--------+
                  |
                  |
             Repository
                  |
                  |
               SQLite
                  |
                  |
     ~/preferences.db (default)
```

## 4.1 模块职责

### MCP Server

负责：

- 初始化 MCP Server。
- 注册 MCP Tools。
- 注册 MCP Resources。
- 处理 MCP HTTP 请求。
- 将请求转发给 Preference Service。

### Preference Service

负责：

- 创建偏好。
- 获取偏好。
- 修改偏好。
- 删除偏好。
- 查询偏好列表。
- 调用 Preference Resolver。

### Preference Resolver

负责：

- 获取全局偏好。
- 获取项目级偏好。
- 合并不同 Scope 的偏好。
- 处理优先级和覆盖关系。

### Repository

负责：

- SQLite 数据读写。
- SQL 查询。
- 数据库初始化。
- 数据库 Migration。

业务逻辑不应直接依赖 SQL 实现。

---

# 5. 数据模型设计

## 5.1 基本设计

MVP 使用一张核心数据表：

`preferences`

采用 Key-Value 结构，每条记录代表一项用户偏好。

Value 使用 JSON 格式存储。

这样可以同时支持简单偏好和复杂结构化偏好。

例如：

简单偏好：

```json
{
  "key": "coding.language",
  "value": "Go"
}
```

复杂偏好：

```json
{
  "key": "filesystem.temporary_files",
  "value": {
    "location": "project_tmp",
    "directory": ".tmp",
    "cleanup": true
  }
}
```

## 5.2 SQLite Schema

建议使用以下 SQL：

```sql
CREATE TABLE IF NOT EXISTS preferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    user_id TEXT NOT NULL DEFAULT 'default',

    scope TEXT NOT NULL DEFAULT 'global',

    scope_id TEXT NOT NULL DEFAULT '',

    key TEXT NOT NULL,

    value TEXT NOT NULL,

    description TEXT NOT NULL DEFAULT '',

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (scope IN ('global', 'project')),

    CHECK (
        (scope = 'global' AND scope_id = '')
        OR
        (scope = 'project' AND scope_id != '')
    ),

    UNIQUE(user_id, scope, scope_id, key)
);

CREATE INDEX IF NOT EXISTS idx_preferences_lookup
ON preferences(user_id, scope, scope_id);

CREATE INDEX IF NOT EXISTS idx_preferences_key
ON preferences(user_id, key);
```

## 5.3 字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| id | INTEGER | 主键 |
| user_id | TEXT | 用户标识 |
| scope | TEXT | global / project |
| scope_id | TEXT | Scope 对应的 ID |
| key | TEXT | 偏好 Key |
| value | TEXT | JSON 格式的偏好值 |
| description | TEXT | 偏好的说明 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 修改时间 |

## 5.4 User ID

第一阶段不需要用户认证。

但是数据模型应当保留 user_id。

默认值：

`default`

启动服务时可以指定默认用户：

```bash
userpref-mcp --user-id default
```

也允许不同 MCP 请求明确指定 user_id，以支持单机上的多个用户配置。

需要注意：

MVP 的 user_id 只是数据隔离标识，不是身份认证机制。

单机部署时，服务默认只监听 localhost，不允许将其直接暴露到公网。

## 5.5 Scope 设计

第一阶段只支持两种 Scope：

### Global Scope

表示用户的全局偏好。

例如：

```text
scope = global
scope_id = ""
```

### Project Scope

表示某一个项目特有的偏好。

例如：

```text
scope = project
scope_id = my-project
```

项目 ID 由 Agent 在调用 MCP Tool 时明确传入。

MCP Server 不负责访问 Agent 所在机器的文件系统，也不负责自动判断当前项目目录。

项目 ID 应由用户或 Agent 配置提供。

## 5.6 Key 命名规则

使用层级命名。

推荐：

```text
general.language
general.response_style

coding.language
coding.style
coding.comments_language

workflow.discuss_before_implementation
workflow.run_tests_after_changes

filesystem.temporary_files
filesystem.default_editor

communication.explanation_detail
communication.preferred_language
```

建议使用小写英文和点号分隔。

Key 应具有明确含义，避免重复和歧义。

## 5.7 Value 格式

Value 使用合法 JSON。

支持：

- String
- Number
- Boolean
- Array
- Object
- Null（如确有需要）

Go 实现时应使用 JSON 校验，禁止写入非法 JSON。

---

# 6. Preference Scope Resolver

Resolver 是系统的重要业务逻辑。

## 6.1 优先级

MVP 采用：

```text
Project Scope
      |
      v
Global Scope
```

项目级偏好覆盖全局偏好。

例如：

Global：

```json
{
  "coding.language": "Go",
  "coding.style": "simple",
  "coding.comments_language": "English"
}
```

Project：

```json
{
  "coding.language": "Python"
}
```

最终有效偏好：

```json
{
  "coding.language": "Python",
  "coding.style": "simple",
  "coding.comments_language": "English"
}
```

## 6.2 合并规则

第一版采用简单的 Key 级覆盖规则：

1. 首先加载 Global Scope。
2. 然后加载指定 Project Scope。
3. 如果 Project 中存在相同 Key，则覆盖 Global。
4. 如果 Project 中不存在，则继承 Global。
5. 不存在的 Key 不自动生成默认值。

注意：

第一版不对数组、对象进行递归合并。

如果项目级设置了某个对象 Value，则整个对象覆盖全局对应 Key 的 Value。

这样可以保证行为简单且可预测。

## 6.3 Resolver 返回结果

建议返回：

```json
{
  "preferences": {
    "coding.language": "Python",
    "coding.style": "simple"
  },
  "sources": {
    "coding.language": "project",
    "coding.style": "global"
  }
}
```

sources 用于说明每项偏好的实际来源。

---

# 7. MCP Tools 设计

MVP 提供五个核心 Tools。

## 7.1 get_preference

获取单项偏好。

输入：

```json
{
  "user_id": "default",
  "key": "coding.language",
  "scope": "global",
  "scope_id": ""
}
```

参数说明：

| 参数 | 必须 | 说明 |
|---|---|---|
| user_id | 否 | 用户 ID，默认 default |
| key | 是 | 偏好 Key |
| scope | 是 | global / project |
| scope_id | 否 | Project ID |

返回：

```json
{
  "key": "coding.language",
  "value": "Go",
  "description": "Preferred programming language"
}
```

如果记录不存在，返回明确的 Not Found 结果，不应返回虚构的默认偏好。

## 7.2 list_preferences

获取偏好列表。

输入：

```json
{
  "user_id": "default",
  "scope": "global",
  "scope_id": "",
  "keyword": "language"
}
```

参数说明：

| 参数 | 必须 | 说明 |
|---|---|---|
| user_id | 否 | 用户 ID |
| scope | 否 | Scope |
| scope_id | 否 | Project ID |
| keyword | 否 | 关键字过滤 |

keyword 采用大小写不敏感的关键字子串匹配，匹配范围包括 Key、Description 和 Value 的文本内容。

例如：

`language`

可以匹配：

```text
coding.language
communication.preferred_language
description 中包含 language 的偏好
value 文本中包含 language 的偏好
```

关键字匹配比前缀匹配有更高的命中率，也更符合用户和 Agent 的检索习惯。

返回：

```json
{
  "preferences": [
    {
      "key": "coding.language",
      "value": "Go"
    },
    {
      "key": "coding.style",
      "value": "simple"
    }
  ],
  "count": 2
}
```

## 7.3 set_preference

创建或修改偏好。

输入：

```json
{
  "user_id": "default",
  "key": "coding.language",
  "value": "Go",
  "scope": "global",
  "scope_id": "",
  "description": "Preferred programming language"
}
```

采用 Upsert 语义：

- 不存在时创建。
- 存在时更新。
- 更新时保留 created_at。
- 更新时刷新 updated_at。

返回：

```json
{
  "success": true,
  "key": "coding.language",
  "scope": "global"
}
```

所有写入操作必须进行 JSON 合法性校验。

## 7.4 delete_preference

删除指定偏好。

输入：

```json
{
  "user_id": "default",
  "key": "coding.language",
  "scope": "global",
  "scope_id": ""
}
```

返回：

```json
{
  "success": true,
  "deleted": true
}
```

如果记录不存在，返回明确的结果。

## 7.5 get_effective_preferences

获取经过 Scope 合并后的最终偏好。

这是 MVP 最重要的 Tool。

输入：

```json
{
  "user_id": "default",
  "project_id": "project-a",
  "keyword": "language"
}
```

参数说明：

| 参数 | 必须 | 说明 |
|---|---|---|
| user_id | 否 | 用户 ID |
| project_id | 否 | 当前项目 ID |
| keyword | 否 | 关键字过滤，匹配 Key、Description 和 Value 文本 |

如果不提供 project_id，则只返回全局偏好。

返回：

```json
{
  "preferences": {
    "coding.language": "Python",
    "coding.style": "simple"
  },
  "sources": {
    "coding.language": "project",
    "coding.style": "global"
  }
}
```

如果没有任何偏好，返回空对象。

不应自动生成或推断用户偏好。

---

# 8. MCP Resources 设计

除了 Tools，建议提供只读 MCP Resources。

Resources 主要用于让 Agent 直接读取用户偏好的上下文。

建议提供：

```text
preferences://global
preferences://global/language
preferences://global/workflow
preferences://global/filesystem
```

URI 路径部分为关键字过滤条件，匹配规则与 Tools 的 keyword 一致。

具体要求：

1. Resource 返回完整的 JSON 内容。
2. 支持按关键字访问。
3. 内容发生变化后，下次读取应该获得最新数据。
4. 不需要实现实时推送。
5. 不需要实现订阅和通知。
6. 不应在服务端缓存导致数据不一致。

第一版暂不要求提供动态 Project Resource。

项目级偏好通过 `get_effective_preferences` Tool 获取。

---

# 9. Agent Bootstrap 机制

这是整个项目实现目标的关键。

MCP Server 本身不会保证 Agent 自动读取偏好。

因此需要在不同 Agent 的全局配置中添加一段简短的 Bootstrap Instruction。

建议提供以下标准文本：

```markdown
# User Preferences

A centralized User Preference MCP Server is available.

Before performing tasks:

1. Retrieve relevant user preferences using
   get_effective_preferences.
2. Respect the user's explicitly configured preferences.
3. Apply project-specific preferences when available.
4. Do not invent missing preferences.
5. Do not modify global preferences unless explicitly
   requested by the user.

Use these preferences as part of your working context.
```

用户只需要配置一次。

不需要在每个项目的 AGENTS.md 中重复维护这些全局偏好。

项目自身特有的说明仍然保留在项目级 AGENTS.md 中。

## 9.1 偏好读取策略

Agent 应根据当前任务读取相关偏好。

例如：

| 任务 | 建议读取的关键字 |
|---|---|
| 编写代码 | coding、language、workflow |
| 创建文件 | filesystem |
| 编写文档 | communication、language |
| 执行项目任务 | workflow、project 相关偏好 |
| 一般对话 | general、communication |

对于复杂任务，可以不传 keyword，一次获取全部有效偏好。

---

# 10. HTTP 服务设计

## 10.1 Endpoint

MCP Endpoint：

```text
POST /mcp
```

实际 HTTP Transport 的 GET、POST、DELETE 等方法和会话处理，应遵循所选 MCP SDK 的 Streamable HTTP 实现，不自行简化或改写协议。

健康检查：

```text
GET /healthz
```

健康检查返回：

```json
{
  "status": "ok"
}
```

第一版不需要额外实现 REST CRUD API。

所有偏好管理通过 MCP Tools 完成。

## 10.2 HTTP Transport

使用标准 MCP Streamable HTTP。

要求：

- 遵循 MCP Streamable HTTP 协议。
- 支持标准 MCP 初始化流程。
- 支持 Tool 调用。
- 支持 Resource 读取。
- 不实现业务级 SSE 推送。
- Tool 结果采用完整响应。
- 不实现 WebSocket。
- 不实现后台任务和异步消息队列。

注意：Streamable HTTP 是协议 Transport。是否使用 SSE 响应，应由 SDK 按 MCP 规范处理，不要自行假设它等同于普通 JSON REST 请求。

## 10.3 默认监听地址

默认：

```text
127.0.0.1:8765
```

MVP 不提供认证。

因此默认禁止监听公网地址。

如果用户主动指定其他监听地址，应通过启动配置明确表达，并在文档中说明当前版本不具备访问控制能力。

---

# 11. 命令行设计

可执行文件名称：

`userpref-mcp`

## 11.1 启动服务

```bash
userpref-mcp
```

默认：

- DB：用户主目录下的 `preferences.db`（如 `~/preferences.db`），直接存放，不创建子目录
- Host：`127.0.0.1`
- Port：`8765`
- User ID：`default`

不带 `--db` 参数时，服务始终使用用户主目录中的默认数据库文件。

## 11.2 自定义参数

```bash
userpref-mcp \
  --db /tmp/userpref-dev/preferences.db \
  --host 127.0.0.1 \
  --port 8765 \
  --user-id default
```

`--db` 用于开发调试时指定独立的数据库文件，避免与本机实际使用的默认数据库混用。该路径可以指向任意可写位置，不要求位于用户主目录内。

建议支持：

| 参数 | 默认值 | 说明 |
|---|---|---|
| --db | `~/preferences.db` | SQLite 数据库路径；未指定时使用用户主目录默认位置 |
| --host | 127.0.0.1 | HTTP 监听地址 |
| --port | 8765 | HTTP 端口 |
| --user-id | default | 默认用户 ID |
| --log-level | info | 日志级别 |

支持 `--help`。

## 11.3 运行要求

- 自动创建不存在的数据库文件。
- 自动执行数据库初始化。
- 默认数据库直接位于用户主目录，不自动创建任何子目录。
- 使用 `--db` 指定路径时，若其父目录不存在，给出清晰错误（或按明确的配置策略创建目录），不静默失败。
- 支持 SIGINT 和 SIGTERM 优雅退出。
- 关闭 HTTP Server 和 SQLite 连接。

---

# 12. Go 项目目录结构

建议采用：

```text
userpref-mcp/
├── cmd/
│   └── userpref-mcp/
│       └── main.go
│
├── internal/
│   ├── mcp/
│   │   ├── server.go
│   │   ├── tools.go
│   │   └── resources.go
│   │
│   ├── preference/
│   │   ├── model.go
│   │   ├── service.go
│   │   └── resolver.go
│   │
│   ├── storage/
│   │   ├── database.go
│   │   ├── repository.go
│   │   ├── sqlite.go
│   │   └── migrations.go
│   │
│   └── config/
│       └── config.go
│
├── migrations/
│   └── 001_init.sql
│
├── docs/
│   └── architecture.md
│
├── go.mod
├── go.sum
└── README.md
```

不需要为了架构完整而增加大量空目录和抽象接口。

只在确实存在职责边界的地方进行模块拆分。

---

# 13. 数据库与并发要求

虽然 MVP 是单机服务，但仍应考虑基本的数据一致性。

要求：

1. 使用 SQLite 事务处理必要的写入操作。
2. 合理配置 SQLite Busy Timeout。
3. 根据实际并发需求配置连接池。
4. 使用 WAL 模式以改善读写并发。
5. 数据库初始化必须具备幂等性。
6. 所有 SQL 使用参数化查询。
7. 通过唯一约束避免重复记录。
8. 更新操作正确刷新 updated_at。

SQLite 应保持单文件存储的简单性，不引入额外数据库服务。

---

# 14. 错误处理

MCP Tools 应返回清晰、可理解的错误。

主要错误类型：

| 错误 | 说明 |
|---|---|
| INVALID_ARGUMENT | 输入参数错误 |
| INVALID_JSON | Value 不是合法 JSON |
| NOT_FOUND | 指定偏好不存在 |
| DATABASE_ERROR | 数据库操作失败 |
| INVALID_SCOPE | Scope 不合法 |
| INTERNAL_ERROR | 未预期的内部错误 |

错误信息应包含必要的上下文，但不返回数据库内部路径等不必要的敏感信息。

---

# 15. 测试要求

## 15.1 单元测试

至少覆盖：

- Preference JSON 验证。
- Key 验证。
- Global Scope 查询。
- Project Scope 查询。
- Global / Project 合并。
- Project 覆盖 Global。
- Keyword 关键字过滤（Key、Description、Value）。
- Upsert 行为。
- 删除行为。

## 15.2 集成测试

使用临时 SQLite 文件测试：

- 数据库初始化。
- 数据持久化。
- 多用户数据隔离。
- MCP Tool 调用。
- MCP Resource 读取。
- HTTP Transport。
- 服务重启后的数据恢复。

## 15.3 端到端测试

至少验证：

1. 启动 MCP Server。
2. MCP Client 成功连接。
3. 创建一项全局偏好。
4. 查询全局偏好。
5. 创建项目级偏好。
6. 查询 Effective Preferences。
7. 验证项目偏好覆盖全局偏好。
8. 删除偏好。
9. 重启 Server。
10. 验证数据仍然存在。

---

# 16. MVP 验收标准

项目完成时必须满足：

- [ ] 使用 Go 实现。
- [ ] SQLite 使用纯 Go Driver，不依赖 CGO。
- [ ] 能编译为独立可执行文件。
- [ ] 能自动创建 SQLite 数据库。
- [ ] 默认数据库文件位于用户主目录，且不创建子目录。
- [ ] 支持通过 `--db` 参数覆盖默认数据库路径，便于开发调试。
- [ ] 能通过标准 MCP Client 连接。
- [ ] 支持 Streamable HTTP。
- [ ] 实现五个核心 MCP Tools。
- [ ] 实现指定的 MCP Resources。
- [ ] 支持 Global Scope。
- [ ] 支持 Project Scope。
- [ ] 支持有效偏好合并。
- [ ] 支持 JSON Value。
- [ ] 支持 user_id 数据隔离。
- [ ] 默认只监听 localhost。
- [ ] 支持服务优雅退出。
- [ ] 通过核心单元测试和集成测试。
- [ ] 提供完整 README 和 MCP Client 配置示例。

---

# 17. 后续扩展方向（不属于 MVP）

在第一版完成并验证实际使用效果之后，可以考虑：

1. Agent Scope。
2. 多用户认证。
3. OAuth。
4. 云端部署。
5. PostgreSQL。
6. Preference History。
7. Preference Versioning。
8. Agent 自动提取偏好。
9. 偏好冲突检测。
10. 偏好分类 Schema。
11. CLI 管理工具。
12. Web 管理界面。

这些功能不应影响 MVP 的架构复杂度。

特别注意：第一阶段不要为了未来的云端部署引入过度抽象。

当前应以 SQLite 为唯一存储实现，保持清晰的模块边界即可。

---

# 18. 开发实施建议

建议按以下顺序完成开发。

### Phase 1：项目基础设施

- 初始化 Go Module。
- 引入 MCP Go SDK。
- 引入 modernc.org/sqlite。
- 实现命令行参数。
- 实现 SQLite 初始化。
- 实现 HTTP Server。

### Phase 2：Preference Core

- 实现 Preference Model。
- 实现 SQLite Repository。
- 实现 Preference Service。
- 实现 Global / Project Scope。
- 实现 Preference Resolver。
- 编写单元测试。

### Phase 3：MCP Integration

- 注册五个 MCP Tools。
- 注册 MCP Resources。
- 实现 Tool 参数验证。
- 实现错误处理。
- 实现标准 MCP HTTP Transport。

### Phase 4：Integration Testing

- 使用真实 MCP Client 测试。
- 验证偏好读写。
- 验证 Scope 合并。
- 验证数据库持久化。
- 验证多用户隔离。

### Phase 5：Release

- 提供 Windows、Linux、macOS 编译版本。
- 提供 README。
- 提供 Claude Code、Codex 等 MCP Client 配置示例。
- 提供 Bootstrap Instruction 示例。
- 发布 MVP。

---

# 19. 最终产品定义

UserPref MCP Server 是一个轻量级、基于 Go 和 SQLite 的用户偏好服务。

它通过标准 MCP 协议为 AI Agent 提供统一的用户偏好访问能力。

它不试图成为一个完整的 Agent Memory 平台，也不试图成为一个复杂的用户管理系统。

它只专注解决一个问题：

**让用户只需要维护一次自己的全局偏好，就能够让所有 AI Agent 共享这些偏好。**

用户负责定义偏好。

SQLite 负责持久化。

Go Server 负责提供服务。

MCP 负责连接不同 Agent。

Agent 负责在工作中遵循这些偏好。

这就是 UserPref MCP Server 的核心价值。
