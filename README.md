# Bamboo Agent

基于 [BambooMessages SDK](https://github.com/bamboo-services/bamboo-messages) 的 Go 语言 AI Agent 框架。

为 Go 开发者提供构建 AI Agent 应用的核心能力：**工具调用、自我迭代、多 Agent 协作和 MCP 集成**。

## 特性

- **ReAct 循环** — 内置 Reason-Act 迭代策略，支持自定义 LoopStrategy
- **并发工具执行** — AI 返回多个 tool_use 时，自动 goroutine 并行执行
- **上下文压缩** — 长对话自动压缩，避免超出模型 token 限制
- **多 Agent 编排** — 依赖图分析，独立任务并行，依赖任务串行
- **MCP Client** — JSON-RPC 2.0 over HTTP，自动发现并桥接外部工具
- **Functional Options** — 灵活的配置模式
- **内置工具** — 文件读写/搜索、Shell 执行、HTTP 请求、代码执行

## 架构

```
┌─────────────────────────────────────────────┐
│              Orchestrator (编排层)            │
│   Task / Channel / AgentBuilder / Dep Graph  │
├─────────────────────────────────────────────┤
│               Agent Core (核心层)            │
│   Agent / Session / ReActLoop / Compressor   │
├─────────────────────────────────────────────┤
│             Tool System (工具层)             │
│   Registry / Executor / Adapter / Builtins   │
├─────────────────────────────────────────────┤
│              MCP (扩展层)                     │
│        Client / Bridge / Config              │
└─────────────────────────────────────────────┘
```

## 快速开始

### 安装

```bash
go get github.com/bamboo-services/bamboo-agent
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"

    "github.com/bamboo-services/bamboo-agent/agent"
    "github.com/bamboo-services/bamboo-agent/orchestrator"
    "github.com/bamboo-services/bamboo-agent/tool/builtin"
    bamboo "github.com/bamboo-services/bamboo-messages/bamboo"
)

func main() {
    ctx := context.Background()

    // 1. 创建 BM-SDK 客户端（需配合 bamboo-messages 使用）
    var bmClient bamboo.BambooClient // = bamboo.NewClient(provider)

    // 2. 使用 Builder 构建 Agent
    ag := orchestrator.NewAgentBuilder().
        WithClient(bmClient).
        WithSystemPrompt("你是一个有帮助的助手").
        WithConfig(agent.AgentConfig{
            MaxTokens:          4096,
            MaxIterations:      10,
            MaxConcurrentTools: 10,
        }).
        WithTools(
            &builtin.FileReadTool{},
            &builtin.ShellTool{},
            &builtin.HTTPTool{},
        ).
        Build()

    // 3. 运行 Agent
    result, err := ag.Run(ctx, "帮我读取 config.json 并分析内容")
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Content)
}
```

### Functional Options

```go
ag := agent.NewAgentWithOptions(bmClient,
    agent.WithSystemPrompt("你是代码助手"),
    agent.WithMaxIterations(20),
    agent.WithMaxTokens(8192),
    agent.WithTemperature(0.7),
    agent.WithTools(&builtin.FileReadTool{}, &builtin.ShellTool{}),
)
```

### 自定义工具

```go
type MyTool struct{}

func (t *MyTool) Info() tool.ToolInfo {
    return tool.ToolInfo{
        Name:        "my_tool",
        Description: "我的自定义工具",
        Parameters: tool.InputSchema{
            Type: "object",
            Properties: map[string]tool.PropertyDef{
                "query": {Type: "string", Description: "搜索关键词"},
            },
            Required: []string{"query"},
        },
    }
}

func (t *MyTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
    return tool.ToolResult{Content: "结果"}, nil
}

agent.AddTool(&MyTool{})
```

### 多 Agent 编排

```go
orch := orchestrator.NewOrchestrator()
orch.RegisterAgent("researcher", researchAgent)
orch.RegisterAgent("writer", writerAgent)

tasks := []orchestrator.Task{
    {ID: "research", Description: "调研主题", Agent: researchAgent, Input: "AI Agent 框架现状"},
    {ID: "write", Description: "撰写报告", Agent: writerAgent, Input: "根据调研写报告", DependsOn: []string{"research"}},
}

results, err := orch.Execute(ctx, tasks)
```

### MCP 集成

```go
mcpClient := mcp.NewClient(mcp.DefaultConfig("http://localhost:8080"))
mcpClient.Connect(ctx)

bridge := mcp.NewBridge(mcpClient)
mcpTools, _ := bridge.DiscoverAndConvert(ctx)

ag := orchestrator.NewAgentBuilder().
    WithClient(bmClient).
    WithTools(mcpTools...).
    Build()
```

## 项目结构

```
agent/             # Agent 核心层
  agent.go         # Agent 接口 + agentCore 实现
  config.go        # AgentConfig 配置
  option.go        # Functional Options
  session.go       # 内存会话管理
  loop.go          # LoopStrategy + ReActLoop
  event.go         # AgentEvent 事件类型
  result.go        # AgentResult + ToolCallRecord
  compressor.go    # ContextCompressor + SummaryCompressor

tool/              # 工具层
  tool.go          # Tool 接口 + ToolInfo + ToolResult
  schema.go        # InputSchema + PropertyDef
  registry.go      # Registry 工具注册表
  executor.go      # ToolExecutor 并发执行器
  adapter.go       # BambooAdapter (Tool → bamboo.Tool)
  builtin/         # 内置工具
    file.go        # FileReadTool / FileWriteTool / FileSearchTool
    shell.go       # ShellTool
    http.go        # HTTPTool
    code.go        # CodeExecTool

orchestrator/      # 编排层
  orchestrator.go  # Orchestrator 多 Agent 编排
  builder.go       # AgentBuilder 构建器
  task.go          # Task + TaskStatus
  channel.go       # Channel + AgentMessage

mcp/               # 扩展层
  config.go        # Config + MCPToolInfo + MCPToolResult
  client.go        # Client (JSON-RPC 2.0)
  bridge.go        # Bridge + mcpToolAdapter
```

## 内置工具

| 工具 | 名称 | 功能 |
|------|------|------|
| `FileReadTool` | `file_read` | 读取文件内容 |
| `FileWriteTool` | `file_write` | 写入文件 |
| `FileSearchTool` | `file_search` | 搜索文件内容（返回匹配行） |
| `ShellTool` | `shell` | 执行 Shell 命令（支持超时） |
| `HTTPTool` | `http_request` | HTTP GET/POST/PUT/DELETE 请求 |
| `CodeExecTool` | `code_exec` | 执行代码片段（Go / Python） |

## 设计原则

1. **接口最小化** — 每个核心组件定义最小接口，扩展通过组合实现
2. **值类型传递** — 消息和事件使用值类型，通过 channel 安全传递
3. **零外部耦合** — 内置工具不依赖具体框架，可独立使用
4. **Functional Options** — 配置通过 Options 模式注入
5. **BM-SDK 对接** — 所有 AI 交互通过 BambooClient 进行
6. **自动压缩** — 长对话自动压缩上下文，开发者无需手动管理
7. **工具并发** — 多工具调用时自动并行执行，最大化吞吐量

## 技术栈

- **语言**: Go 1.25
- **依赖**: [BambooMessages SDK](https://github.com/bamboo-services/bamboo-messages)
- **并发**: goroutine + sync.WaitGroup + channel

## License

[MIT](LICENSE)
