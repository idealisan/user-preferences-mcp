# UserPref MCP Server 技术设计说明书

**项目定位：** 供所有 AI Agent 共享的用户偏好服务
**主要语言：** Go
**数据库：** SQLite（`modernc.org/sqlite`，纯 Go，无 CGO）
**通信协议：** MCP over Streamable HTTP（`userpref-mcp`）或 MCP stdio（`userpref-mcp-stdio`），两个二进制功能完全一致

---

## 1. 背景与问题

旧版本提供 `get_preference`、`list_preferences`、`get_effective_preferences` 等工具，依赖 Agent 按 key、scope 或 keyword 主动查询。实际使用中，Agent 只能根据当前任务猜测关键词，无法保证覆盖用户**全量**偏好：没被查到的偏好就不会被遵守，服务因此失去意义。

## 2. 新设计原则

1. **整体读取优先：** 一次调用返回全部偏好描述，不允许任何形式的过滤、搜索或分页。
2. **先可用，后优化：** 全量读取的开销可以接受；正确性优先于性能。
3. **极简工具集：** 只保留四个 Tool，不提供 Resources。
4. **可选多 profile：** 默认 `default`，可选传 `profile` 隔离不同上下文的偏好；绝大多数场景不用传。
5. **纯文本描述：** 偏好就是自然语言描述，不再是 key/value/JSON/scope 结构。
6. **本地优先：** HTTP 版默认只监听 localhost，无认证；stdio 版不监听端口。
7. **双传输共用核心：** Tool / Service / Repository 只有一份实现，HTTP 与 stdio 只是入口不同。

## 3. 系统架构

```text
                    AI Agent
                   /        \
     Streamable HTTP        stdio (stdin/stdout)
     (userpref-mcp)         (userpref-mcp-stdio)
                   \        /
                 MCP Server (internal/mcp)
                     |
        list_profiles / get_all_preferences /
        replace_all_preferences / append_preferences
                     |
        Preference Service (internal/preference)
                     |-- profile 默认值与描述内容校验
                     |
        Repository (internal/storage)
                     |-- SQLite 事务读写
                     |
        preference_descriptions 表 (~/preferences.db)
```

- `internal/mcp`：注册 Tool、编写面向 Agent 的 instructions 和描述（两个入口共用）。
- `internal/preference`：业务逻辑，校验 `profile` 和描述文本；不依赖 SQL。
- `internal/storage`：SQLite 建表与读写；批量写入使用事务。
- `internal/config`：命令行参数（HTTP 与 stdio 各自解析，共用校验逻辑）。
- `cmd/userpref-mcp`：HTTP 服务、优雅退出。
- `cmd/userpref-mcp-stdio`：`server.Run(ctx, &mcp.StdioTransport{})`，随 Client 退出；stdout 只走 MCP 协议，日志写 stderr。

## 4. 数据模型

新版使用新表 `preference_descriptions`，不复用旧版 `preferences` 表，因此旧数据不会被读取或删除：

```sql
CREATE TABLE IF NOT EXISTS preference_descriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile TEXT NOT NULL DEFAULT 'default',
    description TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_preference_descriptions_profile
    ON preference_descriptions(profile, id);
```

- `description` 是一条完整的偏好描述（纯文本），按 `id` 升序返回，保持插入顺序。
- `profile` 用于隔离不同用户的偏好集合；空值统一归一为 `default`。
- profile 数量预期很少，因此 `list_profiles` 不分页。

## 5. MCP Tools

### 5.1 `list_profiles`

列出所有至少含一条描述的 profile 名称，按字母序返回。无参数。

```json
{ "profiles": ["default", "work"], "count": 2 }
```

用于让 Agent 在不向用户询问精确名称的情况下发现已有 profile；多于一个时应与用户确认。

### 5.2 `get_all_preferences`

读取指定 profile 的**全部**偏好描述。

输入：

```json
{ "profile": "default" }
```

`profile` 可省略，省略时使用服务器默认 profile（默认 `default`）。

返回：

```json
{
  "profile": "default",
  "preferences": ["用中文交流", "改完代码要跑测试"],
  "count": 2
}
```

空 profile 返回空数组和 `count: 0`。

### 5.3 `replace_all_preferences`

用给定列表**全量覆盖**指定 profile 的偏好，保留给定顺序。传空列表清空该 profile。

输入：

```json
{
  "profile": "default",
  "preferences": ["用中文交流", "改完代码要跑测试"]
}
```

返回：

```json
{ "profile": "default", "success": true, "count": 2 }
```

Agent 只在用户明确要求重写全部偏好时调用。

### 5.4 `append_preferences`

在指定 profile 现有列表末尾追加描述，不改变已有内容。

输入：

```json
{
  "profile": "default",
  "preferences": ["临时文件放在 .tmp 目录"]
}
```

返回：

```json
{ "profile": "default", "success": true, "count": 3, "added": 1 }
```

`count` 是追加后的总数。

### 5.5 参数校验

- `profile` 去除首尾空白；为空时使用默认值。
- 每条描述去除首尾空白；空白描述返回 `INVALID_ARGUMENT`。
- 数据库错误返回 `DATABASE_ERROR`。

## 6. MCP Instructions

MCP SDK 的 `Instructions` 在初始化阶段发送给客户端，是约束 Agent 行为的主要通道。新版 instructions 明确要求：

1. 任何任务开始前调用一次 `get_all_preferences`，读取完整列表，不搜索、不过滤、不猜测。
2. 遵循所有返回的偏好描述。
3. 用户提到其他 profile 时，先 `list_profiles` 发现名称，多于一个则与用户确认。
4. 只在用户明确要求时写入偏好。
5. 不虚构未存储的偏好。

README 中另提供可复制到 Agent 全局配置的 Bootstrap Instruction。

## 7. 两种传输

### 7.1 HTTP 版 `userpref-mcp`

- MCP Endpoint：`POST /mcp`（Streamable HTTP，由 SDK 处理 GET/POST/DELETE 和会话）
- 健康检查：`GET /healthz` → `{"status":"ok"}`
- 默认监听 `127.0.0.1:8765`
- 支持 SIGINT / SIGTERM 优雅退出
- 需要常驻后台进程，可被多个 Agent / 多客户端共享

### 7.2 stdio 版 `userpref-mcp-stdio`

- 由 MCP Client 以子进程方式拉起，通过 stdin/stdout 交换 newline-delimited JSON-RPC
- `server.Run(ctx, &mcp.StdioTransport{})` 阻塞直到 Client 关闭 stdin
- 不监听端口，无后台常驻进程，随 Client 退出而退出
- stdout 是协议通道，**只能写 MCP 消息**；所有日志必须写 stderr（实现中使用 `slog` + `os.Stderr`）
- 单客户端使用；Client 配置中应写二进制绝对路径
- 与 HTTP 版可共存，也可同时访问同一个 SQLite 文件（WAL 模式）

## 8. 命令行

HTTP 版支持全部参数：

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--db` | `~/preferences.db` | 数据库路径；父目录必须已存在 |
| `--host` | `127.0.0.1` | 监听地址（仅 HTTP 版） |
| `--port` | `8765` | 监听端口（仅 HTTP 版） |
| `--profile` | `default` | 默认 profile 名 |
| `--user-id` | — | `--profile` 的兼容别名 |
| `--log-level` | `info` | `debug` / `info` / `warn` / `error` |

stdio 版只支持 `--db`、`--profile`、`--user-id`、`--log-level`，不接受 `--host` / `--port`。

## 9. 并发与一致性

- SQLite 开启 WAL、`busy_timeout(5000)`、`synchronous(NORMAL)`。
- `replace_all` 和 `append` 在单个事务中完成，避免部分写入。
- 所有 SQL 使用参数化查询。
- 数据库初始化幂等。

## 10. 测试

- `internal/storage`：建表持久化、重开数据库后数据仍在、profile 隔离、覆盖与追加顺序。
- `internal/preference`：默认值归一化、空白描述拒绝。
- `internal/mcp`：通过真实 MCP Client（Streamable HTTP）验证 instructions、工具清单与 annotations、四个 Tool 的完整调用流程、profile 隔离与错误返回。
- `cmd/userpref-mcp-stdio`：编译真实二进制，用 `CommandTransport` 拉起子进程，验证工具清单和读写行为与 HTTP 版一致。

## 11. 明确不做

- keyword / scope / JSON value / MCP Resources
- 分页、排序参数、增量查询
- Web UI、CLI 管理工具、导入导出
- 认证、OAuth、云端部署

这些功能如未来确有需要再单独评估，不影响当前实现的简单性。
