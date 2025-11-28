package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// ----------------------
// UnifiedClient
// ----------------------
type UnifiedClient struct {
	mode string // "http", "ws", "sse"
	http *HTTPClient
	ws   *WSClient
	sse  *SSEClient
}

// NewUnifiedClientHTTP 创建 HTTP 方式的 MCP 客户端
func NewUnifiedClientHTTP(url string) *UnifiedClient {
	return &UnifiedClient{
		mode: "http",
		http: NewHTTPClient(url),
	}
}

// NewUnifiedClientWS 创建 WebSocket 方式的 MCP 客户端
func NewUnifiedClientWS(url string) (*UnifiedClient, error) {
	ws, err := NewWSClient(url)
	if err != nil {
		return nil, err
	}
	return &UnifiedClient{
		mode: "ws",
		ws:   ws,
	}, nil
}

// NewUnifiedClientSSE 创建 SSE 方式的 MCP 客户端
func NewUnifiedClientSSE(url string) *UnifiedClient {
	return &UnifiedClient{
		mode: "sse",
		sse:  NewSSEClient(url),
	}
}

// CallTool 调用工具
func (c *UnifiedClient) CallTool(ctx context.Context, toolName string, args interface{}, result interface{}) error {
	switch c.mode {
	case "http":
		return c.http.CallTool(ctx, toolName, args, result)
	case "ws":
		return c.ws.CallTool(ctx, toolName, args, result)
	case "sse":
		return fmt.Errorf("SSE client does not support RPC calls")
	default:
		return fmt.Errorf("unknown client mode")
	}
}

func (c *UnifiedClient) Call(ctx context.Context, method string, args interface{}, result interface{}) error {
	switch c.mode {
	case "http":
		return c.http.Call(ctx, method, args, result)
	case "ws":
		return c.ws.Call(ctx, method, args, result)
	case "sse":
		return fmt.Errorf("SSE client does not support RPC calls")
	default:
		return fmt.Errorf("unknown client mode")
	}
}

// ServerInfo 获取服务信息
func (c *UnifiedClient) ServerInfo(ctx context.Context) (*ServerInfoResp, error) {
	switch c.mode {
	case "http":
		var out ServerInfoResp
		err := c.http.Call(ctx, "server.info", map[string]any{}, &out)
		return &out, err
	case "ws":
		var out ServerInfoResp
		err := c.ws.Call(ctx, "server.info", map[string]any{}, &out)
		return &out, err
	case "sse":
		return nil, fmt.Errorf("SSE client does not support RPC calls")
	default:
		return nil, fmt.Errorf("unknown client mode")
	}
}

// ServerToolsList 获取服务工具列表
func (c *UnifiedClient) ServerToolsList(ctx context.Context) (*ServerListResp, error) {
	switch c.mode {
	case "http":
		var out ServerListResp
		err := c.http.Call(ctx, "tools.list", map[string]any{}, &out)
		return &out, err
	case "ws":
		var out ServerListResp
		err := c.ws.Call(ctx, "tools.list", map[string]any{}, &out)
		return &out, err
	case "sse":
		return nil, fmt.Errorf("SSE client does not support RPC calls")
	default:
		return nil, fmt.Errorf("unknown client mode")
	}
}

// WatchEvents 监听事件
func (c *UnifiedClient) WatchEvents(handler func(event string, data json.RawMessage)) error {
	switch c.mode {
	case "http":
		return fmt.Errorf("HTTP client does not support SSE")
	case "ws":
		return fmt.Errorf("WebSocket client does not support SSE")
	case "sse":
		return c.sse.ListenSSE(handler)
	default:
		return fmt.Errorf("unknown client mode")
	}
}

// Close 关闭客户端
func (c *UnifiedClient) Close() {
	switch c.mode {
	case "http":
		c.http.Close()
	case "ws":
		c.ws.Close()
	case "sse":
		c.sse.Close()
	}
}
