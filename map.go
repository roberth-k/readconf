package configkit

import (
	"encoding"
	"fmt"
	"reflect"
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

type fieldMap map[string]structField

func (m fieldMap) Set(path []string, f reflect.StructField, v reflect.Value) {
	var key string
	{
		ss := make([]string, len(path))
		for i := range path {
			ss[i] = transformStructKey(path[i])
		}
		key = strings.Join(ss, "__")
		key = normalizeKey(key)
	}

	m[key] = structField{
		field: f,
		value: v,
	}
}
