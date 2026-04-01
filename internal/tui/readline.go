package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
)

// SlashCommands defines completable slash commands.
var SlashCommands = []struct {
	Name string
	Desc string
}{
	{"/help", "Show available commands"},
	{"/exit", "Quit the session"},
	{"/clear", "Clear the screen"},
	{"/cost", "Show token usage and cost"},
	{"/model", "Show current model"},
	{"/compact", "Compress conversation context"},
	{"/version", "Show version info"},
	{"/tools", "List available tools"},
}

// NewReadline creates a liner instance with tab completion and history.
func NewReadline() *liner.State {
	line := liner.NewLiner()
	line.SetCtrlCAborts(true)

	line.SetCompleter(func(line string) []string {
		if !strings.HasPrefix(line, "/") {
			return nil
		}
		var completions []string
		for _, cmd := range SlashCommands {
			if strings.HasPrefix(cmd.Name, line) {
				completions = append(completions, cmd.Name)
			}
		}
		return completions
	})

	if f, err := os.Open(historyPath()); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	return line
}

// SaveHistory persists command history to disk.
func SaveHistory(line *liner.State) {
	if f, err := os.Create(historyPath()); err == nil {
		line.WriteHistory(f)
		f.Close()
	}
}

func historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ccx_history")
}
