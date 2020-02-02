package readconf

import (
	"encoding"
	"reflect"
)

type DefaultConfig interface {
	DefaultConfig() Map
}

type Unmarshaler interface {
	UnmarshalConfig(s string) error
}

var (
	_defaultConfigType   = reflect.TypeOf(new(DefaultConfig)).Elem()
	_unmarshalerType     = reflect.TypeOf(new(Unmarshaler)).Elem()
	_textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()
)
