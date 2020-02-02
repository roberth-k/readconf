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
