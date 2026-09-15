# Chat-TUI 🚀

> 基于 Go 的极简终端 Agent 聊天应用。对话通过 [CloudWeGo Eino ADK](https://github.com/cloudwego/eino) 的 `ChatModelAgent` + `Runner` 流式执行，支持多种官方 ChatModel 提供商。

[![Go Version](https://img.shields.io/github/go-mod/go-version/evallife/chat-tui)](https://github.com/evallife/chat-tui)
[![Latest Release](https://img.shields.io/github/v/release/evallife/chat-tui)](https://github.com/evallife/chat-tui/releases)
[![License](https://img.shields.io/github/license/evallife/chat-tui)](LICENSE)

---

## ✨ 功能特性

- 🔌 **多提供商**：通过 Eino 官方 `eino-ext` ChatModel 接入 OpenAI 兼容接口、Ark（火山引擎）、Ollama、Claude（Anthropic）、Gemini（Google）、Qwen（DashScope）、DeepSeek。
- 🧠 **Eino Agent（P0 通用 Agent）**：`ChatModelAgent` + `Runner` 流式执行；默认编程助手 Instruction、可配置 `max_iterations`、`ExitTool`、工作区沙箱（`workspace_root`）、可关闭 `write_file` / `run_command`；工具含 time/cwd/list/read/write/http_get/run_command/`glob_files`/`search_text`/`make_directory`。详见 [`docs/requirements.md`](docs/requirements.md) 与 [`docs/prd.md`](docs/prd.md)。
- 🌊 **流式交互**：助手文本增量渲染；出现 tool-call / 多步状态时在聊天区提示。`Esc` 或底部 **Stop** 取消进行中的 run。
- 🛡️ **危险工具确认**：`write_file` / `run_command` 执行前弹出 Allow/Deny；拒绝后工具返回错误，Agent 不会改文件或跑命令。
- 💬 **多行输入**：输入框支持多行编辑，`Shift+Enter` 换行，`Enter` 发送。
- 📋 **安全粘贴**：支持括号粘贴（bracketed paste），多行粘贴不会被拆成多次发送。
- 🧾 **复制模式**：一键进入 Copy Mode，支持选中文本并复制到系统剪贴板。
- 🎨 **主题系统**：内置多种配色主题，可在 Settings 中一键切换（night / nord / gruvbox / solarized-dark / light）。
- 📂 **会话管理**：
  - **历史回溯**：自动保存对话，支持随时加载历史记录。
  - **安全删除**：支持删除历史会话，内置二次确认防止误操作。
  - **一键导出**：支持将对话导出为标准的 Markdown 格式。
- 🖥️ **现代 TUI**：
  - **鼠标支持**：底部操作栏支持鼠标点击触发。
  - **优雅渲染**：集成 Markdown 语法高亮，代码块阅读更舒适。
- ⌨️ **极客操作**：丰富的快捷键支持，完全脱离鼠标亦可高效运行。
- 🗂️ **文件注入**：通过 `/read` 指令快速读取本地文件内容发送给 AI。

---

## 🚀 快速开始

### 安装方式

#### 方式一：直接通过 Go 安装 (推荐)
```bash
go install github.com/evallife/chat-tui/cmd/chat-tui@latest
```
*注意：安装后的可执行文件名将默认为 `chat-tui`。*

#### 方式二：手动构建
```bash
# 克隆仓库
git clone https://github.com/evallife/chat-tui.git
cd chat-tui

# 编译并命名为 chat-tui
go build -o chat-tui ./cmd/chat-tui

# 运行
./chat-tui
```

### 环境要求
- **Go**: 1.25+

首次运行会在 `~/.chat-tui.json` 写入默认配置（`api_key` 为空，工作区为 `~/chat-tui-workspace`）并直接进入 TUI。在 **Settings**（`Ctrl+S`）里填凭据后 Save 即可，不必再重启。

若本机仍有旧文件 `~/.xftui.json` / `~/.xftui.db`，启动时会复制到新路径；新文件可用后删除旧文件。已填写的 `api_key`、会话历史会保留；空的 `workspace_root` 会写成 `~/chat-tui-workspace`。

---

## ⚙️ 配置说明

您可以在应用的 **Settings** 界面直接修改提供商、模型与凭据，配置将自动保存。

**手动配置路径**：`~/.chat-tui.json`（会话库：`~/.chat-tui.db`）

缺少 `provider` 时默认 `openai`，继续使用原有的 `base_url` / `api_key` / `model`。

```json
{
  "provider": "openai",
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4o-mini",
  "theme": "night",
  "region": "",
  "access_key": "",
  "secret_key": "",
  "max_iterations": 20,
  "workspace_root": "~/chat-tui-workspace",
  "agent_name": "chat-tui",
  "disable_write_file": false,
  "disable_run_command": false,
  "disable_default_instruction": false
}
```

Agent 相关字段也可在 Settings 中修改；`/agent` 查看运行参数与已启用工具。

### 支持的 provider

| provider | 说明 | 主要字段 |
| :--- | :--- | :--- |
| `openai`（默认） | OpenAI 及任意 OpenAI 兼容接口 | `api_key`, `base_url`, `model` |
| `ark` | 火山引擎 Ark / 豆包 | `api_key` 或 `access_key`+`secret_key`，可选 `region`、`base_url` |
| `ollama` | 本地 Ollama | `model`，`base_url` 默认 `http://localhost:11434` |
| `claude` | Anthropic Claude | `api_key`，可选 `base_url` / `region`（Bedrock） |
| `gemini` | Google Gemini | `api_key`，可选 `base_url` |
| `qwen` | 阿里云 DashScope | `api_key`，`base_url` 默认 compatible-mode |
| `deepseek` | DeepSeek | `api_key`，可选 `base_url` |

别名（载入时规范化）：`openai-compatible` → `openai`，`anthropic` → `claude`，`google` → `gemini`，`doubao`/`volcengine` → `ark`，`dashscope` → `qwen`。

---

## 🏗️ Agent 架构

```
TUI (tview)  ──messages (go-openai types in SQLite)──►  api.Client
                                                          │
                                                          ├─ NewChatModel(provider)
                                                          ├─ adk.NewChatModelAgent (tools: extraToolsFor + HITL)
                                                          └─ adk.NewRunner(EnableStreaming)
                                                                │
                                                                ▼
                                                          AgentEvent stream
                                                          (text deltas / tool status)
```

存储与 UI 仍使用 `openai.ChatCompletionMessage`；在 `internal/api` 边界转换成 Eino `schema.Message`。工具由 `extraToolsFor(cfg, confirmer)` 按策略、工作区沙箱与 HITL 确认组装；继续扩展只需追加 `tool.BaseTool`，无需改 Runner 路径。

---

## ⌨️ 快捷键指南

| 快捷键 | 功能描述 |
| :--- | :--- |
| `Ctrl + N` | **新建对话** (New Chat) |
| `Ctrl + H` | **历史记录** (History List) |
| `Ctrl + S` | **设置中心** (Settings：provider / model / credentials) |
| `Ctrl + E` | **导出对话** (Export Markdown) |
| `Ctrl + Shift + E` | **导出对话 (带文件名对话框)** |
| `Ctrl + B` | **侧边栏开关** |
| `Ctrl + Y` | **进入 Copy Mode** |
| `Ctrl + C` | **主界面弹出退出确认** (Copy Mode 中为复制) |
| `Esc` | **取消流式 Agent run**；无流时退出确认（Copy Mode 中返回） |
| `Enter` | **发送消息** (输入框内) |
| `Shift + Enter` | **输入换行** |

底部操作栏提供 **Stop** 按钮，与 `Esc` 一样取消进行中的流。

---

## 📋 Copy Mode（复制模式）

进入 Copy Mode 后可进行文本选择与复制：
- `Ctrl + Y` 进入复制模式
- 鼠标拖拽或 Shift+方向键选中文本
- `Ctrl + C` 复制选中内容（未选择则复制全部）
- `Esc` 返回聊天界面

---

## 📝 高级指令

在聊天输入框内输入：
- `/read <path>`：读取指定路径的文件内容并发送给 AI（例如：`/read ./cmd/chat-tui/main.go`）。
- `/tools`：列出当前启用的 Agent 工具。
- `/agent`：显示 Instruction 摘要、迭代上限、工作区、工具策略与 HITL 确认。
- `/config`：显示当前 provider / model / agent 配置。
- `/help`：指令与快捷键一览。

---

## 🛠️ 开发与发布

本仓库已配置 **GitHub Actions** 自动化工作流。

- **自动发布**：推送以 `v` 开头的标签（如 `git tag v0.1.1 && git push origin v0.1.1`）将自动触发多平台二进制构建。
- **构建产物**：涵盖 Windows (amd64)、Linux (amd64) 和 macOS (amd64)。

```bash
go build -o chat-tui ./cmd/chat-tui
go build ./...
```

---

## 🤝 贡献与支持

欢迎提交 Issue 或 Pull Request 来完善项目。

*Powered by [Eino](https://github.com/cloudwego/eino), [tview](https://github.com/rivo/tview) & [glamour](https://github.com/charmbracelet/glamour)*
