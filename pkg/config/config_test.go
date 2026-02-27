package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var yamlExample = []byte(`Hacker: true
name: steve
hobbies:
- skateboarding
- snowboarding
- go
clothing:
  jacket: leather
  trousers: denim
  pants:
    size: large
age: 35
eyes : brown
beard: true
`)

func TestGetConfig_EnvironmentVariablesShouldMatch(t *testing.T) {
	t.Parallel()

	_ = os.Setenv("RUNIAC_PRIMARY_REGION", "centralus")
	_ = os.Setenv("RUNIAC_RUNNER", "terraform")
	_ = os.Setenv("RUNIAC_STEP_WHITELIST", "default/default")
	conf, err := GetConfig()

	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, "centralus", conf.PrimaryRegion)
	require.Equal(t, "terraform", conf.Runner)
	require.NotEmpty(t, conf.StepWhitelist)
	require.Equal(t, "default/default", conf.StepWhitelist[0])
}

func TestConfigStructTags_ShouldBeValid(t *testing.T) {
	t.Parallel()

	rt := reflect.TypeOf(Config{})
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := string(field.Tag)
		if tag == "" {
			continue
		}

		// Check that all key:"value" pairs have properly quoted values
		// A malformed tag like `mapstructure:runner` (missing quotes) will not
		// be parseable by reflect.StructTag.Lookup
		for _, key := range []string{"mapstructure", "required"} {
			if !strings.Contains(tag, key) {
				continue
			}
			_, ok := field.Tag.Lookup(key)
			require.True(t, ok, fmt.Sprintf(
				"struct field %q has malformed %q tag: %s (missing quotes around value?)",
				field.Name, key, tag,
			))
		}
	}
}
