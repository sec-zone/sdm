package utils

import "strings"

func ParseHeaders(input string) map[string]string {
	parsedHeaders := map[string]string{}
	headersArray := strings.Split(input, ";")
	for _, header := range headersArray {
		keyValue := strings.Split(header, "=")
		parsedHeaders[keyValue[0]] = strings.TrimSpace(keyValue[1])
	}
	return parsedHeaders
}
