# Chat-TUI — 通用 Agent 需求说明

| 字段 | 内容 |
|------|------|
| 产品 | chat-tui（Go / Eino ADK `ChatModelAgent` + `Runner`） |
| 文档版本 | 0.9.0-draft |
| 日期 | 2026-09-15 |
| 目标用户 | 在终端用多模型做编程与日常任务的开发者 |

## 1. 背景与现状

当前能力（截至 `main` / tools + TUI）：

- 多提供商 ChatModel（openai / ark / ollama / claude / gemini / qwen / deepseek）
- 流式 Agent 跑通；内置工具：时间、cwd、列目录、读/写文件、HTTP GET、跑命令
- TUI 展示 tool-call / 结果；`/tools`、`/help`；会话、主题、系统提示词管理

与「通用 coding / personal agent」仍有差距：缺默认人格与迭代上限、缺工作区沙箱、缺搜索/glob、危险工具不可关、无 Exit 收束、无轻量策略与可观测状态。

## 2. 产品目标

把 chat-tui 从「能调工具的聊天框」推进到 **可配置的通用终端 Agent**：

1. **可预期**：默认指令像编程助手；工具循环有上限；可主动 `exit`。
2. **可约束**：工作区根目录限制文件/命令；可关闭写文件与执行命令。
3. **可发现**：内置搜索与 glob；`/agent` 展示运行参数与工具集。
4. **可演进**：文档区分 P0 / P1；P1 不阻塞本轮交付。

非目标（本轮不做）：DeepAgent 多代理编排、完整 HITL 审批流、MCP 插件市场、长期向量记忆、云端同步。

## 3. 用户故事

| ID | 作为… | 我想… | 以便… |
|----|--------|--------|--------|
| US-1 | 开发者 | 不配 system prompt 也有合理 Agent 行为 | 开箱能改代码、查问题 |
| US-2 | 开发者 | 限制 Agent 只能动某个项目目录 | 避免误改家目录 |
| US-3 | 开发者 | 按模式关掉 `run_command` / `write_file` | 只读勘察更安全 |
| US-4 | 开发者 | 用 glob / 文本搜索找文件 | 少靠盲猜路径 |
| US-5 | 开发者 | 看到迭代上限与启用工具列表 | 调试 Agent 行为 |
| US-6 | 开发者 | Agent 在完成后调用 exit | 减少无意义空转 |

## 4. 功能需求（分级）

### P0 — 本轮必须

| ID | 需求 | 验收标准 |
|----|------|----------|
| R-01 | 默认 Agent Instruction | `systemPrompt` 为空时使用内置通用编程助手指令；非空时仍以用户为准 |
| R-02 | `MaxIterations` 可配置 | 配置项 `max_iterations`（默认 20，范围 1–100）；传入 `ChatModelAgentConfig` |
| R-03 | 启用 `ExitTool` | Agent 可调用 exit 结束；TUI 已有 exit status 展示即可 |
| R-04 | `workspace_root` | 文件类工具与 `run_command` 解析路径相对该根；禁止逃逸到根外（`..` / 绝对路径越界拒绝） |
| R-05 | 工具策略开关 | `enable_write_file`、`enable_run_command`（默认 true）；false 时不注册对应工具 |
| R-06 | 新工具 `glob_files` | 按 glob 列文件，有数量上限 |
| R-07 | 新工具 `search_text` | 在工作区内按子串/简单模式搜文本（可用 `rg` 若存在，否则 Go 回退），有结果上限 |
| R-08 | 新工具 `make_directory` | 在工作区内创建目录 |
| R-09 | `/agent` 命令 | 打印 instruction 摘要、max_iterations、workspace_root、已启用工具名 |
| R-10 | Settings / 配置持久化 | 上述字段写入 `~/.xftui.json`，Settings 可改 |
| R-11 | 文档 | 本文件 + `docs/prd.md`；README 功能列表同步 P0 |

### P1 — 后续（文档保留，本轮可不实现）

| ID | 需求 | 说明 |
|----|------|------|
| R-20 | 危险工具 HITL | **已实现**：`write_file` / `run_command` 执行前 TUI Allow/Deny |
| R-21 | 会话笔记记忆 | 跨会话短笔记注入 Instruction |
| R-22 | Checkpoint / Resume | Eino Store + CheckpointID |
| R-23 | DeepAgent / 子代理 | 调研 / 编码分工 |
| R-24 | MCP 工具桥 | 外部 MCP server 动态挂工具 |
| R-25 | Reasoning 折叠展示 | 已有 EventReasoning，UI 增强 |

## 5. 非功能需求

- 工具输出继续有大小/超时上限；搜索默认超时 ≤ 10s。
- 配置缺省向后兼容：旧 `~/.xftui.json` 无新字段时用默认值。
- 单测覆盖：路径沙箱、工具开关、默认 instruction 选择、glob/search 基本行为。
- 不引入需 Pro 的 Cloud Agent 依赖；实现落在本仓库 Go 代码。

## 6. 开放问题

- Windows 上 `rg` 缺失时 search 回退路径是否足够（P0：Go walk 回退）。
- `workspace_root` 为空时是否等于进程 cwd（P0：是）。
