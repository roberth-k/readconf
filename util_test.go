package readconf

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReferences(t *testing.T) {
	tests := []struct {
		in   string
		refs []string
		defs map[string]string
	}{
		{`foo`, []string{}, map[string]string{}},
		{`${foo}`, []string{`foo`}, map[string]string{}},
		{`this-${foo}-and-${bar}-x`, []string{`foo`, `bar`}, map[string]string{}},
		{`${foo}-${bar:-default}`, []string{`foo`, `bar`}, map[string]string{`bar`: `default`}},
	}

	for _, test := range tests {
		refs, defs := parseReferences(test.in)
		require.ElementsMatch(t, test.refs, refs)
		require.Equal(t, test.defs, defs)
	}
}

func TestReplaceReferences(t *testing.T) {
	require.Equal(t,
		"this-xyz-and-${bar}-xyz-x",
		replaceReferences(
			"this-${foo}-and-${bar}-${foo}-x",
			Map{"foo": "xyz"}))
}

func TestTransformStructKey(t *testing.T) {
	require.Equal(t, "MY", transformStructKey("My"))
	require.Equal(t, "MY_FIELD", transformStructKey("MyField"))
	require.Equal(t, "MY_URL_FIELD", transformStructKey("MyURLField"))
	require.Equal(t, "MY_FIELD_URL", transformStructKey("MyFieldURL"))
	require.Equal(t, "MY_URL_FOR_OAUTH2", transformStructKey("MyURLForOauth2"))
	require.Equal(t, "MY_URL_FOR_O_AUTH2", transformStructKey("MyURLForOAuth2"))
	require.Equal(t, "2_FOO", transformStructKey("2Foo"))
}

func TestNormalizeKey(t *testing.T) {
	require.Equal(t, "MY_FIELD", normalizeKey("MY_FIELD"))
	require.Equal(t, "MY_FIELD", normalizeKey("  my_Field "))
}

func TestWalkStruct(t *testing.T) {
	type Embedded struct {
		Bar int
	}

	var theStruct struct {
		Inner  string
		Nested struct {
			Foo string
		}
		ignored int
		Embedded
	}

	keys := []string{}

	err := walkStruct(
		&theStruct,
		func(path []string, f reflect.StructField, v reflect.Value) error {
			if !v.CanSet() {
				return nil
			}

			key := structKey(path)
			keys = append(keys, key)
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, []string{
		``, `INNER`, `NESTED`, `NESTED__FOO`, ``, `BAR`,
	}, keys)
}

func TestResolveValueMap(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		m := Map{
			`FOO`: `BAR`,
			`BAR`: `1-800-${FOO}`,
			`BAZ`: `MY-${BAR}`,
			`BAF`: `MY-${BAX:-123}`,
			`BAT`: `MY-${BAF}`,
			`BAM`: `MY-${BAF:-000}`,
		}

		err := resolveValueMap(m)

		require.NoError(t, err)
		require.Equal(t, Map{
			`FOO`: `BAR`,
			`BAR`: `1-800-BAR`,
			`BAZ`: `MY-1-800-BAR`,
			`BAF`: `MY-123`,
			`BAT`: `MY-MY-123`,
			`BAM`: `MY-MY-123`,
		}, m)
	})

	t.Run("missing", func(t *testing.T) {
		m := Map{
			`BAR`: `${BAF}`,
		}

		err := resolveValueMap(m)
		require.EqualError(t, err, `key BAF referenced by BAR not found`)
	})

	t.Run("simple cyclic reference", func(t *testing.T) {
		m := Map{
			`BAR`: `${BAR}`,
		}

		err := resolveValueMap(m)
		require.EqualError(t, err, `cyclic reference: BAR, BAR`)
	})

	t.Run("long cyclic reference", func(t *testing.T) {
		m := Map{
			`BAR`: `${BAX}`,
			`BAX`: `${BAR}`,
		}

		err := resolveValueMap(m)

		require.EqualError(t, err, `cyclic reference: BAR, BAX, BAR`)
	})
}
