package readconf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
)

func NewBuilder() *Builder {
	return &Builder{}
}

type Builder struct {
	err      error
	values   Map
	validate *validator.Validate
}

func (b *Builder) Error() error {
	return b.err
}

func (b *Builder) hasError() bool {
	return b.err != nil
}

func (b *Builder) Set(k, v string) *Builder {
	if b.hasError() {
		return b
	}

	m := Map{}
	m.Set(k, v)
	return b.MergeMap(m)
}

func (b *Builder) WithValidator(v *validator.Validate) *Builder {
	if b.hasError() {
		return b
	}

	b.validate = v
	return b
}

func (b *Builder) Build(target interface{}) error {
	if err := validateIsPointerToStruct(target); err != nil {
		return err
	}

	if b.hasError() {
		return b.err
	}

	values := Map{}
	knownFields := map[string]reflect.Value{}

	// walk fields
	if err := walkStruct(
		target,
		func(path []string, f reflect.StructField, v reflect.Value) (bool, error) {
			if !v.CanSet() {
				return false, nil
			}

			if tag, ok := f.Tag.Lookup(_configTag); ok && tag != `` {
				if tag == `-` {
					return false, nil
				}

				path = append(path[:len(path)-1], normalizeKey(tag))
			}

			key := structKey(path)

			if canUnmarshalDirectly(v) {
				knownFields[key] = v

				if tag, ok := f.Tag.Lookup(_defaultTag); ok {
					values.Set(key, tag)
				}
			}

			return true, nil
		},
	); err != nil {
		return err
	}

	// walk structs
	if err := walkStruct(
		target,
		func(path []string, f reflect.StructField, v reflect.Value) (bool, error) {
			if !v.CanSet() {
				return false, nil
			}

			if tag, ok := f.Tag.Lookup(_configTag); ok && tag != `` {
				if tag == `-` {
					return false, nil
				}

				path1 := make([]string, len(path))
				copy(path1, path)
				path1[len(path1)-1] = normalizeKey(tag)
				path = path1
			}

			key := structKey(path)

			if v.Type().Implements(_defaultConfigType) {
				if m1 := v.Interface().(DefaultConfig).DefaultConfig(); m1 != nil {
					m2 := make(Map, len(m1))
					for k, v := range m1 {
						if key != "" {
							k = key + _separator + k
						}
						m2[k] = v
					}

					values.Merge(m2)
				}
			}

			return true, nil
		},
	); err != nil {
		return err
	}

	values.Merge(b.values)

	{
		missingKeys := []string{}
		for key := range knownFields {
			if _, ok := values.Lookup(key); !ok {
				missingKeys = append(missingKeys, key)
			}
		}
		sort.Strings(missingKeys)

		if len(missingKeys) > 0 {
			plural := ""
			if len(missingKeys) > 1 {
				plural = "s"
			}

			return fmt.Errorf(
				"missing %d configuration key%s: %s",
				len(missingKeys), plural,
				strings.Join(missingKeys, ", "))
		}
	}

	if err := resolveValueMap(values); err != nil {
		return wrapError(err, "resolve values")
	}

	for key, field := range knownFields {
		if err := values.Unmarshal(key, field.Addr().Interface()); err != nil {
			return wrapError(err, "unmarshal value")
		}
	}

	if err := b.Validator().Struct(target); err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			keys := make([]string, 0, len(errs))

			for _, err := range errs {
				var key string

				if ns := strings.SplitN(err.StructNamespace(), ".", 2); len(ns) == 2 {
					key = ns[1]
				} else {
					key = ns[0]
				}

				key = stringReplaceAll(key, `.`, _separator)
				key = normalizeKey(key)
				keys = append(keys, key)
			}

			sort.Strings(keys)

			return fmt.Errorf(`validation failed: %s`, strings.Join(keys, `, `))
		}

		return err
	}

	return nil
}

func (b *Builder) MustBuild(v interface{}) {
	if err := b.Build(v); err != nil {
		panic(err)
	}
}

func (b *Builder) MergeFile(filename string) *Builder {
	if b.hasError() {
		return b
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		b.err = err
		return b
	}

	return b.MergeData(data)
}

func (b *Builder) MergeData(data []byte) *Builder {
	if b.hasError() {
		return b
	}

	lines := bytes.Split(data, []byte("\n"))
	m := make(Map, len(lines))

	for i, line := range lines {
		line := bytes.TrimSpace(line)

		switch {
		case len(line) == 0:
			continue
		case line[0] == '#':
			continue
		}

		kvp := bytes.SplitN(line, []byte("="), 2)

		key := string(bytes.TrimSpace(kvp[0]))
		if len(key) == 0 {
			b.err = fmt.Errorf(`invalid empty key on line %d`, i+1)
			return b
		}

		if len(kvp) == 1 {
			m[key] = ``
		} else {
			m[key] = string(bytes.TrimSpace(kvp[1]))
		}
	}

	return b.MergeMap(m)
}

func (b *Builder) MergeEnviron(prefix string, env []string) *Builder {
	if b.hasError() {
		return b
	}

	m := make(Map)

	for _, x := range env {
		kvp := strings.SplitN(x, "=", 2)
		key := kvp[0]

		if !strings.HasPrefix(key, prefix) {
			continue
		}

		key = strings.TrimPrefix(key, prefix)

		if len(kvp) == 1 {
			m[key] = ""
		} else {
			m[key] = kvp[1]
		}
	}

	return b.MergeMap(m)
}

func (b *Builder) MergeMap(m Map) *Builder {
	if b.hasError() {
		return b
	}

	if b.values == nil {
		b.values = Map{}
	}

	for k, v := range m {
		b.values[k] = v
	}

	return b
}

func (b *Builder) MapValidator(f func(v *validator.Validate)) *Builder {
	if b.hasError() {
		return b
	}

	f(b.validate)
	return b
}

func (b *Builder) Validator() *validator.Validate {
	if b.validate == nil {
		return validator.New()
	}

	return b.validate
}
