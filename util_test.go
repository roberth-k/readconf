package readconf

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReferences(t *testing.T) {
	require.ElementsMatch(t, []string{}, parseReferences("foo"))
	require.ElementsMatch(t, []string{"foo"}, parseReferences("${foo}"))
	require.ElementsMatch(t, []string{"foo", "bar"}, parseReferences("this-${foo}-and-${bar}-x"))
}

func TestReplaceReferences(t *testing.T) {
	require.Equal(t, "this-xyz-and-${bar}-xyz-x", replaceReferences(
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
