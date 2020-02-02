package configkit

import (
	"encoding"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type Map map[string]string

func (m Map) Lookup(key string) (string, bool) {
	key = normalizeKey(key)
	v, ok := m[key]
	return v, ok
}

func (m Map) Get(key string) string {
	v, _ := m.Lookup(key)
	return v
}

func (m Map) Set(key, value string) {
	key = normalizeKey(key)
	m[key] = value
}

func (m Map) Unmarshal(key string, v interface{}) (err error) {
	defer func() {
		err = wrapError(err, "configuration key \"%s\"", key)
	}()

	value, ok := m.Lookup(key)
	if !ok {
		return fmt.Errorf("not found")
	}

	vv, vt := reflect.ValueOf(v), reflect.TypeOf(v)
	switch {
	case vt.Implements(_unmarshalerType):
		return vv.Interface().(Unmarshaler).UnmarshalConfig(value)
	case vt.Implements(_textUnmarshalerType):
		return vv.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value))
	default:
		switch vt.Kind() {
		case reflect.String:
			vv.SetString(value)
			return nil
		case reflect.Int, reflect.Int64:
			var iv int64
			iv, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}

			vv.SetInt(iv)
			return nil
		default:
			panic(fmt.Sprintf("reflection of kind %d not implemented", vt.Kind()))
		}
	}
}

func (m Map) Merge(other Map) {
	for k, v := range other {
		m[k] = v
	}
}

type structField struct {
	field reflect.StructField
	value reflect.Value
}

type fieldMap struct {
	m   map[string]structField
	sep string
}

func (m *fieldMap) init() {
	if m.m == nil {
		m.m = map[string]structField{}
	}

	if m.sep == "" {
		m.sep = "__"
	}
}

func (m *fieldMap) Len() int {
	return len(m.m)
}

func (m *fieldMap) Range() map[string]structField {
	out := make(map[string]structField, len(m.m))
	for k, v := range m.m {
		out[k] = v
	}
	return out
}

func (m *fieldMap) Keys() []string {
	m.init()

	out := make([]string, 0, len(m.m))
	for k := range m.m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (m *fieldMap) Lookup(key string) (structField, bool) {
	m.init()

	key = normalizeKey(key)
	v, ok := m.m[key]
	return v, ok
}

func (m *fieldMap) Set(path []string, f reflect.StructField, v reflect.Value) {
	m.init()

	if len(path) == 0 {
		panic("empty path")
	}

	var key string
	{
		ss := make([]string, len(path))
		for i := range path {
			if path[i] == "" {
				panic("empty key")
			}
			ss[i] = transformStructKey(path[i])
		}
		key = strings.Join(ss, m.sep)
		key = normalizeKey(key)
	}

	m.m[key] = structField{
		field: f,
		value: v,
	}
}
