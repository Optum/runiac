package config

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
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
