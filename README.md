# Chat-TUI 🚀

> 基于 Go 语言开发的极简主义、高性能终端聊天应用。支持 OpenAI 协议，让您在终端即可享受流畅的 AI 对话体验。

[![Go Version](https://img.shields.io/github/go-mod/go-version/evallife/chat-tui)](https://github.com/evallife/chat-tui)
[![Latest Release](https://img.shields.io/github/v/release/evallife/chat-tui)](https://github.com/evallife/chat-tui/releases)
[![License](https://img.shields.io/github/license/evallife/chat-tui)](LICENSE)

---

## ✨ 功能特性

- 🤖 **广泛兼容**：支持所有兼容 OpenAI 协议的 API（可自定义 Base URL / API Key / Model）。
- 🌊 **流式交互**：打字机般的流式回应体验，拒绝等待。
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

---

## ⚙️ 配置说明

您可以在应用的 **Settings** 界面直接修改配置，配置将自动保存。

**手动配置路径**：`~/.xftui.json` (默认配置文件名暂保持兼容)

```json
{
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4-turbo",
  "theme": "night"
}
```

---

## ⌨️ 快捷键指南

| 快捷键 | 功能描述 |
| :--- | :--- |
| `Ctrl + N` | **新建对话** (New Chat) |
| `Ctrl + H` | **历史记录** (History List) |
| `Ctrl + S` | **设置中心** (Settings) |
| `Ctrl + E` | **导出对话** (Export Markdown) |
| `Ctrl + Shift + E` | **导出对话 (带文件名对话框)** |
| `Ctrl + B` | **侧边栏开关** |
| `Ctrl + Y` | **进入 Copy Mode** |
| `Ctrl + C` | **主界面弹出退出确认** (Copy Mode 中为复制) |
| `Esc` | **退出应用** (在 Copy Mode 中返回) |
| `Enter` | **发送消息** (输入框内) |
| `Shift + Enter` | **输入换行** |

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

---

## 🛠️ 开发与发布

本仓库已配置 **GitHub Actions** 自动化工作流。

- **自动发布**：推送以 `v` 开头的标签（如 `git tag v0.1.1 && git push origin v0.1.1`）将自动触发多平台二进制构建。
- **构建产物**：涵盖 Windows (amd64)、Linux (amd64) 和 macOS (amd64)。

---

## 🤝 贡献与支持

欢迎提交 Issue 或 Pull Request 来完善项目。

*Powered by [tview](https://github.com/rivo/tview) & [glamour](https://github.com/charmbracelet/glamour)*
