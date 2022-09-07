package whatapmongo

import (
	"fmt"
	"strings"
)

const (
	protocol = "tcp"
)

type URI struct {
	uri string
}

//example) "mongodb://user:pass@sample.host:27017 => "tcp@user@sample.host:27017"
func NewURI(uri string) URI {
	trimmed := trimUri(uri)

	userStr, addresses := parseUserStr(trimmed), parseAddresses(trimmed)
	return URI{
		uri: fmt.Sprintf("%s@%s%s", protocol, userStr, addresses),
	}
}

func (uri URI) ToString() string {
	return uri.uri
}

func (uri URI) Appended(arg string) URI {
	return URI{
		uri: fmt.Sprintf("%s/%s", uri.ToString(), arg),
	}
}

func removeMongoPrefix(uri string) string {
	splitted := strings.Split(uri, "://")
	if len(splitted) < 2 {
		return uri
	}
	return splitted[1]
}

func cutUriByUser(uri string) (string, string) {
	splitted := strings.Split(uri, "@")
	//user string이 없음
	if len(splitted) <= 1 {
		return "", uri
	}
	userAndPassword := splitted[0]
	addrAndParams := splitted[1]

	return userAndPassword, addrAndParams
}

func parseUserStr(trimmedUri string) string {
	userAndPassword, _ := cutUriByUser(trimmedUri)
	if userAndPassword == "" {
		return ""
	}
	user := strings.Split(userAndPassword, ":")[0]

	return user + "@"
}

func parseAddresses(trimmedUri string) string {
	_, addrAndParams := cutUriByUser(trimmedUri)

	addresses := ""
	for _, ch := range addrAndParams {
		if ch == '/' {
			break
		}
		addresses += string(ch)
	}
	return addresses
}

func trimUri(uri string) string {
	return strings.TrimRight(
		removeParams(
			removeMongoPrefix(uri),
		),
		"/",
	)
}

//remove 'mongodb://' from uri
func removeParams(uri string) string {
	splitted := strings.Split(uri, "?")
	return splitted[0]
}
