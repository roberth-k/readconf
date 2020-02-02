// +build !go1.12

package readconf

import "strings"

func stringReplaceAll(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}
