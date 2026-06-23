package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvOrReturnsFallbackAndConfiguredValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		key      string
		value    string
		fallback string
		want     string
	}{
		{
			name:     "Given an environment variable is set when envOr is called then returns the configured value",
			key:      "TEST_ENV_OR",
			value:    "value",
			fallback: "fallback",
			want:     "value",
		},
		{
			name:     "Given an environment variable is missing when envOr is called then returns the fallback",
			key:      "TEST_ENV_OR_MISSING",
			value:    "",
			fallback: "fallback",
			want:     "fallback",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.value != "" {
				require.NoError(t, os.Setenv(tc.key, tc.value))
				defer os.Unsetenv(tc.key)
			} else {
				require.NoError(t, os.Unsetenv(tc.key))
			}

			switch tc.want {
			case "value":
				require.Equal(t, "value", envOr(tc.key, tc.fallback))
			case "fallback":
				require.Equal(t, "fallback", envOr(tc.key, tc.fallback))
			default:
				t.Fatalf("unexpected expected value %q", tc.want)
			}
		})
	}
}
