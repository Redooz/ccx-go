package skill

import (
	"context"
	"fmt"
	"strings"
)

// ExecResult holds the result of executing a skill.
type ExecResult struct {
	Output  string
	IsError bool
}

// ExecuteFunc runs the skill's body as a prompt. The implementation
// would typically call the Claude API or spawn an agent.
type ExecuteFunc func(ctx context.Context, prompt string) (string, error)

// Execute runs a skill, either inline or by spawning an agent.
func Execute(ctx context.Context, skill *Skill, args string, run ExecuteFunc) (*ExecResult, error) {
	if skill == nil {
		return &ExecResult{Output: "skill not found", IsError: true}, nil
	}

	// Build the prompt from skill body + args
	prompt := skill.Body
	if args != "" {
		prompt = fmt.Sprintf("%s\n\nArguments: %s", prompt, args)
	}

	output, err := run(ctx, prompt)
	if err != nil {
		return &ExecResult{Output: fmt.Sprintf("skill execution error: %v", err), IsError: true}, nil
	}

	return &ExecResult{Output: output}, nil
}

// FindSkillByName finds a skill by name from a list.
func FindSkillByName(skills []*Skill, name string) *Skill {
	name = strings.ToLower(name)
	for _, s := range skills {
		if strings.ToLower(s.Name) == name {
			return s
		}
	}
	return nil
}

// FindSkillByTrigger finds skills whose trigger matches the given text.
func FindSkillByTrigger(skills []*Skill, text string) []*Skill {
	text = strings.ToLower(text)
	var matches []*Skill
	for _, s := range skills {
		if s.Trigger != "" && strings.Contains(text, strings.ToLower(s.Trigger)) {
			matches = append(matches, s)
		}
	}
	return matches
}
