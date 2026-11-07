# UserPref MCP Server

一个轻量级的、基于 Go 和 SQLite 的用户偏好服务，通过标准 MCP（Streamable HTTP）协议为不同 AI Agent 提供统一的用户偏好访问能力。

**One User. One Preference Profile. Every AI Agent.**

## 特性

- 纯 Go 实现，SQLite 使用 `modernc.org/sqlite`，不依赖 CGO
- MCP Streamable HTTP（`POST /mcp`）
- Global / Project 两级 Scope，Project 覆盖 Global
- JSON Value，支持字符串、数字、布尔值、数组和对象
- 五个核心 Tools：`get_preference`、`list_preferences`、`set_preference`、`delete_preference`、`get_effective_preferences`
- 只读 Resources：`preferences://global` 与 `preferences://global/{keyword}`
- 默认只监听 localhost，支持优雅退出

## 构建

需要 Go 1.24+：

```bash
go build -o userpref-mcp ./cmd/userpref-mcp
```

## 运行

```bash
./userpref-mcp
```

默认配置：

| 配置项 | 默认值 |
|---|---|
| 数据库 | 用户主目录下的 `preferences.db`（如 `~/preferences.db`），直接存放，不创建子目录 |
| 监听地址 | `127.0.0.1` |
| 端口 | `8765` |
| 默认用户 | `default` |

启动后：

- MCP Endpoint：`http://127.0.0.1:8765/mcp`
- 健康检查：`http://127.0.0.1:8765/healthz`

### 自定义参数

```bash
./userpref-mcp \
  --db /tmp/userpref-dev/preferences.db \
  --host 127.0.0.1 \
  --port 8765 \
  --user-id default \
  --log-level info
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--db` | `~/preferences.db` | SQLite 数据库路径；用于开发调试时隔离真实数据 |
| `--host` | `127.0.0.1` | HTTP 监听地址 |
| `--port` | `8765` | HTTP 端口 |
| `--user-id` | `default` | 默认用户 ID |
| `--log-level` | `info` | `debug` / `info` / `warn` / `error` |

`--db` 指定的父目录必须已存在；程序会创建数据库文件，但不会自动创建目录。

## MCP Client 配置

### Claude Code / 通用 JSON 配置

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

### Codex (`~/.codex/config.toml`)

```toml
[mcp_servers.userpref]
url = "http://127.0.0.1:8765/mcp"
```

## Bootstrap Instruction

将以下内容加入 Agent 的全局配置（例如 `~/.claude/CLAUDE.md` 或 `~/.codex/AGENTS.md`），让 Agent 主动读取偏好：

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

## MCP Tools

| Tool | 说明 |
|---|---|
| `get_preference` | 获取单项偏好 |
| `list_preferences` | 查询偏好列表，支持 scope 与关键字过滤 |
| `set_preference` | 创建或更新偏好（Upsert） |
| `delete_preference` | 删除偏好 |
| `get_effective_preferences` | 获取 Global / Project 合并后的最终偏好 |

`keyword` 为大小写不敏感的子串匹配，作用于 Key、Description 和 Value 文本。例如 `language` 可以同时匹配 `coding.language` 和 `communication.preferred_language`。

示例（通过 MCP Client 调用）：

```json
{
  "key": "coding.language",
  "value": "Go",
  "scope": "global",
  "description": "Preferred programming language"
}
```

## 开发

```bash
go test ./...
go vet ./...
```

开发调试时建议使用独立数据库，避免污染真实偏好数据：

```bash
mkdir -p /tmp/userpref-dev
go run ./cmd/userpref-mcp --db /tmp/userpref-dev/preferences.db --log-level debug
```

## 发布

推送 `v*` 形式的 tag 会触发 GitHub Actions，构建以下 6 个产物并创建 Release：

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

```bash
git tag v0.1.0
git push origin v0.1.0
```
