package key

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	rulesInstance *Rules
	once          sync.Once
	jsonMutex     sync.Mutex
)

func getRulesInstance() (*Rules, error) {
	var err error
	once.Do(func() {
		rulesInstance, err = ReadRuleFile()
	})
	return rulesInstance, err
}

func MatchRules(input string) error {
	rules, err := getRulesInstance()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	for _, rule := range rules.Rules {
		if rule.Enabled {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return fmt.Errorf("failed to compile regex for rule %s: %v", rule.Id, err)
			}
			matches := re.FindAllStringSubmatch(input, -1)
			for _, match := range matches {
				if len(match) > 0 {
					if strings.TrimSpace(match[0]) == "" {
						continue
					}
					err := appendToJSON(rule.Id, match[0])
					if err != nil {
						return fmt.Errorf("failed to append to JSON: %v", err)
					}
				}
			}
		}
	}

	return nil
}

func appendToJSON(ruleId, matchedContent string) error {
	jsonMutex.Lock()
	defer jsonMutex.Unlock()

	file, err := os.OpenFile("sensitive_data.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open JSON file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("failed to close JSON file: %v", err)
		}
	}(file)

	record := map[string]string{
		"rule_id": ruleId,
		"content": matchedContent,
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("failed to write to JSON file: %v", err)
	}

	return nil
}
