package kimi

import (
	"encoding/json"
)

type Option func([]string) []string

func WithExecutable(executable string) Option {
	return func(args []string) []string {
		args[0] = executable
		return nil
	}
}

func WithConfig(config *Config) Option {
	return func([]string) []string {
		// SAFETY: we guaranteed that the config is valid to be marshalled to JSON
		cfg, _ := json.Marshal(config)
		return []string{"--config", string(cfg)}
	}
}

func WithConfigFile(file string) Option {
	return func([]string) []string {
		return []string{"--config-file", file}
	}
}

func WithModel(model string) Option {
	return func([]string) []string {
		return []string{"--model", model}
	}
}

func WithWorkDir(dir string) Option {
	return func([]string) []string {
		return []string{"--work-dir", dir}
	}
}

func WithSession(session string) Option {
	return func([]string) []string {
		return []string{"--session", session}
	}
}

func WithMCPConfigFile(file string) Option {
	return func([]string) []string {
		return []string{"--mcp-config-file", file}
	}
}

func WithMCPConfig(config *MCPConfig) Option {
	return func([]string) []string {
		cfg, _ := json.Marshal(config)
		return []string{"--mcp-config", string(cfg)}
	}
}

func WithAutoApprove() Option {
	return func([]string) []string {
		return []string{"--auto-approve"}
	}
}

func WithThinking(thinking bool) Option {
	return func([]string) []string {
		if thinking {
			return []string{"--thinking"}
		} else {
			return []string{"--no-thinking"}
		}
	}
}

func WithSkillsDir(dir string) Option {
	return func([]string) []string {
		return []string{"--skills-dir", dir}
	}
}
