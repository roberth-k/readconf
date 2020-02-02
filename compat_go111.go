// +build go1.11,!go1.12

package readconf

import (
	"fmt"
	"strings"
)

func wrapError(err error, msg string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	args = append(args, err)
	return fmt.Errorf(msg+": %v", args...)
}

func stringReplaceAll(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}
