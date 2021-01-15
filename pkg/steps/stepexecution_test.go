package steps

import (
	"testing"

	"github.optum.com/healthcarecloud/terrascale/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

var sut config.Stepper
var logger = logrus.NewEntry(logrus.New())
var DefaultStubAccountID = "1"

func TestNewExecution_ShouldSetFields(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	stubRegion := "region"
	stubRegionalDeployType := config.RegionalRegionDeployType

	stubStep := config.Step{
		Dir:  "stub",
		Name: "stubName",
		DeployConfig: config.Config{
			DeploymentRing:            "stubDeploymentRing",
			Project:                   "stubProject",
			DryRun:                    true,
			RegionalRegions:           []string{"stub"},
			UniqueExternalExecutionID: "stubExecutionID",
			MaxRetries:                3,
			MaxTestRetries:            2,
		},
		TrackName: "stubTrackName",
	}
	// act
	mock := NewExecution(stubStep, logger, fs, stubRegionalDeployType, stubRegion, map[string]map[string]string{})

	// assert
	require.Equal(t, stubStep.Dir, mock.Dir, "Dir should match stub value")
	require.Equal(t, stubStep.Name, mock.StepName, "Name should match stub value")
	require.Equal(t, stubRegion, mock.Region, "Region should match stub value")
	require.Equal(t, stubRegionalDeployType, mock.RegionDeployType, "RegionDeployType should match stub value")
	require.Equal(t, stubStep.DeployConfig.DeploymentRing, mock.DeploymentRing, "DeploymentRing should match stub value")
	require.Equal(t, stubStep.DeployConfig.Project, mock.Project, "Project should match stub value")
	require.Equal(t, stubStep.DeployConfig.DryRun, mock.DryRun, "DryRun should match stub value")
	require.Equal(t, stubStep.TrackName, mock.TrackName, "TrackName should match stub value")
	require.Equal(t, stubStep.DeployConfig.UniqueExternalExecutionID, mock.UniqueExternalExecutionID, "UniqueExternalExecutionID should match stub value")
	require.Equal(t, stubStep.DeployConfig.RegionalRegions, mock.RegionGroupRegions, "RegionGroupRegions should match stub value")
	require.Equal(t, stubStep.DeployConfig.MaxRetries, mock.MaxRetries, "MaxRetries should match stub value")
	require.Equal(t, stubStep.DeployConfig.MaxTestRetries, mock.MaxTestRetries, "MaxTestRetries should match stub value")

}
