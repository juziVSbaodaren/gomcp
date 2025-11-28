package mcpserver

import (
	"encoding/json"
	"fmt"
)

// ---------------------- Tool 定义 ----------------------
type Tool struct {
	Name        string
	Description string
	Handler     func(args json.RawMessage) (interface{}, error)
}
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ---------------------- Tool Registry ----------------------
var toolRegistry = make(map[string]*Tool)

func RegisterTool(tool *Tool) {
	toolRegistry[tool.Name] = tool
}

func ListTools() []ToolSummary {
	list := []ToolSummary{}
	for _, t := range toolRegistry {
		list = append(list, ToolSummary{
			Name:        t.Name,
			Description: t.Description,
		})
	}
	return list
}

func CallToolByName(name string, args json.RawMessage) (interface{}, error) {
	if tool, ok := toolRegistry[name]; ok {
		return tool.Handler(args)
	}
	return nil, fmt.Errorf("tool not found: %s", name)
}

// ---------------------- 测试工具 ----------------------
func testTools() {
	RegisterTool(&Tool{
		Name:        "geocode",
		Description: "Convert address to coordinates",
		Handler: func(args json.RawMessage) (interface{}, error) {
			var input GeocodeToolInput
			if err := json.Unmarshal(args, &input); err != nil {
				return nil, err
			}
			return handleGeocode(input), nil
		},
	})

	RegisterTool(&Tool{
		Name:        "poi_search",
		Description: "Search POI by keyword",
		Handler: func(args json.RawMessage) (interface{}, error) {
			var input POISearchToolInput
			if err := json.Unmarshal(args, &input); err != nil {
				return nil, err
			}
			return handlePOISearch(input), nil
		},
	})

	RegisterTool(&Tool{
		Name:        "route",
		Description: "Route planning between two addresses",
		Handler: func(args json.RawMessage) (interface{}, error) {
			var input RouteToolInput
			if err := json.Unmarshal(args, &input); err != nil {
				return nil, err
			}
			return handleRoute(input), nil
		},
	})
}
