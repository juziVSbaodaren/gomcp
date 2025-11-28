package mcpserver

import (
	"fmt"
	"sync"
)

// -------------------- Resource --------------------
type Resource struct {
	Name string
	Type string
	Data interface{}
}

var (
	resourceRegistry = make(map[string]*Resource)
	resourceLock     sync.RWMutex
)

func RegisterResource(r *Resource) {
	resourceLock.Lock()
	defer resourceLock.Unlock()
	resourceRegistry[r.Name] = r
}

func GetResource(name string) (*Resource, error) {
	resourceLock.RLock()
	defer resourceLock.RUnlock()
	if r, ok := resourceRegistry[name]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("resource not found: %s", name)
}

func ListResources() []map[string]string {
	resourceLock.RLock()
	defer resourceLock.RUnlock()
	list := []map[string]string{}
	for _, r := range resourceRegistry {
		list = append(list, map[string]string{
			"name": r.Name,
			"type": r.Type,
		})
	}
	return list
}

// ---------------------- resource ----------
func testResource() {
	r1 := &Resource{
		Name: "test1",
		Type: "string",
		Data: "hello world",
	}
	r2 := &Resource{
		Name: "test2",
		Type: "int",
		Data: 123,
	}
	RegisterResource(r1)
	RegisterResource(r2)
}
