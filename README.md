# UserPref MCP Server

一个轻量级的用户偏好服务：把用户的偏好以纯文本描述的形式集中存放，通过标准 MCP 提供给所有 AI Agent 使用。

**One User. One Preference Profile. Every AI Agent.**

## 两个版本，功能完全一致

| 二进制 | 传输方式 | 生命周期 | 适用场景 |
|---|---|---|---|
| `userpref-mcp` | Streamable HTTP（`POST /mcp`） | 需要常驻后台进程 | 多个 Agent / 多客户端共享 |
| `userpref-mcp-stdio` | MCP stdio（stdin/stdout） | 由 MCP Client 拉起，随 Client 退出 | 本地单客户端，不占用常驻进程 |

两个版本共用同一套 Tool、Service 和 SQLite 存储代码，因此**功能、参数、行为完全相同**，可以共存，也可以同时操作同一个数据库文件。

## 为什么这么设计

旧版本提供按 key / scope / keyword 查询的工具，Agent 必须自己猜测该查什么关键词，很容易漏掉没有命中的偏好，导致实际工作中没有遵守用户的完整要求。

现在只保留四个功能，偏好**永远整体读取、整体写入**：

- Agent 每次开始工作前一次性读取全部偏好描述，不存在"查漏"问题。
- 全量读取会带来一点额外开销，但先保证功能正确，再谈优化。

## 特性

- 纯 Go 实现，SQLite 使用 `modernc.org/sqlite`，不依赖 CGO
- 偏好是纯文本描述列表，按插入顺序保存
- 可选的 `profile` 参数，支持多个 profile；默认 `default`
- 只读 Resources、keyword 过滤、scope / JSON value 等概念已移除
- stdio 版 stdout 只走 MCP 协议，所有日志写入 stderr

## 构建

需要 Go 1.24+：

```bash
go build -o userpref-mcp ./cmd/userpref-mcp
go build -o userpref-mcp-stdio ./cmd/userpref-mcp-stdio
```

## 运行：HTTP 版

```bash
./userpref-mcp
```

默认配置：

| 配置项 | 默认值 |
|---|---|
| 数据库 | 用户主目录下的 `preferences.db`（如 `~/preferences.db`） |
| 监听地址 | `127.0.0.1` |
| 端口 | `8765` |
| 默认 profile | `default` |

启动后：

- MCP Endpoint：`http://127.0.0.1:8765/mcp`
- 健康检查：`http://127.0.0.1:8765/healthz`

```bash
./userpref-mcp \
  --db /tmp/userpref-dev/preferences.db \
  --host 127.0.0.1 \
  --port 8765 \
  --profile default \
  --log-level info
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--db` | `~/preferences.db` | SQLite 数据库路径；父目录必须已存在 |
| `--host` | `127.0.0.1` | HTTP 监听地址 |
| `--port` | `8765` | HTTP 端口 |
| `--profile` | `default` | 未显式传 `profile` 时使用的 profile 名 |
| `--user-id` | — | `--profile` 的兼容别名 |
| `--log-level` | `info` | `debug` / `info` / `warn` / `error` |

## 运行：stdio 版

不需要自己启动。在 MCP Client 配置里指定可执行文件路径，由 Client 按需拉起：

### Claude Code

```bash
claude mcp add userpref -- /path/to/userpref-mcp-stdio
```

需要隔离数据库时：

```bash
claude mcp add userpref -- /path/to/userpref-mcp-stdio --db /tmp/userpref-dev/preferences.db
```

### 通用 JSON 配置

```json
{
  "mcpServers": {
    "userpref": {
      "command": "/path/to/userpref-mcp-stdio",
      "args": ["--db", "/path/to/preferences.db"]
    }
  }
}
```

### Codex (`~/.codex/config.toml`)

```toml
[mcp_servers.userpref]
command = "/path/to/userpref-mcp-stdio"
args = ["--db", "/path/to/preferences.db"]
```

stdio 版支持的参数只有 `--db`、`--profile`、`--user-id`、`--log-level`；没有 `--host` / `--port`。建议使用二进制的绝对路径，因为 Client 可能设置自己的工作目录。Client 断开连接（关闭 stdin）后进程会自动退出。

### HTTP 版的 MCP Client 配置

```json
{
  "mcpServers": {
    "userpref": {
      "type": "http",
      "url": "http://127.0.0.1:8765/mcp"
    }
  }
}
```

```toml
[mcp_servers.userpref]
url = "http://127.0.0.1:8765/mcp"
```

## Bootstrap Instruction

将以下内容加入 Agent 的全局配置（例如 `~/.claude/CLAUDE.md` 或 `~/.codex/AGENTS.md`），让 Agent 每次工作前完整读取偏好：

```markdown
# User Preferences

A centralized User Preference MCP Server is available.

Before starting any task:

1. Call get_all_preferences once and read the ENTIRE list. Do not search,
   filter or guess which preferences are relevant.
2. Follow every returned preference description.
3. If the user mentions another profile, call list_profiles first and
   confirm which profile to use. Omitting profile uses "default".
4. Only change preferences when the user explicitly asks. Use
   append_preferences to add, replace_all_preferences to rewrite everything.
5. Do not invent preferences that are not stored here.

Use these preferences as part of your working context.
```

## MCP Tools（两个版本完全相同）

| Tool | 说明 |
|---|---|
| `list_profiles` | 列出所有已有 profile 的名称（无参数） |
| `get_all_preferences` | 读取指定 profile 的全部偏好描述（默认 `default`） |
| `replace_all_preferences` | 用给定列表全量覆盖指定 profile 的偏好 |
| `append_preferences` | 向指定 profile 末尾追加偏好描述 |

所有读写工具都有一个**可选**的 `profile` 参数，绝大多数情况下不需要传。

偏好是纯文本字符串，每条是一条完整的偏好描述，例如：

```json
{
  "preferences": [
    "用中文与用户交流",
    "修改代码后运行 go test ./...",
    "临时文件放在项目 .tmp 目录"
  ]
}
```

`replace_all_preferences` 传空列表会清空该 profile；追加时传入空字符串会被拒绝。

## 开发

```bash
go test ./...
go vet ./...
```

`cmd/userpref-mcp-stdio/main_test.go` 会编译真实的 stdio 二进制并通过 `CommandTransport` 调用，验证它与 HTTP 版行为一致。

开发调试时建议使用独立数据库，避免污染真实偏好数据：

```bash
mkdir -p /tmp/userpref-dev
go run ./cmd/userpref-mcp --db /tmp/userpref-dev/preferences.db --log-level debug
```

## 关于旧数据

新版使用 `preference_descriptions` 表，不会读取或删除旧版本的 `preferences` 表；旧的 key/value/scope 数据会原样保留在数据库文件中，但不会自动迁移到新的描述列表。如需沿用，可手动整理后通过 `replace_all_preferences` 写入。

## 发布

推送 `v*` 形式的 tag 会触发 GitHub Actions，构建 6 个平台 × 2 个二进制，共 12 个产物并创建 Release：

- `userpref-mcp_<os>_<arch>`（HTTP 版）
- `userpref-mcp-stdio_<os>_<arch>`（stdio 版）

平台：`linux/{amd64,arm64}`、`darwin/{amd64,arm64}`、`windows/{amd64,arm64}`。
