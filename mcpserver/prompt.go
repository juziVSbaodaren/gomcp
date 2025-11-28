package mcpserver

import (
	"fmt"
	"sync"
)

// -------------------- Prompt --------------------
type Prompt struct {
	Name     string
	Template string
}

var (
	promptRegistry = make(map[string]*Prompt)
	promptLock     sync.RWMutex
)

func RegisterPrompt(p *Prompt) {
	promptLock.Lock()
	defer promptLock.Unlock()
	promptRegistry[p.Name] = p
}

func GetPrompt(name string) (*Prompt, error) {
	promptLock.RLock()
	defer promptLock.RUnlock()
	if p, ok := promptRegistry[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("prompt not found: %s", name)
}

func ListPrompts() []string {
	promptLock.RLock()
	defer promptLock.RUnlock()
	names := []string{}
	for n := range promptRegistry {
		names = append(names, n)
	}
	return names
}
