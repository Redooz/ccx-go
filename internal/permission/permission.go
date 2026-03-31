package permission

import (
	"path/filepath"
	"strings"
	"sync"
)

// Decision represents a permission decision.
type Decision int

const (
	Allow Decision = iota
	Deny
	Ask // prompt the user
)

// Rule defines a permission rule with glob pattern matching.
type Rule struct {
	ToolPattern string   `json:"tool"`
	Action      Decision `json:"action"`
}

// Manager handles permission checks for tool execution.
type Manager struct {
	mu    sync.RWMutex
	rules []Rule
	// Permanent allow decisions from "Always Allow"
	alwaysAllow map[string]bool
}

// NewManager creates a new permission manager with default rules.
func NewManager() *Manager {
	return &Manager{
		rules:       defaultRules(),
		alwaysAllow: make(map[string]bool),
	}
}

// Check determines whether a tool call should be allowed, denied, or prompted.
func (m *Manager) Check(toolName string) Decision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Always-allow overrides everything
	if m.alwaysAllow[toolName] {
		return Allow
	}

	// Check rules in order (first match wins)
	for _, r := range m.rules {
		matched, _ := filepath.Match(r.ToolPattern, toolName)
		if matched {
			return r.Action
		}
	}

	return Ask // default: ask user
}

// SetAlwaysAllow marks a tool as permanently allowed for this session.
func (m *Manager) SetAlwaysAllow(toolName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alwaysAllow[toolName] = true
}

// AddRule adds a permission rule.
func (m *Manager) AddRule(r Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, r)
}

// LoadRules replaces the current rules.
func (m *Manager) LoadRules(rules []Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = rules
}

func defaultRules() []Rule {
	return []Rule{
		// Read-only tools are always allowed
		{ToolPattern: "Read", Action: Allow},
		{ToolPattern: "Glob", Action: Allow},
		{ToolPattern: "Grep", Action: Allow},
		{ToolPattern: "WebFetch", Action: Allow},
		// Write tools need permission
		{ToolPattern: "Write", Action: Ask},
		{ToolPattern: "Edit", Action: Ask},
		{ToolPattern: "Bash", Action: Ask},
	}
}

// ParseDecision converts a string to a Decision.
func ParseDecision(s string) Decision {
	switch strings.ToLower(s) {
	case "allow", "y", "yes":
		return Allow
	case "deny", "n", "no":
		return Deny
	default:
		return Ask
	}
}

// String returns the string representation of a Decision.
func (d Decision) String() string {
	switch d {
	case Allow:
		return "allow"
	case Deny:
		return "deny"
	default:
		return "ask"
	}
}
