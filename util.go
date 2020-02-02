package readconf

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

func validateIsPointerToStruct(v interface{}) error {
	switch {
	case v == nil:
		return fmt.Errorf("expected non-nil target")
	case reflect.TypeOf(v).Kind() != reflect.Ptr:
		return fmt.Errorf("expected a pointer")
	case reflect.TypeOf(v).Elem().Kind() != reflect.Struct:
		return fmt.Errorf("expected pointer to struct")
	default:
		return nil
	}
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

func walkStruct(
	x interface{},
	walker func(path []string, f reflect.StructField, v reflect.Value) error,
) error {
	xv := reflect.ValueOf(x)
	if xv.Type().Kind() == reflect.Ptr {
		xv = xv.Elem()
	}

	if xv.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct")
	}

	var walk func(vv reflect.Value, f reflect.StructField, prefix []string) error

	walk = func(vv reflect.Value, f reflect.StructField, prefix []string) error {
		vt := vv.Type()

		for i := 0; i < vt.NumField(); i++ {
			fv, ft := vv.Field(i), vt.Field(i)

			path := prefix
			if !ft.Anonymous {
				path = copyAppend(path, ft.Name)
			}

			if err := walker(path, ft, fv); err != nil {
				return err
			}

			if ft.Type.Kind() == reflect.Struct {
				if err := walk(fv, ft, path); err != nil {
					return err
				}
			}
		}

		return nil
	}

	wrapper := reflect.StructField{
		Name:      "",
		PkgPath:   "",
		Type:      xv.Type(),
		Tag:       "",
		Offset:    0,
		Index:     nil,
		Anonymous: false,
	}

	if err := walker([]string{}, wrapper, xv); err != nil {
		return err
	}

	return walk(xv, wrapper, nil)
}

func canAssignConfig(v reflect.Value) bool {
	t := v.Type()

	switch {
	case t.Implements(_unmarshalerType):
		return true
	case t.Kind() != reflect.Struct:
		return true
	default:
		return false
	}
}

func structKey(path []string) string {
	ss := make([]string, len(path))
	for i := range path {
		ss[i] = transformStructKey(path[i])
	}

	key := strings.Join(ss, "__")
	key = normalizeKey(key)

	return key
}
