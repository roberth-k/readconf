package configkit

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
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

	vt := reflect.TypeOf(v)
	if vt.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to value")
	}

	vv, vt := reflect.ValueOf(v).Elem(), vt.Elem()

	value, ok := m.Lookup(key)
	if !ok {
		return fmt.Errorf("not found")
	}

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
