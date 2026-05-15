package agent

import (
	"fmt"
	"reflect"
)

type Tool interface {
	Name() string
	Description() string
	Execute(params map[string]interface{}) (interface{}, error)
}

var registeredTools = make(map[string]Tool)

func RegisterTool(tool Tool) {
	registeredTools[tool.Name()] = tool
}

func GetTool(name string) (Tool, bool) {
	tool, exists := registeredTools[name]
	return tool, exists
}

func ExecuteTool(name string, params map[string]interface{}) (interface{}, error) {
	tool, exists := registeredTools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool.Execute(params)
}

type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func GetAvailableTools() []ToolInfo {
	var tools []ToolInfo
	for _, tool := range registeredTools {
		tools = append(tools, ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  getToolParams(tool),
		})
	}
	return tools
}

func getToolParams(tool Tool) map[string]interface{} {
	params := make(map[string]interface{})
	t := reflect.TypeOf(tool)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.Name == "Execute" {
			mt := method.Type
			if mt.NumIn() > 1 {
				paramType := mt.In(1)
				if paramType.Kind() == reflect.Map {
					params["type"] = "object"
					params["properties"] = make(map[string]string)
				}
			}
		}
	}
	return params
}