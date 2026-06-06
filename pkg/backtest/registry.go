package backtest

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Constructor builds a new strategy instance from a validated param map.
type Constructor func(params map[string]ParamValue) (Strategy, error)

// StrategyInfo exposes a strategy's name and parameter definitions (Requirement 2.5).
type StrategyInfo struct {
	Name   string      `json:"name"`
	Params []ParamSpec `json:"params"`
}

type regEntry struct {
	specs []ParamSpec
	ctor  Constructor
}

// Registry registers and looks up strategies by name (Requirement 2).
type Registry struct {
	mu      sync.RWMutex
	entries map[string]regEntry
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]regEntry)}
}

// Register registers a strategy (Requirements 2.1, 2.4, 2.6, 3.5).
//   - empty/whitespace name -> error (Requirement 2.6)
//   - duplicate name -> error, keeping the first registrant (Requirements 2.4, 3.5)
func (r *Registry) Register(name string, specs []ParamSpec, ctor Constructor) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("策略名称不能为空")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("策略名称已被占用: %q", name)
	}
	r.entries[name] = regEntry{specs: specs, ctor: ctor}
	return nil
}

// New looks up by name and constructs an instance with params (Requirements 2.2, 2.3).
func (r *Registry) New(name string, params map[string]ParamValue) (Strategy, error) {
	r.mu.RLock()
	entry, ok := r.entries[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("策略未注册: %q", name)
	}
	validated, err := ApplyDefaults(entry.specs, params)
	if err != nil {
		return nil, err
	}
	return entry.ctor(validated)
}

// Info returns a strategy's name and parameter definitions (Requirement 18.2).
func (r *Registry) Info(name string) (StrategyInfo, error) {
	r.mu.RLock()
	entry, ok := r.entries[name]
	r.mu.RUnlock()
	if !ok {
		return StrategyInfo{}, fmt.Errorf("策略未注册: %q", name)
	}
	return StrategyInfo{Name: name, Params: entry.specs}, nil
}

// List returns all registered strategies sorted by name (Requirement 2.5);
// an empty registry returns an empty (non-nil) list.
func (r *Registry) List() []StrategyInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]StrategyInfo, 0, len(r.entries))
	for name, entry := range r.entries {
		out = append(out, StrategyInfo{Name: name, Params: entry.specs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// NewBatch constructs multiple instances of the same strategy with different
// param sets (Requirements 4.3, 4.4). Each instance holds an independent copy
// of its params; any out-of-range set returns an error naming the offending
// combination index and does not produce that instance.
func (r *Registry) NewBatch(name string, paramSets []map[string]ParamValue) ([]Strategy, error) {
	out := make([]Strategy, 0, len(paramSets))
	for i, ps := range paramSets {
		// Deep-copy the input map so instances cannot share parameter state.
		cp := make(map[string]ParamValue, len(ps))
		for k, v := range ps {
			cp[k] = v
		}
		s, err := r.New(name, cp)
		if err != nil {
			return nil, fmt.Errorf("参数组合 #%d 非法: %w", i, err)
		}
		out = append(out, s)
	}
	return out, nil
}

// Default is the global registry; builtin strategies register here via init().
var Default = NewRegistry()
