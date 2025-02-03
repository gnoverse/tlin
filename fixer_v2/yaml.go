package fixerv2

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type FixRule struct {
	Name        string `yaml:"name"`
	Pattern     string `yaml:"pattern"`
	Replacement string `yaml:"replacement"`
}

type RulesConfig struct {
	Rules []FixRule `yaml:"rules"`
}

func Load(path string) ([]FixRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg RulesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.Rules, nil
}

func Apply(subject string, rules []FixRule) string {
	result := subject
	for _, rule := range rules {
		pat, err := Lex(rule.Pattern)
		if err != nil {
			log.Printf("failed to parse pattern %q: %v", rule.Pattern, err)
			continue
		}
		nodes, err := Parse(pat)
		if err != nil {
			log.Printf("failed to parse pattern %q: %v", rule.Pattern, err)
			continue
		}
		replacement, err := Lex(rule.Replacement)
		if err != nil {
			log.Printf("failed to parse replacement %q: %v", rule.Replacement, err)
			continue
		}
		replacementNodes, err := Parse(replacement)
		if err != nil {
			log.Printf("failed to parse replacement %q: %v", rule.Replacement, err)
			continue
		}

		repl := NewReplacer(nodes, replacementNodes)
		newResult := repl.ReplaceAll(result)
		if newResult != result {
			log.Printf("applied rule %q: %q -> %q", rule.Name, result, newResult)
		}
		result = newResult
	}
	return result
}
