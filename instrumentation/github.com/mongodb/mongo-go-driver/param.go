package whatapmongo

import "fmt"

type parameter struct {
	database   string
	commandRaw string
}

func (param parameter) toString() string {
	return fmt.Sprintf("%s:%s %s:%s",
		"database", param.database,
		"commandRaw", param.commandRaw,
	)
}
