package configkit_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tetratom/configkit"
)

func b() *configkit.Builder {
	return configkit.NewBuilder()
}

func TestBuilder_ExpectStructPointer(t *testing.T) {
	t.Run("nil target", func(t *testing.T) {
		err := b().Build(nil)
		require.Errorf(t, err, "expected non-nil target")
	})

	t.Run("struct by value", func(t *testing.T) {
		var s struct {
			Foo string
		}
		err := b().Build(s)
		require.Errorf(t, err, "expected a pointer")
		require.Empty(t, s)
	})

	t.Run("not passing a struct", func(t *testing.T) {
		var s string
		err := b().Build(&s)
		require.Errorf(t, err, "expected pointer to struct")
		require.Empty(t, s)
	})
}

type EmbeddedWithAllDefaults struct {
	EmbeddedFoo string `default:"test11"`
	EmbeddedBar int    `default:"12"`
}

type NestedWithAllDefaults struct {
	Foo string `default:"test21"`
	Bar int    `default:"22"`
}

type configWithAllDefaults struct {
	Foo string `default:"test1"`
	Bar int    `default:"2"`
	EmbeddedWithAllDefaults
	Nested NestedWithAllDefaults
	ignore string // nolint shouldn't affect anything
}

type EmbeddedWithPartialDefaults struct {
	EmbeddedFoo string `default:"test11"`
	EmbeddedBar int
}

type NestedWithPartialDefaults struct {
	Foo string
	Bar int `default:"22"`
}

type configWithPartialDefaults struct {
	Foo string
	Bar int `default:"1"`
	EmbeddedWithPartialDefaults
	Nested NestedWithPartialDefaults
	ignore string // nolint shouldn't affect anything
}

type EmbeddedWithInterfacedDefaults struct {
	EmbeddedFoo string `default:"test11"`
	EmbeddedBar int
}

func (EmbeddedWithInterfacedDefaults) DefaultConfig() configkit.Map {
	return configkit.Map{
		`EMBEDDED_BAR`: `12`,
	}
}

type NestedWithInterfacedDefaults struct {
	Foo string
	Bar int `default:"22"`
}

func (NestedWithInterfacedDefaults) DefaultConfig() configkit.Map {
	return configkit.Map{
		`FOO`: `test21`,
	}
}

type configWithInterfacedDefaults struct {
	Foo string `default:"test1"`
	Bar int
	EmbeddedWithInterfacedDefaults
	Nested NestedWithInterfacedDefaults
	ignore string // nolint shouldn't affect anything
}

func (configWithInterfacedDefaults) DefaultConfig() configkit.Map {
	return configkit.Map{`BAR`: `2`}
}

func TestBuilder_Build(t *testing.T) {
	t.Run("all defaults provided", func(t *testing.T) {
		var conf configWithAllDefaults
		err := b().Build(&conf)
		require.NoError(t, err)
		require.Equal(t, configWithAllDefaults{
			Foo: "test1",
			Bar: 2,
			EmbeddedWithAllDefaults: EmbeddedWithAllDefaults{
				EmbeddedFoo: "test11",
				EmbeddedBar: 12,
			},
			Nested: NestedWithAllDefaults{
				Foo: "test21",
				Bar: 22,
			},
			ignore: "",
		}, conf)
	})

	t.Run("some defaults provided", func(t *testing.T) {
		t.Run("root level value not provided", func(t *testing.T) {
			var conf configWithPartialDefaults
			err := b().Build(&conf)
			require.Errorf(t, err, "missing configuration key: FOO")
			require.Empty(t, conf)
		})

		t.Run("embedded value not provided", func(t *testing.T) {
			var conf configWithPartialDefaults
			err := b().
				MergeMap(configkit.Map{
					`FOO`:         `bar`,
					`NESTED__FOO`: `baf`,
				}).
				Build(&conf)
			require.Errorf(t, err, "missing configuration key: EMBEDDED_FOO")
			require.Empty(t, conf)
		})

		t.Run("nested value not provided", func(t *testing.T) {
			var conf configWithPartialDefaults
			err := b().
				MergeMap(configkit.Map{
					`FOO`:          `bar`,
					`EMBEDDED_FOO`: `baf`,
				}).
				Build(&conf)
			require.Errorf(t, err, "missing configuration key: NESTED__FOO")
			require.Empty(t, conf)
		})
	})

	t.Run("defaults interface provided", func(t *testing.T) {
		var conf configWithInterfacedDefaults
		err := b().Build(&conf)
		require.NoError(t, err)
		require.Equal(t, configWithInterfacedDefaults{
			Foo: "test1",
			Bar: 2,
			EmbeddedWithInterfacedDefaults: EmbeddedWithInterfacedDefaults{
				EmbeddedFoo: "test11",
				EmbeddedBar: 12,
			},
			Nested: NestedWithInterfacedDefaults{
				Foo: "test21",
				Bar: 22,
			},
			ignore: "",
		}, conf)
	})

	t.Run("validation", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			var conf struct {
				Foo string `default:"aaa" validate:"min=2"`
			}

			err := b().Build(&conf)
			require.NoError(t, err)
			require.Equal(t, `aaa`, conf.Foo)
		})

		t.Run("failure", func(t *testing.T) {
			var conf struct {
				Foo string `default:"a" validate:"min=2"`
				Bar string `default:"a" validate:"min=2"`
			}

			err := b().Build(&conf)
			require.Errorf(t, err, "validation failed: BAR, FOO")
		})
	})
}

func TestBuilder_MergeMap(t *testing.T) {
	var conf configWithPartialDefaults
	err := b().
		MergeMap(configkit.Map{
			`FOO`:          `foofoo`,
			`BAR`:          `2`,
			`EMBEDDED_BAR`: `99`,
			`NESTED__FOO`:  `nested_foo`,
		}).
		Build(&conf)
	require.NoError(t, err)
	require.Equal(t, configWithPartialDefaults{
		Foo: "foofoo",
		Bar: 2,
		EmbeddedWithPartialDefaults: EmbeddedWithPartialDefaults{
			EmbeddedFoo: "test11",
			EmbeddedBar: 99,
		},
		Nested: NestedWithPartialDefaults{
			Foo: "nested_foo",
			Bar: 22,
		},
		ignore: "",
	}, conf)
}

func TestBuilder_MergeData(t *testing.T) {
	var conf configWithPartialDefaults
	err := b().
		MergeData([]byte(`
			FOO = foofoo

			BAR = 2
			# comment
			EMBEDDED_BAR = ${BAR}9
			NESTED__FOO = nested_${FOO}_foo
		`)).
		Build(&conf)

	require.NoError(t, err)
	require.Equal(t, configWithPartialDefaults{
		Foo: "foofoo",
		Bar: 2,
		EmbeddedWithPartialDefaults: EmbeddedWithPartialDefaults{
			EmbeddedFoo: "test11",
			EmbeddedBar: 29,
		},
		Nested: NestedWithPartialDefaults{
			Foo: "nested_foofoo_foo",
			Bar: 22,
		},
		ignore: "",
	}, conf)
}

func TestBuilder_MergeEnviron(t *testing.T) {
	var conf configWithPartialDefaults
	err := b().
		MergeEnviron(`APP__`, []string{
			`FOO=foo1`,
			`APP__FOO=foo2`,
			`APP__BAR=2`,
			`APP__EMBEDDED_BAR=${BAR}9`,
			`APP__NESTED__FOO=nested_${FOO}_foo`,
		}).
		Build(&conf)

	require.NoError(t, err)
	require.Equal(t, configWithPartialDefaults{
		Foo: "foo2",
		Bar: 2,
		EmbeddedWithPartialDefaults: EmbeddedWithPartialDefaults{
			EmbeddedFoo: "test11",
			EmbeddedBar: 29,
		},
		Nested: NestedWithPartialDefaults{
			Foo: "nested_foo2_foo",
			Bar: 22,
		},
		ignore: "",
	}, conf)
}