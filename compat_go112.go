// +build !go1.13

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
	return strings.ReplaceAll(s, old, new)
}
