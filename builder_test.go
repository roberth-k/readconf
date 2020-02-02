package configkit_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tetratom/configkit"
)

type Embedded struct {
	Bax string
	Baf int
}

func (Embedded) DefaultConfig() configkit.Map {
	return configkit.Map{"BAF": "3"}
}

type Nested struct {
	Foo string `validate:"required"`
	Bar int
	Bax string
}

func (Nested) DefaultConfig() configkit.Map {
	return configkit.Map{"FOO": "bar"}
}

type Config struct {
	Foo string `default:"xyzz"`
	Bar int
	Embedded
	Nested Nested
}

func (Config) DefaultConfig() configkit.Map {
	return configkit.Map{"BAR": "10"}
}

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

func TestBuilder_Defaults(t *testing.T) {
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
}

func TestBuilder(t *testing.T) {
	t.Run("smoke test", func(t *testing.T) {
		t.Run("initial values", func(t *testing.T) {
			var conf Config
			err := configkit.
				NewBuilder().
				Build(&conf)

			t.Log(conf)
			require.NoError(t, err)
			require.Equal(t, 3, conf.Embedded.Baf)
			require.Equal(t, "bar", conf.Nested.Foo)
			require.Equal(t, 10, conf.Bar)
			require.Equal(t, "xyzz", conf.Foo)
		})

		t.Run("merge map", func(t *testing.T) {
			var conf Config
			err := configkit.
				NewBuilder().
				MergeMap(configkit.Map{
					"FOO":         "foofoo",
					"NESTED__BAR": "5",
					"BAF":         "99",
				}).
				Build(&conf)

			t.Log(conf)
			require.NoError(t, err)
			require.Equal(t, 99, conf.Embedded.Baf)
			require.Equal(t, "bar", conf.Nested.Foo)
			require.Equal(t, 5, conf.Nested.Bar)
			require.Equal(t, "", conf.Nested.Bax)
			require.Equal(t, 10, conf.Bar)
			require.Equal(t, "foofoo", conf.Foo)
		})

		t.Run("merge data", func(t *testing.T) {
			var conf Config
			err := configkit.
				NewBuilder().
				MergeData([]byte(`
					NESTED__BAX=blah
					NESTED__FOO=${NESTED__BAX}
					FOO=${NESTED__FOO}
				`)).
				Build(&conf)

			t.Log(conf)
			require.NoError(t, err)
			require.Equal(t, "blah", conf.Nested.Bax)
			require.Equal(t, "blah", conf.Nested.Foo)
			require.Equal(t, "blah", conf.Foo)
		})

		t.Run("merge file", func(t *testing.T) {
			var conf Config
			err := configkit.
				NewBuilder().
				MergeFile("testdata/config.env").
				Build(&conf)

			t.Log(conf)
			require.NoError(t, err)
			require.Equal(t, "foo from file", conf.Foo)
			require.Equal(t, 1, conf.Nested.Bar)
		})
	})
}
