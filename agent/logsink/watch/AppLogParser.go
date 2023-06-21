package watch

import (
	"encoding/json"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/go-api/agent/util/logutil"
)

func ApplyAppLog(p *pack.LogSinkPack, line string) (ret bool) {
	if p == nil || len(line) < 1 {
		return
	}

	matches := AppLogPattern.FindAllStringSubmatch(line, -1)

	for _, submatchpair := range matches {

		if len(submatchpair) < 2 {
			continue
		}
		jsonString := submatchpair[1]
		if len(jsonString) < 1 {
			continue
		}
		tags := map[string]string{}

		err := json.Unmarshal([]byte(jsonString), &tags)
		if err == nil {
			ret = true
			for k, v := range tags {
				if k == TxIdTag {
					p.Category = AppLogCategory
				}
				p.Tags.PutString(k, v)
			}
		} else {
			if DebugAppLogParser {
				logutil.Println("WA-ALP-39", "Error :", err.Error())
			}
		}
	}

	return
}

func validateTxHeader(line string) (ret bool) {

	matches := AppLogPattern.FindAllStringSubmatch(line, -1)
	return len(matches) > 0
}
