package configkit

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	_capital1  = regexp.MustCompile(`[A-Z][a-z]+`)
	_capital2  = regexp.MustCompile(`[A-Z][A-Z]+`)
	_reference = regexp.MustCompile(`\$\{([^}]+)\}`)
)

func parseReferences(v string) []string {
	ss := _reference.FindAllStringSubmatch(v, -1)
	out := make([]string, len(ss))
	for i := range ss {
		out[i] = ss[i][1]
	}
	return out
}

func replaceReferences(s string, data Map) string {
	return _reference.ReplaceAllStringFunc(s, func(s string) string {
		v, ok := data[s[2:len(s)-1]]
		if !ok {
			return s
		}

		return v
	})
}

func transformStructKey(v string) string {
	v = _capital2.ReplaceAllString(v, `_$0`)
	v = _capital1.ReplaceAllStringFunc(v, func(s string) string {
		return "_" + strings.ToUpper(s)
	})
	v = strings.Trim(v, "_")
	return v
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ToUpper(key)
	return key
}

func copyAppend(in []string, ss ...string) []string {
	out := make([]string, 0, len(in)+len(ss))
	out = append(out, in...)
	out = append(out, ss...)
	return out
}

func extractStructFields(v interface{}) fieldMap {
	out := fieldMap{}

	var walk func(vv reflect.Value, path []string)

	walk = func(vv reflect.Value, path []string) {
		vt := vv.Type()

		for i := 0; i < vv.NumField(); i++ {
			fv, ft := vv.Field(i), vt.Field(i)

			if fv.Kind() == reflect.Struct {
				path := path
				if !ft.Anonymous {
					path = copyAppend(path, ft.Name)
				}

				walk(fv, path)
			}

			out.Set(path, ft, fv)
		}
	}

	walk(reflect.ValueOf(v).Elem(), nil)
	return out
}

func resolveValueMap(m Map) error {
	for oneMorePass, lastRefMapLen := true, 0; oneMorePass; {
		oneMorePass = false

		// collect all values without references: these
		// are eligible to be used as reference values.
		refMap := Map{}
		unrefs := []string{}
		for k, v := range m {
			if len(parseReferences(v)) == 0 {
				refMap[k] = v
			} else {
				unrefs = append(unrefs, k)
			}
		}

		if len(refMap) == lastRefMapLen {
			return fmt.Errorf(
				"suspected cyclic reference between: %s",
				strings.Join(unrefs, ", "))
		}

		lastRefMapLen = len(refMap)

		for k, v := range m {
			refs := parseReferences(v)

			for _, ref := range refs {
				if ref == k {
					return fmt.Errorf("cyclic reference in %s", k)
				}

				if _, ok := refMap[ref]; !ok {
					oneMorePass = true
					goto nextValue
				}
			}

			// reaching this means all references can be resolved
			m[k] = replaceReferences(v, refMap)

		nextValue:
		}
	}

	return nil
}
