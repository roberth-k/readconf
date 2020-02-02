package configkit

import (
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

func TestExtractStructFields(t *testing.T) {
	type Embedded struct {
		Bar string
	}

	var theStruct struct {
		Inner  string
		Nested struct {
			Foo string
		}
		ignored int
		Embedded
	}

	t.Run("must be a pointer", func(t *testing.T) {
		m, err := extractStructFields(theStruct)
		require.Errorf(t, err, "expected pointer to struct")
		require.Empty(t, m)

		m, err = extractStructFields(&theStruct)
		require.NoError(t, err)
		require.NotEmpty(t, m)
	})

	t.Run("field map of an empty struct", func(t *testing.T) {
		var s struct{}
		m, err := extractStructFields(&s)
		require.NoError(t, err)
		require.Empty(t, m.Keys())
	})

	t.Run("field map", func(t *testing.T) {
		m, err := extractStructFields(&theStruct)
		require.NoError(t, err)
		require.Equal(t, []string{
			`BAR`,
			`INNER`,
			`NESTED__FOO`,
		}, m.Keys())
	})
}
