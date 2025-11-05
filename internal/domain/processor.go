package domain

import (
    "fmt"
    "net/url"
    "regexp"
    "strings"
)

var (
    labelRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
)

func NormalizeLabel(input string) (string, error) {
    if input == "" {
        return "", fmt.Errorf("empty input")
    }

    if strings.Contains(input, "://") {
        u, err := url.Parse(input)
        if err == nil && u.Host != "" {
            input = u.Host
        }
    }

    input = strings.TrimSuffix(input, ".")
    if strings.Contains(input, ".") {
        parts := strings.SplitN(input, ".", 2)
        input = parts[0]
    }

    input = strings.ToLower(input)

    if !isValidLabel(input) {
        return "", fmt.Errorf("invalid domain label: %s", input)
    }

    return input, nil
}

func isValidLabel(label string) bool {
    if len(label) > 63 || len(label) == 0 {
        return false
    }
    return labelRegexp.MatchString(label)
}
