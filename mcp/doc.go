// Package mcp 提供 MCP（Model Context Protocol）扩展层支持。
//
// 该包实现了 JSON-RPC 2.0 over HTTP 的客户端，用于连接 MCP 服务器并发现外部工具。
// 通过 Bridge 桥接机制，可以将 MCP 服务器工具无缝集成到 agent 工具系统中。
//
// 核心组件：
//   - Client: JSON-RPC 2.0 over HTTP 客户端，提供连接、工具发现和工具调用功能
//   - Bridge: 桥接 MCP 服务器工具到 agent 工具系统，包含 mcpToolAdapter 适配器
//   - Config: MCP 客户端配置，包含服务器 URL、超时时间、HTTP 请求头等
//   - MCPToolInfo: 工具元数据结构，描述 MCP 服务器提供的工具信息
//   - MCPToolResult: 工具调用结果结构，包含内容列表和错误状态
//
// 典型使用流程：
//  1. 使用 DefaultConfig 或自定义 Config 创建客户端配置
//  2. 通过 NewClient 创建 Client 实例
//  3. 调用 Connect 连接到 MCP 服务器
//  4. 通过 Bridge.DiscoverAndConvert 发现并转换工具
//  5. 将转换后的 tool.Tool 接口注册到 agent 工具系统
package mcp