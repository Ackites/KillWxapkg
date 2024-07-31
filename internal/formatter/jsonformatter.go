package formatter

import (
	"bytes"
	"encoding/json"
)

// JSONFormatter 格式化 JSON 文件
type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Format(input []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, input, "", "    ")
	if err != nil {
		return nil, err
	}
	return prettyJSON.Bytes(), nil
}

func init() {
	RegisterFormatter(".json", NewJSONFormatter())
}
