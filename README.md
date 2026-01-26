# xftui

基于 Go 的终端 TUI 聊天应用，支持 OpenAI 协议（可自定义 Base URL / API Key / Model），提供流式回答、会话历史、新建对话、导出和可点击操作。

## 功能特性
- **OpenAI 兼容协议**：可配置 `base_url`、`api_key`、`model`
- **流式回答**：边生成边显示
- **会话历史**：可加载历史对话
- **删除历史**：支持删除指定对话（含确认弹窗）
- **导出对话**：导出为 Markdown 文件
- **可点击操作**：底部按钮区支持鼠标点击
- **快捷键支持**：提升操作效率

## 快速开始

### 环境要求
- Go 1.25+（以 `go.mod` 为准）

### 构建与运行
```bash
# 直接运行
go run ./cmd

# 构建二进制
go build -o xftui ./cmd
```

## 配置说明

可以在界面 `Settings` 中填写并保存配置；也可以直接编辑配置文件：

- 配置文件路径：`~/.xftui.json`
- 配置示例：
```json
{
  "base_url": "https://api.openai.com/v1",
  "api_key": "YOUR_API_KEY",
  "model": "gpt-3.5-turbo"
}
```

## 使用说明

### 常用操作
- **New**：新建对话
- **History**：查看历史对话
- **Settings**：配置 API 参数
- **Export**：导出当前对话（`chat_export_<timestamp>.md`）
- **Quit**：退出应用

### 快捷键
- `Ctrl+N`：新建对话
- `Ctrl+H`：历史记录
- `Ctrl+S`：设置
- `Ctrl+E`：导出
- `Esc`：退出

### 其他指令
- `/read <path>`：读取文件内容并作为消息注入对话

## GitHub Actions 自动发布

已配置 GitHub Actions 自动编译并发布 release：

### 方式一：Tag 触发（推荐）
```bash
git tag v0.1.0
git push origin v0.1.0
```

### 方式二：手动触发
在 GitHub Actions 的 `Release` 工作流中手动触发，并填写 Tag 名称（如 `v0.1.0`）。

构建产物会自动上传到 GitHub Release。
