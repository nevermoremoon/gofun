package n9e

import (
	"encoding/json"
	"fmt"
	"strings"
)

func convertToStruct(in, out interface{}) error {
	b, err := json.Marshal(in)
	if err == nil {
		err = json.Unmarshal(b, out)
	}
	return err
}

func LabelToMap(in string) (out map[string]string) {
	out = make(map[string]string)
	for _, v := range strings.Split(in, ",") {
		label := strings.Split(v, "=")
		if label[0] == "" {
			continue
		}
		switch len(label) {
		case 1:
			out[label[0]] = ""
		case 2:
			out[label[0]] = label[1]
		default:
			;
		}
	}
	return
}

func LabelToString(in map[string]string) (out string) {
	labelString := ""
	for k, v := range in {
		labelString = fmt.Sprintf("%s,%s=%s", labelString, k, v)
	}
	labelString = strings.Trim(labelString, ",")
	out = labelString
	return
}