package key

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Id      string `yaml:"id"`
	Enabled bool   `yaml:"enabled"`
	Pattern string `yaml:"pattern"`
}

type Rules struct {
	Rules []Rule `yaml:"rules"`
}

func init() {
	configDir := "config"
	configFile := filepath.Join(configDir, "rule.yaml")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return
		}
		CreateConfigFile()
	}
}

func ReadRuleFile() (*Rules, error) {
	configFile := filepath.Join("config", "rule.yaml")
	file, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading rule file: %v", err)
	}

	var rules Rules
	if err := yaml.Unmarshal(file, &rules); err != nil {
		return nil, fmt.Errorf("error unmarshalling rule file: %v", err)
	}

	return &rules, nil
}

func CreateConfigFile() {
	configFile := filepath.Join("config", "rule.yaml")
	defaultRules := Rules{
		Rules: []Rule{
			{Id: "domain", Enabled: false, Pattern: ""},
			{Id: "path", Enabled: false, Pattern: ""},
			{Id: "domain_url", Enabled: false, Pattern: ""},
			{Id: "ip", Enabled: false, Pattern: ""},
			{Id: "ip_url", Enabled: false, Pattern: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`},
			{Id: "email", Enabled: true, Pattern: `\b[A-Za-z0-9._\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,61}\b`},
			{Id: "id_card", Enabled: true, Pattern: `\b([1-9]\d{5}(19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx])\b`},
			{Id: "phone", Enabled: true, Pattern: `\b1[3-9]\d{9}\b`},
			{Id: "jwt_token", Enabled: true, Pattern: `eyJ[A-Za-z0-9_/+\-]{10,}={0,2}\.[A-Za-z0-9_/+\-\\]{15,}={0,2}\.[A-Za-z0-9_/+\-\\]{10,}={0,2}`},
			{Id: "Aliyun_AK_ID", Enabled: true, Pattern: `\bLTAI[A-Za-z\d]{12,30}\b`},
			{Id: "QCloud_AK_ID", Enabled: true, Pattern: `\bAKID[A-Za-z\d]{13,40}\b`},
			{Id: "JDCloud_AK_ID", Enabled: true, Pattern: `\bJDC_[0-9A-Z]{25,40}\b`},
			{Id: "AWS_AK_ID", Enabled: true, Pattern: `["''](?:A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}["'']`},
			{Id: "VolcanoEngine_AK_ID", Enabled: true, Pattern: `\b(?:AKLT|AKTP)[a-zA-Z0-9]{35,50}\b`},
			{Id: "Kingsoft_AK_ID", Enabled: true, Pattern: `\bAKLT[a-zA-Z0-9-_]{16,28}\b`},
			{Id: "GCP_AK_ID", Enabled: true, Pattern: `\bAIza[0-9A-Za-z_\-]{35}\b`},
			{Id: "secret_key", Enabled: true, Pattern: ""},
			{Id: "bearer_token", Enabled: true, Pattern: `\b[Bb]earer\s+[a-zA-Z0-9\-=._+/\\]{20,500}\b`},
			{Id: "basic_token", Enabled: true, Pattern: `\b[Bb]asic\s+[A-Za-z0-9+/]{18,}={0,2}\b`},
			{Id: "auth_token", Enabled: true, Pattern: `["''\[]*[Aa]uthorization["''\]]*\s*[:=]\s*[''"]?\b(?:[Tt]oken\s+)?[a-zA-Z0-9\-_+/]{20,500}[''"]?`},
			{Id: "private_key", Enabled: true, Pattern: `-----\s*?BEGIN[ A-Z0-9_-]*?PRIVATE KEY\s*?-----[a-zA-Z0-9\/\n\r=+]*-----\s*?END[ A-Z0-9_-]*? PRIVATE KEY\s*?-----`},
			{Id: "gitlab_v2_token", Enabled: true, Pattern: `\b(glpat-[a-zA-Z0-9\-=_]{20,22})\b`},
			{Id: "github_token", Enabled: true, Pattern: `\b((?:ghp|gho|ghu|ghs|ghr|github_pat)_[a-zA-Z0-9_]{36,255})\b`},
			{Id: "qcloud_api_gateway_appkey", Enabled: true, Pattern: `\bAPID[a-zA-Z0-9]{32,42}\b`},
			{Id: "wechat_appid", Enabled: true, Pattern: `["''](wx[a-z0-9]{15,18})["'']`},
			{Id: "wechat_corpid", Enabled: true, Pattern: `["''](ww[a-z0-9]{15,18})["'']`},
			{Id: "wechat_id", Enabled: true, Pattern: `["''](gh_[a-z0-9]{11,13})["'']`},
			{Id: "password", Enabled: true, Pattern: `(?i)(?:admin_?pass|password|[a-z]{3,15}_?password|user_?pass|user_?pwd|admin_?pwd)\\?['"]*\s*[:=]\s*\\?['"][a-z0-9!@#$%&*]{5,50}\\?['"]`},
			{Id: "wechat_webhookurl", Enabled: true, Pattern: `\bhttps://qyapi.weixin.qq.com/cgi-bin/webhook/send\?key=[a-zA-Z0-9\-]{25,50}\b`},
			{Id: "dingtalk_webhookurl", Enabled: true, Pattern: `\bhttps://oapi.dingtalk.com/robot/send\?access_token=[a-z0-9]{50,80}\b`},
			{Id: "feishu_webhookurl", Enabled: true, Pattern: `\bhttps://open.feishu.cn/open-apis/bot/v2/hook/[a-z0-9\-]{25,50}\b`},
			{Id: "slack_webhookurl", Enabled: true, Pattern: `\bhttps://hooks.slack.com/services/[a-zA-Z0-9\-_]{6,12}/[a-zA-Z0-9\-_]{6,12}/[a-zA-Z0-9\-_]{15,24}\b`},
			{Id: "grafana_api_key", Enabled: true, Pattern: `\beyJrIjoi[a-zA-Z0-9\-_+/]{50,100}={0,2}\b`},
			{Id: "grafana_cloud_api_token", Enabled: true, Pattern: `\bglc_[A-Za-z0-9\-_+/]{32,200}={0,2}\b`},
			{Id: "grafana_service_account_token", Enabled: true, Pattern: `\bglsa_[A-Za-z0-9]{32}_[A-Fa-f0-9]{8}\b`},
			{Id: "app_key", Enabled: true, Pattern: `\b(?:VUE|APP|REACT)_[A-Z_0-9]{1,15}_(?:KEY|PASS|PASSWORD|TOKEN|APIKEY)['"]*[:=]"(?:[A-Za-z0-9_\-]{15,50}|[a-z0-9/+]{50,100}==?)"`},
		},
	}

	data, err := yaml.Marshal(&defaultRules)
	if err != nil {
		fmt.Printf("Error marshalling default rules: %v\n", err)
		return
	}

	if err := os.WriteFile(configFile, data, 0755); err != nil {
		fmt.Printf("Error writing default rule file: %v\n", err)
	}
}
