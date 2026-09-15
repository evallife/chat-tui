package api

import "github.com/cloudwego/eino/components/tool"

// extraTools is the ReAct tool extension point for ChatModelAgent.
// Leave empty for chat-only; append tool.BaseTool implementations (InvokableTool
// or StreamableTool) to enable tool calling without changing the Runner path.
func extraTools() []tool.BaseTool {
	return nil
}
