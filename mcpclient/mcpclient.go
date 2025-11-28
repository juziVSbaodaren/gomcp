package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// ----------------------
// JSON-RPC 类型
// ----------------------
type rpcRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      uint64      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// ----------------------
// ServerInfo 类型
// ----------------------
type ServerInfoResp struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Tools   []struct {
		Name string `json:"name"`
	} `json:"tools"`
}

type ServerListResp struct {
	Tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tools"`
}

// ----------------------
// MCPClient 接口
// ----------------------
type MCPClient interface {
	CallTool(ctx context.Context, toolName string, args interface{}, result interface{}) error
	Call(ctx context.Context, method string, args interface{}, result interface{}) error
	ServerInfo(ctx context.Context) (*ServerInfoResp, error)
	WatchEvents(handler func(event string, data json.RawMessage)) error
	Close()
}

// ----------------------
// HTTPClient
// ----------------------
type HTTPClient struct {
	URL     string
	counter uint64
}

func NewHTTPClient(url string) *HTTPClient {
	return &HTTPClient{URL: url}
}

func (c *HTTPClient) Call(ctx context.Context, method string, args interface{}, result interface{}) error {
	reqID := atomic.AddUint64(&c.counter, 1)
	reqBody := rpcRequest{
		JsonRPC: "2.0",
		ID:      reqID,
		Method:  method, // "tools.run", "tools.list", "server.info" 等
		Params:  args,   // 如果是 tools.run，则传 map{name:"", arguments:...}
	}

	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("MCP Error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if result != nil {
		return json.Unmarshal(rpcResp.Result, result)
	}
	return nil
}

func (c *HTTPClient) CallTool(ctx context.Context, toolName string, args interface{}, result interface{}) error {
	return c.Call(ctx, "tools.run", map[string]interface{}{"name": toolName, "arguments": args}, result)
}

func (c *HTTPClient) ListenSSE(handler func(event string, data json.RawMessage)) error {
	return fmt.Errorf("HTTP client does not support SSE")
}

func (c *HTTPClient) Close() {}

// ----------------------
// WSClient
// ----------------------
type WSClient struct {
	URL     string
	conn    *websocket.Conn
	counter uint64
}

func NewWSClient(url string) (*WSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &WSClient{URL: url, conn: conn}, nil
}
func (c *WSClient) Call(ctx context.Context, method string, args interface{}, result interface{}) error {
	reqID := atomic.AddUint64(&c.counter, 1)
	req := rpcRequest{
		JsonRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  args,
	}

	if err := c.conn.WriteJSON(req); err != nil {
		return err
	}

	var rpcResp rpcResponse
	if err := c.conn.ReadJSON(&rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("MCP Error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if result != nil {
		return json.Unmarshal(rpcResp.Result, result)
	}
	return nil
}
func (c *WSClient) CallTool(ctx context.Context, toolName string, args interface{}, result interface{}) error {
	return c.Call(ctx, "tools.run", map[string]interface{}{"name": toolName, "arguments": args}, result)
}

func (c *WSClient) ListenSSE(handler func(event string, data json.RawMessage)) error {
	return fmt.Errorf("WebSocket client does not support SSE")
}

func (c *WSClient) Close() {
	if c.conn != nil {
		// 先发送 Close 帧，告诉服务器“我准备关闭了”。
		// 服务器收到 Close 帧，可以返回 CloseNormalClosure，不会报 1006 错误。
		// 然后再真正关闭 TCP 连接。
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
		c.conn.Close()
	}
}

// ----------------------
// SSEClient
// ----------------------
type SSEClient struct {
	URL string
}

func NewSSEClient(url string) *SSEClient {
	return &SSEClient{URL: url}
}
func (c *SSEClient) Call(ctx context.Context, method string, args interface{}, result interface{}) error {
	return fmt.Errorf("SSE client does not support RPC calls")
}
func (c *SSEClient) CallTool(ctx context.Context, toolName string, args interface{}, result interface{}) error {
	return c.Call(ctx, toolName, args, result)
}

func (c *SSEClient) ListenSSE(handler func(event string, data json.RawMessage)) error {
	req, _ := http.NewRequest("GET", c.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var eventName string
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("event: ")) {
			eventName = string(line[7:])
		} else if bytes.HasPrefix(line, []byte("data: ")) {
			data := line[6:]
			handler(eventName, data)
		}
	}
}

func (c *SSEClient) Close() {}
