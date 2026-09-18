package ui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func pathFor(basePath, suffix string) string {
	suffix = strings.Trim(suffix, "/")
	if basePath == "" || basePath == "/" {
		if suffix == "" {
			return "/"
		}
		return "/" + suffix
	}
	if suffix == "" {
		return basePath
	}
	return basePath + "/" + suffix
}

func entityPath(basePath, route string) string {
	return pathFor(basePath, route)
}

func recordPath(basePath, route, id string) string {
	return pathFor(basePath, route+"/"+url.PathEscape(id))
}

func displayValue(value any) string {
	switch value := value.(type) {
	case nil:
		return "—"
	case time.Time:
		return value.Format(time.RFC3339)
	case []byte:
		return string(value)
	case string:
		return value
	}

	if encoded, err := json.Marshal(value); err == nil {
		return string(encoded)
	}
	return fmt.Sprint(value)
}
