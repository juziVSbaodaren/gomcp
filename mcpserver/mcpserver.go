package mcpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// MCP 里常用的 method 示例
// Method 名称	说明
// "tools.run"	执行某个工具，参数包含 "name" 和 "arguments"
// "tools.list"	列出服务端注册的所有工具
// "server.info"	获取服务端信息（名称、版本、工具列表）
// "system.describe"	可选方法，一些 JSON-RPC 服务提供的自描述接口
// "system.listMethods"	列出服务端支持的所有方法
// "system.version"	获取服务端 JSON-RPC 版本

// Methods 全局方法开关表
// key: 方法名，如 "tools.run"
// value: 是否启用（true=启用，false=禁用）
var Methods = map[string]bool{
	"tools.run":          true,
	"tools.list":         true,
	"resources.get":      true,
	"resources.list":     true,
	"prompts.get":        true,
	"prompts.list":       true,
	"server.info":        true,
	"system.describe":    true,
	"system.listMethods": true,
	"system.version":     true,
}

// 检查方法是否启用
func IsMethodEnabled(method string) bool {
	enabled, ok := Methods[method]
	return ok && enabled
}

// 设置方法开关
func SetMethodEnabled(method string, enabled bool) {
	if _, ok := Methods[method]; ok {
		Methods[method] = enabled
	}
}

// 获取当前启用的 Method 列表
func ListEnabledMethods() []string {
	enabled := []string{}
	for method, ok := range Methods {
		if ok {
			enabled = append(enabled, method)
		}
	}
	return enabled
}

// ---------------------- JSON-RPC 基础结构 ----------------------
type RPCRequest struct {
	JsonRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type RPCResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      uint64      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// ---------------------- 工具参数结构 ----------------------
type GeocodeToolInput struct {
	Address string `json:"address"`
	City    string `json:"city,omitempty"`
}

type POISearchToolInput struct {
	Keywords string `json:"keywords"`
	City     string `json:"city,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type RouteToolInput struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Mode        string `json:"mode,omitempty"`
}

// ---------------------- 工具逻辑 ----------------------
func handleGeocode(input GeocodeToolInput) map[string]interface{} {
	return map[string]interface{}{
		"address": input.Address,
		"lat":     39.9042,
		"lng":     116.4074,
		"city":    input.City,
	}
}

func handlePOISearch(input POISearchToolInput) []map[string]interface{} {
	result := []map[string]interface{}{}
	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}

	for i := 0; i < limit; i++ {
		result = append(result, map[string]interface{}{
			"name": fmt.Sprintf("%s_POI_%d", input.Keywords, i+1),
			"lat":  39.90 + float64(i)*0.01,
			"lng":  116.40 + float64(i)*0.01,
			"city": input.City,
		})
	}
	return result
}

func handleRoute(input RouteToolInput) map[string]interface{} {
	return map[string]interface{}{
		"origin":      input.Origin,
		"destination": input.Destination,
		"mode":        input.Mode,
		"distance":    "10km",
		"duration":    "20min",
	}
}

// ---------------------- 工具列表 ----------------------
func listTools() interface{} {
	return map[string]interface{}{"tools": ListTools()}
}

// ---------------------- HTTP MCP Handler ----------------------
func httpHandler(w http.ResponseWriter, r *http.Request) {
	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	resp := RPCResponse{
		JsonRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {

	case "tools.list":
		resp.Result = listTools()

	case "tools.run":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			break
		}

		if result, err := CallToolByName(params.Name, params.Arguments); err != nil {
			resp.Error = &RPCError{Code: -32601, Message: err.Error()}
		} else {
			resp.Result = result
		}
		// resources
	case "resources.get":
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			break
		}
		if r, err := GetResource(params.Name); err != nil {
			resp.Error = &RPCError{Code: -32601, Message: err.Error()}
		} else {
			resp.Result = r
		}
	case "resources.list":
		resp.Result = map[string]interface{}{"resources": ListResources()}

	// prompts
	case "prompts.get":
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
			break
		}
		if p, err := GetPrompt(params.Name); err != nil {
			resp.Error = &RPCError{Code: -32601, Message: err.Error()}
		} else {
			resp.Result = p
		}
	case "prompts.list":
		resp.Result = map[string]interface{}{"prompts": ListPrompts()}
	case "server.info":
		resp.Result = map[string]interface{}{
			"name":    "MCP Server",
			"version": "1.0.0",
			"tools":   ListTools(),
		}

	case "system.describe":
		resp.Result = map[string]interface{}{
			"description": "This is a JSON-RPC server for MCP.",
			"version":     "1.0.0",
			"methods":     ListEnabledMethods(),
		}

	case "system.listMethods":
		resp.Result = ListEnabledMethods()

	case "system.version":
		resp.Result = "2.0"

	default:
		resp.Error = &RPCError{Code: -32601, Message: "Method not found"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ---------------------- WebSocket MCP Handler ----------------------
var upgrader = websocket.Upgrader{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}
	defer conn.Close()

	done := make(chan struct{}) // 用于通知 goroutine 停止
	// 启动心跳 goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					log.Println("Ping error, closing:", err)
					conn.Close()
					return
				}
			}
		}
	}()
	for {
		var req RPCRequest
		if err := conn.ReadJSON(&req); err != nil {
			// 非主动关闭连接
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("WS read error:", err)
			}

			break
		}

		resp := RPCResponse{
			JsonRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {

		case "tools.list":
			resp.Result = listTools()

		case "tools.run":
			var params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			json.Unmarshal(req.Params, &params)

			if result, err := CallToolByName(params.Name, params.Arguments); err != nil {
				resp.Error = &RPCError{Code: -32601, Message: err.Error()}
			} else {
				resp.Result = result
			}
			// resources
		case "resources.get":
			var params struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
				break
			}
			if r, err := GetResource(params.Name); err != nil {
				resp.Error = &RPCError{Code: -32601, Message: err.Error()}
			} else {
				resp.Result = r
			}
		case "resources.list":
			resp.Result = map[string]interface{}{"resources": ListResources()}

		// prompts
		case "prompts.get":
			var params struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
				break
			}
			if p, err := GetPrompt(params.Name); err != nil {
				resp.Error = &RPCError{Code: -32601, Message: err.Error()}
			} else {
				resp.Result = p
			}
		case "prompts.list":
			resp.Result = map[string]interface{}{"prompts": ListPrompts()}

		case "server.info":
			resp.Result = map[string]interface{}{
				"name":    "MCP Server",
				"version": "1.0.0",
				"tools":   ListTools(),
			}

		case "system.describe":
			resp.Result = map[string]interface{}{
				"description": "This is a JSON-RPC server for MCP.",
				"version":     "1.0.0",
				"methods":     ListEnabledMethods(),
			}

		case "system.listMethods":
			resp.Result = ListEnabledMethods()

		case "system.version":
			resp.Result = "2.0"
		default:
			resp.Error = &RPCError{Code: -32601, Message: "Method not found"}
		}

		if err := conn.WriteJSON(resp); err != nil {
			log.Println("WS write error:", err)
			return
		}
	}
	close(done)
}

// ---------------------- SSE Handler（Optional） ----------------------
type SSEClient struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

var sseClients = make(map[*SSEClient]struct{})

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher := w.(http.Flusher)
	client := &SSEClient{writer: w, flusher: flusher}

	sseClients[client] = struct{}{}
	defer delete(sseClients, client)

	notify := w.(http.CloseNotifier).CloseNotify()
	<-notify
}

func broadcastSSE(event string, data interface{}) {
	payload, _ := json.Marshal(data)
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, payload)
	for client := range sseClients {
		client.writer.Write([]byte(msg))
		client.flusher.Flush()
	}
}

type McpConf struct {
	Addr string `yaml:"addr" default:"localhost"`
	Port int    `yaml:"port" default:"8074"`
}

type McpServer struct {
	conf McpConf
}

func NewMcpServer(conf McpConf) *McpServer {
	return &McpServer{conf: conf}
}

func (s *McpServer) Start() {
	http.HandleFunc("/mcp", httpHandler)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/sse", sseHandler)

	// 定时 SSE 事件
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		count := 0
		for range ticker.C {
			count++
			broadcastSSE("update", map[string]interface{}{
				"message": fmt.Sprintf("Event #%d", count),
			})
		}
	}()
	fmt.Printf("✅ MCP Server running at: http://%s:%d\n", s.conf.Addr, s.conf.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", s.conf.Addr, s.conf.Port), nil))

}

// ---------------------- 启动 Server ----------------------
func StartMcpServer() {

	// 注册工具
	testTools()

	// 启动服务
	mcp := NewMcpServer(McpConf{
		Addr: "localhost",
		Port: 8074,
	})
	mcp.Start()
}
