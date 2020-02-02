// +build !go1.13

package readconf

import "fmt"

func wrapError(err error, msg string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	args = append(args, err)
	return fmt.Errorf(msg+": %v", args...)
}
