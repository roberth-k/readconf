package readconf

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var (
	_capital1  = regexp.MustCompile(`[A-Z][a-z]+`)
	_capital2  = regexp.MustCompile(`[A-Z][A-Z]+`)
	_reference = regexp.MustCompile(`\$\{([^}]+)(?:\:-[^}]*)?\}`)
)

func parseReferences(v string) (refs []string, defaults map[string]string) {
	ss := _reference.FindAllStringSubmatch(v, -1)

	refs = make([]string, len(ss))
	defaults = make(map[string]string, len(ss))

	for i := range ss {
		sss := strings.SplitN(ss[i][1], ":-", 2)
		refs[i] = sss[0]
		if len(sss) == 2 {
			defaults[sss[0]] = sss[1]
		}
	}

	return
}

// ignores defaults
func replaceReferences(s string, data Map) string {
	return _reference.ReplaceAllStringFunc(s, func(s string) string {
		ss := strings.SplitN(s[2:len(s)-1], ":-", 2)
		v, ok := data[ss[0]]
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

// Resolves references among values in the given Map. A reference is a substring
// of the form ${other_var} or ${other_var:-default}. To "resolve" is to replace
// references with their values from the given map.
//
// A value in the map is available to be used for resolution if it no longer
// contains any references itself.
func resolveValueMap(m Map) error {
	var resolve func(key string, cycle []string) (bool, error)

	resolve = func(key string, cycle []string) (bool, error) {
		for _, ref := range cycle {
			if ref == key {
				return false, fmt.Errorf(
					`cyclic reference: %s, %s`,
					key, strings.Join(cycle, `, `))
			}
		}

		cycle = append([]string{key}, cycle...)

		value, ok := m[key]
		if !ok {
			return false, nil
		}

		valueRefs, valueDefs := parseReferences(value)
		resolved := make(Map, len(valueRefs))

		for _, ref := range valueRefs {
			ok, err := resolve(ref, cycle)
			switch {
			case err != nil:
				return false, err
			case !ok:
				v, ok := valueDefs[ref]
				if !ok {
					return false, fmt.Errorf(`key %s referenced by %s not found`, ref, key)
				}

				resolved[ref] = v
			default:
				resolved[ref] = m[ref]
			}
		}

		if len(resolved) != len(valueRefs) {
			panic(`expected to have resolved all references in ` + key)
		}

		m[key] = replaceReferences(m[key], resolved)
		return true, nil
	}

	sortedKeys := make([]string, 0, len(m))
	for k := range m {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, k := range sortedKeys {
		ok, err := resolve(k, nil)
		switch {
		case err != nil:
			return err
		case !ok:
			panic(`unexpected !ok for ` + k)
		}
	}

	return nil
}

func walkStruct(
	x interface{},
	walker func(path []string, f reflect.StructField, v reflect.Value) (bool, error),
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

			if ok, err := walker(path, ft, fv); err != nil {
				return err
			} else if !ok {
				continue
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

	if ok, err := walker([]string{}, wrapper, xv); err != nil {
		return err
	} else if !ok {
		return nil
	}

	return walk(xv, wrapper, nil)
}

// Returns true when the given value is something we can
// unmarshal config into.
func canUnmarshalDirectly(v reflect.Value) bool {
	t := v.Type()

	switch {
	case t.Implements(_unmarshalerType):
		return true
	case t.Kind() == reflect.Struct:
		return false
	default:
		return true
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
