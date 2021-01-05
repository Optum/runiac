package tracks_test

import (
	"flag"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.optum.com/healthcarecloud/terrascale/mocks"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"
	"github.optum.com/healthcarecloud/terrascale/pkg/tracks"
	"os"
	"path/filepath"
	"testing"
)

var DefaultStubAccountID string
var StubVersion string

var fs afero.Fs
var logger *logrus.Entry
var stepperFactory steps.StepperFactory
var sut tracks.Tracker

var stubTrackCount int
var stubStepTestsCount int

var stubStepWithTests steps.Step
var stubTracks map[string]tracks.Track
var stubPreTrackName string
var stubTrackNameA string
var stubTrackNameB string

func TestMain(m *testing.M) {
	// arrange

	fs = afero.NewMemMapFs()
	logs := logrus.New()
	logs.SetReportCaller(true)
	logger = logs.WithField("environment", "unittest")

	DefaultStubAccountID = "1"
	StubVersion = "v0.0.5"

	tracks.DestroyTrack = tracks.ExecuteDestroyTrack
	tracks.DeployTrack = tracks.ExecuteDeployTrack
	tracks.DeployTrackRegion = tracks.ExecuteDeployTrackRegion
	tracks.DestroyTrackRegion = tracks.ExecuteDestroyTrackRegion
	tracks.ExecuteStep = tracks.ExecuteStepImpl

	sut = tracks.DirectoryBasedTracker{
		Fs:  fs,
		Log: logger,
	}

	stubStepWithTests = steps.Step{
		Name:                   "a11",
		TestsExist:             true,
		ProgressionLevel:       1,
		TrackName:              "track",
		RegionalResourcesExist: true,
	}

	stubPreTrackName = tracks.PRE_TRACK_NAME
	stubTrackNameA = "track-a"
	stubTrackNameB = "track-b"

	stubTracks = map[string]tracks.Track{
		stubPreTrackName: {
			Name:                  stubPreTrackName,
			StepProgressionsCount: 1,
			OrderedSteps: map[int][]steps.Step{
				1: {
					stubStepWithTests,
					{
						Name:             "pretrackstep",
						TestsExist:       false,
						ProgressionLevel: 1,
					},
				},
			},
		},
		stubTrackNameA: {
			Name:                  stubTrackNameA,
			StepProgressionsCount: 2,
			OrderedSteps: map[int][]steps.Step{
				1: {
					stubStepWithTests,
					{
						Name:             "a12",
						TestsExist:       false,
						ProgressionLevel: 1,
					},
				},
				2: {
					{
						Name:             "a21",
						TestsExist:       false,
						ProgressionLevel: 2,
					},
				},
			},
		},
		stubTrackNameB: {
			Name:                  stubTrackNameB,
			StepProgressionsCount: 1,
			OrderedSteps: map[int][]steps.Step{
				1: {
					{
						Name:             "b11",
						TestsExist:       false,
						ProgressionLevel: 1,
					},
					{
						Name:             "b12",
						TestsExist:       false,
						ProgressionLevel: 1,
					},
				},
			},
		},
	}

	stubTrackCount = len(stubTracks)
	stubStepTestsCount = 0

	for key, track := range stubTracks {
		track.Dir = fmt.Sprintf("tracks/%s", track.Name)

		for progression, stubSteps := range track.OrderedSteps {
			for _, stubStep := range stubSteps {

				stepDir := fmt.Sprintf("%s/step%v_%v", track.Dir, progression, stubStep.Name)
				fs.MkdirAll(stepDir, 0755)

				track.StepsCount++

				if stubStep.TestsExist {
					stubStepTestsCount++
					track.StepsWithTestsCount++
					fs.MkdirAll(fmt.Sprintf("%s/tests", stepDir), 0755)
					_ = afero.WriteFile(fs, fmt.Sprintf("%s/tests/tests.test", stepDir), []byte(`
					faketestbinary
					`), 0644)
				}

				if stubStep.RegionalResourcesExist {
					regionDir := filepath.Join(stepDir, "regional")
					fs.MkdirAll(regionDir, 0755)
					_ = afero.WriteFile(fs, fmt.Sprintf("%s/main.tf", regionDir), []byte(`
					faketestbinary
					`), 0644)
				}

				stubTracks[key] = track
			}
		}
	}

	// act
	flag.Parse()
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

func TestGetTracksWithTargetAll_ShouldReturnCorrectTracks(t *testing.T) {
	// act
	mockTracks := sut.GatherTracks(config.Config{
		TargetAll: true,
	})

	// assert
	require.Equal(t, stubTrackCount, len(mockTracks), "Three tracks should have been gathered")

	// Gather all track names
	var trackNames []string
	var trackAIndex int
	for index, track := range mockTracks {
		trackNames = append(trackNames, track.Name)
		if track.Name == stubTrackNameA {
			trackAIndex = index
		}
	}

	// Ensures all expected tracks are present
	require.Contains(t, trackNames, stubPreTrackName, "The pretrack should be found")
	require.Contains(t, trackNames, stubTrackNameA, "Track A should be found")
	require.Contains(t, trackNames, stubTrackNameB, "Track B should be found")

	// Spot check one of the tracks (Track A)
	require.Equal(t, len(stubTracks[stubTrackNameA].OrderedSteps), mockTracks[trackAIndex].StepProgressionsCount, "StepProgressionsCount should be derived correctly based on steps")
	require.Equal(t, len(stubTracks[stubTrackNameA].OrderedSteps[1]), len(mockTracks[trackAIndex].OrderedSteps[1]), "Track A Step Progression 1 should have 2 step(s)")
	require.Equal(t, len(stubTracks[stubTrackNameA].OrderedSteps[2]), len(mockTracks[trackAIndex].OrderedSteps[2]), "Track A Step Progression 2 should have 1 step(s)")

	for _, track := range mockTracks {
		totalStepSteps := 0
		totalStepsWithTestsCount := 0

		if track.Name == stubPreTrackName {
			require.True(t, track.IsPreTrack, "A track with the name _pretrack should be identified as a pretrack")
		}

		for progressionLevel, steps := range track.OrderedSteps {
			for _, step := range steps {
				totalStepSteps++
				if shouldHaveTests(stubTracks[track.Name].OrderedSteps[progressionLevel], step.Name) {
					require.True(t, step.TestsExist, fmt.Sprintf("Step %v should return true for steps existing", step.Name))
					totalStepsWithTestsCount++
				} else {
					require.False(t, step.TestsExist, fmt.Sprintf("Step %v should return false for tests existing", step.Name))
				}

				if shouldHaveRegionDeployment(stubTracks[track.Name].OrderedSteps[progressionLevel], step.Name) {
					require.True(t, step.RegionalResourcesExist, fmt.Sprintf("Step %v should return true for region deployment", step.Name))
				} else {
					require.False(t, step.RegionalResourcesExist, fmt.Sprintf("Step %v should return false for region deployment", step.Name))
				}
			}

			require.Equal(t, len(stubTracks[track.Name].OrderedSteps[progressionLevel]), len(steps), fmt.Sprintf("track %v progression level %v should have correct step count", track.Name, progressionLevel))

		}

		// important to match for handling channels
		require.Equal(t, totalStepSteps, track.StepsCount, "Track steps count should match total steps in OrderedSteps field")
		require.Equal(t, totalStepsWithTestsCount, track.StepsWithTestsCount, "Track StepsWithTestsCount should match total steps in OrderedSteps field")
		require.Contains(t, stubTracks, track.Name, "Track should be named correctly")
		require.Equal(t, stubTracks[track.Name].StepsWithTestsCount, track.StepsWithTestsCount, "Track StepsWithTestsCount should match total steps in OrderedSteps field")
	}
}

func TestGetTracksWithStepTarget_ShouldReturnCorrectTracks(t *testing.T) {
	stubStepWhitelist := []string{fmt.Sprintf("#core#%s#%s", stubTrackNameA, stubStepWithTests.Name), fmt.Sprintf("#core#%s#%s", stubTrackNameB, "b11")}
	// act
	mockTracks := sut.GatherTracks(config.Config{
		StepWhitelist: stubStepWhitelist,
		Project:         "core",
	})

	// assert
	assert.Equal(t, 2, len(mockTracks), "Two tracks should have been gathered")
	stepCount := 0

	for _, track := range mockTracks {
		for _, steps := range track.OrderedSteps {
			for _, step := range steps {
				stepCount++
				require.Contains(t, stubStepWhitelist, fmt.Sprintf("#core#%s#%s", track.Name, step.Name), "Returned step %s should be in the step whitelist", step.Name)
			}
		}
	}
	// important to match for handling channels
	require.Equal(t, len(stubStepWhitelist), stepCount, "Track steps count should match total steps in defined in whitelist")
}

func shouldHaveTests(s []steps.Step, e string) bool {
	for _, a := range s {
		if a.Name == e {
			return a.TestsExist
		}
	}
	return false
}

func shouldHaveRegionDeployment(s []steps.Step, e string) bool {
	for _, a := range s {
		if a.Name == e {
			return a.RegionalResourcesExist
		}
	}
	return false
}

func TestExecuteTracks_SkipsAllTracksIfPreTrackFails(t *testing.T) {
	stubPrimaryRegion := "primaryregion"

	deployTrackStub := map[string]tracks.Output{
		"_pretrack": {
			Name: "_pretrack",
			Executions: []tracks.RegionExecution{
				{
					Output: tracks.ExecutionOutput{
						Steps: map[string]steps.Step{
							"project_provisioning": {
								Output: steps.StepOutput{
									Status: steps.Fail,
								},
							},
						},
					},
					Region:           stubPrimaryRegion,
					RegionDeployType: steps.PrimaryRegionDeployType,
				},
			},
		},
		"track-a": {
			Name: "track-a",
		},
		"track-b": {
			Name: "track-b",
		},
	}
	deployTrackExecutionSpy := []tracks.Execution{}

	tracks.DeployTrack = func(execution tracks.Execution, cfg config.Config, t tracks.Track, out chan<- tracks.Output) {
		execution.Output.Name = t.Name
		deployTrackExecutionSpy = append(deployTrackExecutionSpy, execution)
		out <- deployTrackStub[t.Name]
		return
	}

	// act
	mockExecution := sut.ExecuteTracks(nil, config.Config{
		TargetAll:   true,
		SelfDestroy: true,
	})

	require.NotNil(t, mockExecution)

	for _, executionSpy := range deployTrackExecutionSpy {
		require.Contains(t, []string{"_pretrack", "track-a", "track-b"}, executionSpy.Output.Name, "Should find the expected tracks")
	}

	for _, tr := range mockExecution.Tracks {
		if tr.Name != tracks.PRE_TRACK_NAME {
			require.True(t, tr.Skipped, "All other tracks should be skipped")
		}
	}
}

func TestExecuteTracks_ShouldHandleRegionalAutoDestroyWithRegionalOutputVariables(t *testing.T) {
	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stubPrimaryStepOutputVars := map[string]map[string]string{
		"step1": {
			"primary": "primary",
		},
	}

	stubRegionalStepOutputVars := map[string]map[string]string{
		"step1": {
			"regional": "regional",
		},
	}

	stubRegionalRegion := "regionalregion"
	stubPrimaryRegion := "primaryregion"

	deployTrackStub := map[string]tracks.Output{
		"_pretrack": {
			Name:                       "_pretrack",
			PrimaryStepOutputVariables: stubPrimaryStepOutputVars,
			Executions: []tracks.RegionExecution{
				{
					Output: tracks.ExecutionOutput{
						StepOutputVariables: stubPrimaryStepOutputVars,
					},
					Region:           stubPrimaryRegion,
					RegionDeployType: steps.PrimaryRegionDeployType,
				},
			},
		},
		"track-a": {
			Name:                       "track-a",
			PrimaryStepOutputVariables: stubPrimaryStepOutputVars,
			Executions: []tracks.RegionExecution{
				{
					Output: tracks.ExecutionOutput{
						StepOutputVariables: stubPrimaryStepOutputVars,
					},
					Region:           stubPrimaryRegion,
					RegionDeployType: steps.PrimaryRegionDeployType,
				},
				{
					Output: tracks.ExecutionOutput{
						StepOutputVariables: stubRegionalStepOutputVars,
					},
					Region:           stubRegionalRegion,
					RegionDeployType: steps.RegionalRegionDeployType,
				},
			},
		},
		"track-b": {
			Name:                       "track-b",
			PrimaryStepOutputVariables: nil,
			Executions:                 nil,
		},
	}
	var destroyTrackASpy tracks.Execution
	deployTrackExecutionSpy := []tracks.Execution{}

	tracks.DeployTrack = func(execution tracks.Execution, cfg config.Config, t tracks.Track, out chan<- tracks.Output) {
		execution.Output.Name = t.Name
		deployTrackExecutionSpy = append(deployTrackExecutionSpy, execution)
		out <- deployTrackStub[t.Name]
		return
	}

	tracks.DestroyTrack = func(execution tracks.Execution, cfg config.Config, t tracks.Track, out chan<- tracks.Output) {
		if t.Name == "track-a" {
			destroyTrackASpy = execution
		}

		out <- tracks.Output{
			Name:                       "",
			PrimaryStepOutputVariables: nil,
			Executions:                 nil,
		}

		return
	}

	// act
	mockExecution := sut.ExecuteTracks(nil, config.Config{
		TargetAll:   true,
		SelfDestroy: true,
	})

	require.NotNil(t, mockExecution)

	for _, executionSpy := range deployTrackExecutionSpy {
		require.Contains(t, []string{"_pretrack", "track-a", "track-b"}, executionSpy.Output.Name, "Should execute correct tracks")
	}
	require.Equal(t, stubPrimaryStepOutputVars, destroyTrackASpy.DefaultExecutionStepOutputVariables[steps.PrimaryRegionDeployType.String()+"-"+stubPrimaryRegion], "Should pass primary step output vars to destroy")
	require.Equal(t, stubRegionalStepOutputVars, destroyTrackASpy.DefaultExecutionStepOutputVariables[steps.RegionalRegionDeployType.String()+"-"+stubRegionalRegion], "Should pass regional region specific step output vars to destroy")
}

func TestExecuteDeployTrack_ShouldExecuteCorrectStepsAndRegions(t *testing.T) {
	// arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stubStepperFactory := mocks.NewMockStepperFactory(ctrl)

	var test = map[string]struct {
		stubExecutedFailCount     int
		stubExecutedTestFailCount int
		stubRegionalDeployment    bool
		regionGroup               string
		stubTargetRegions         []string
		expectedCallCount         int
	}{
		"ShouldExecutePrimaryRegionOnceWithNoRegionalResources": {
			stubExecutedFailCount:     0,
			stubExecutedTestFailCount: 0,
			stubRegionalDeployment:    false,
			regionGroup:               "us",
			stubTargetRegions:         []string{"us-east-1"},
			expectedCallCount:         1,
		},
		"ShouldExecutePrimaryRegionTwiceWithRegionalResources": {
			stubExecutedFailCount:     0,
			stubExecutedTestFailCount: 0,
			stubRegionalDeployment:    true,
			regionGroup:               "us",
			stubTargetRegions:         []string{"us-east-1"},
			expectedCallCount:         2,
		},
		"ShouldExecuteN+PrimaryTimesWithRegionalResourcesTargetingNRegions": {
			stubExecutedFailCount:     0,
			stubExecutedTestFailCount: 0,
			stubRegionalDeployment:    true,
			regionGroup:               "us",
			stubTargetRegions:         []string{"us-east-1", "us-east-2"},
			expectedCallCount:         3,
		},
	}

	executionParams := []tracks.RegionExecution{}

	for name, test := range test {
		t.Run(name, func(t *testing.T) {

			var callCount int
			tracks.DeployTrackRegion = func(in <-chan tracks.RegionExecution, out chan<- tracks.RegionExecution) {
				regionExecution := <-in
				callCount++

				executionParams = append(executionParams, regionExecution)

				regionExecution.Output = tracks.ExecutionOutput{
					FailureCount:        test.stubExecutedFailCount,
					FailedTestCount:     test.stubExecutedTestFailCount,
					StepOutputVariables: regionExecution.DefaultStepOutputVariables,
				}

				regionExecution.Output.StepOutputVariables["test-regional"] = map[string]string{
					"region": regionExecution.Region,
				}

				out <- regionExecution
				return
			}

			trackChan := make(chan tracks.Output, 1)

			// act
			tracks.ExecuteDeployTrack(tracks.Execution{
				Logger:         logger,
				Fs:             fs,
				Output:         tracks.ExecutionOutput{},
				StepperFactory: stubStepperFactory,
			}, config.Config{
				TerrascaleTargetRegions: test.stubTargetRegions,
				TerrascaleRegionGroup:   test.regionGroup,
				CSP:                     "aws",
			}, tracks.Track{
				RegionalDeployment: test.stubRegionalDeployment,
			}, trackChan)

			mockOutput := <-trackChan

			require.Len(t, mockOutput.Executions, callCount)

			for _, exec := range mockOutput.Executions {
				require.Equal(t, exec.Region, exec.Output.StepOutputVariables["test-regional"]["region"], "Region variables should stay scoped to executing region function")
			}

			require.Equal(t, map[string]map[string]string(nil), executionParams[0].Output.StepOutputVariables, "Primary execution should start with no incoming previous step variables")
			require.Equal(t, test.stubExecutedFailCount*callCount, mockOutput.Executions[0].Output.FailureCount, "Should correctly set failed step count")
			require.Equal(t, steps.PrimaryRegionDeployType, executionParams[0].RegionDeployType, "First execution should be primary region")
			require.Equal(t, test.expectedCallCount, callCount, "Should call DeployTrackRegion() the expected amount of times")
		})
	}
}

func TestAddToTrackOutput(t *testing.T) {
	stepOutputVariables := make(map[string]interface{})
	stepOutputVariables["resource_name"] = "my-cool-resource"
	stepOutputVariables["resource_id"] = "resource/my-cool-resource"

	stepOutput := steps.StepOutput{
		OutputVariables: stepOutputVariables,
		StepName:        "cool_step1",
	}

	trackOutputVars := make(map[string]map[string]string)

	mockPrevStepVars := tracks.AppendTrackOutput(trackOutputVars, stepOutput)

	require.Equal(t, "my-cool-resource", mockPrevStepVars[stepOutput.StepName]["resource_name"], "The track output should have the correct key and value set")
	require.Equal(t, "resource/my-cool-resource", mockPrevStepVars[stepOutput.StepName]["resource_id"], "The track output should have the correct key and value set")
}

func TestAppendPreTrackOutputsToDefaultStepOutputVariables_AddsPrimaryRegionExecutionsFromPreTrackToVars(t *testing.T) {
	// Mock existing step output vars
	defaultStepOutputVariables := make(map[string]map[string]string)
	defaultStepOutputVariables["iam"] = map[string]string{
		"var1": "out1",
	}

	preTrackOutputs := &tracks.Output{
		Name: tracks.PRE_TRACK_NAME,
		PrimaryStepOutputVariables: map[string]map[string]string{
			"account": {
				"name": "new-account",
			},
		},
		Executions: []tracks.RegionExecution{
			{
				RegionDeployType: steps.PrimaryRegionDeployType,
				Region:           "centralus",
				Output: tracks.ExecutionOutput{
					StepOutputVariables: map[string]map[string]string{
						"account": {
							"name": "new-account",
						},
					},
				},
			},
			{
				RegionDeployType: steps.RegionalRegionDeployType,
				Region:           "centralus",
				Output: tracks.ExecutionOutput{
					StepOutputVariables: map[string]map[string]string{
						"account": {
							"group": "new-group",
						},
					},
				},
			},
		},
	}

	regionDeployType := steps.PrimaryRegionDeployType
	region := "centralus"

	newDefaultStepOutputVariables := tracks.AppendPreTrackOutputsToDefaultStepOutputVariables(defaultStepOutputVariables, preTrackOutputs, regionDeployType, region)
	require.NotEmpty(t, newDefaultStepOutputVariables, "The new map should not be empty")
	require.Equal(t, 2, len(newDefaultStepOutputVariables), "The map should contain the expected number of keys")

	// Existing step output vars should remain
	iamStepOutVarMap, iamKeyExists := newDefaultStepOutputVariables["iam"]
	require.True(t, iamKeyExists, "The map should still contain the original iam key")
	require.Equal(t, 1, len(iamStepOutVarMap), "The iam step should have the expected number of out vars")
	var1Value, iamKeyVar1KeyExists := iamStepOutVarMap["var1"]
	require.True(t, iamKeyVar1KeyExists, "The original key in iam should still exist")
	require.Equal(t, "out1", var1Value, "The output in the iam step for var1 should be the expected value")

	// Pretrack outputs from the primary region only (since this was a primary region deployment) should be added
	preTrackAccountStepOutVarMap, preTrackKeyExists := newDefaultStepOutputVariables["pretrack-account"]
	require.True(t, preTrackKeyExists, "There should be a key for the pretrack in the expected naming format")
	require.Equal(t, 1, len(preTrackAccountStepOutVarMap), "The pre track account step should have the expected number of out vars")
	accountKeyVal, accountKeyExists := preTrackAccountStepOutVarMap["name"]
	require.True(t, accountKeyExists, "The name output var from the pretrack account step should be added")
	require.Equal(t, "new-account", accountKeyVal, "The name output var from the pretrack account step should have the expected value")
}

func TestAppendPreTrackOutputsToDefaultStepOutputVariables_AddsRegionalExecutionsFromPreTrackToVars(t *testing.T) {
	// Mock existing step output vars
	defaultStepOutputVariables := make(map[string]map[string]string)
	defaultStepOutputVariables["iam"] = map[string]string{
		"var1": "out1",
	}

	preTrackOutputs := &tracks.Output{
		Name: tracks.PRE_TRACK_NAME,
		PrimaryStepOutputVariables: map[string]map[string]string{
			"account": {
				"name": "new-account",
			},
		},
		Executions: []tracks.RegionExecution{
			{
				RegionDeployType: steps.PrimaryRegionDeployType,
				Region:           "centralus",
				Output: tracks.ExecutionOutput{
					StepOutputVariables: map[string]map[string]string{
						"account": {
							"name": "new-account",
						},
					},
				},
			},
			{
				RegionDeployType: steps.RegionalRegionDeployType,
				Region:           "centralus",
				Output: tracks.ExecutionOutput{
					StepOutputVariables: map[string]map[string]string{
						"account": {
							"group": "new-group1-1",
						},
					},
				},
			},
			{
				RegionDeployType: steps.RegionalRegionDeployType,
				Region:           "eastus",
				Output: tracks.ExecutionOutput{
					StepOutputVariables: map[string]map[string]string{
						"account": {
							"group": "new-group-2",
						},
					},
				},
			},
		},
	}

	regionDeployType := steps.RegionalRegionDeployType
	region := "centralus"

	newDefaultStepOutputVariables := tracks.AppendPreTrackOutputsToDefaultStepOutputVariables(defaultStepOutputVariables, preTrackOutputs, regionDeployType, region)
	fmt.Printf("newDefaultStepOutputVariables: %+v\n", newDefaultStepOutputVariables)
	require.NotEmpty(t, newDefaultStepOutputVariables, "The new map should not be empty")
	require.Equal(t, 2, len(newDefaultStepOutputVariables), "The map should contain the expected number of keys")

	// Existing step output vars should remain
	iamStepOutVarMap, iamKeyExists := newDefaultStepOutputVariables["iam"]
	require.True(t, iamKeyExists, "The map should still contain the original iam key")
	require.Equal(t, 1, len(iamStepOutVarMap), "The iam step should have the expected number of out vars")
	var1Value, iamKeyVar1KeyExists := iamStepOutVarMap["var1"]
	require.True(t, iamKeyVar1KeyExists, "The original key in iam should still exist")
	require.Equal(t, "out1", var1Value, "The output in the iam step for var1 should be the expected value")

	// Pretrack outputs from the regional deployment (in the same region) should be added
	preTrackAccountStepOutVarMap, preTrackKeyExists := newDefaultStepOutputVariables["pretrack-account"]
	require.True(t, preTrackKeyExists, "There should be a key for the pretrack in the expected naming format")
	require.Equal(t, 1, len(preTrackAccountStepOutVarMap), "The pre track account step should have the expected number of out vars")
	groupKeyVal, groupKeyExists := preTrackAccountStepOutVarMap["group"]
	require.True(t, groupKeyExists, "The group output var from the pretrack account step should be added")
	require.Equal(t, "new-group1-1", groupKeyVal, "The group output var from the pretrack account step should have the expected value")
}

func TestAppendTrackOutput_WithRegionalStepDeploymentOutput(t *testing.T) {
	stepOutputVariables := make(map[string]interface{})
	stepOutputVariables["resource_name"] = "my-cool-resource"
	stepOutputVariables["resource_id"] = "resource/my-cool-resource"

	stepOutput := steps.StepOutput{
		OutputVariables:  stepOutputVariables,
		StepName:         "cool_step1",
		RegionDeployType: steps.RegionalRegionDeployType,
	}

	trackOutputVars := make(map[string]map[string]string)

	mockPrevStepVars := tracks.AppendTrackOutput(trackOutputVars, stepOutput)

	key := fmt.Sprintf("%s-%s", stepOutput.StepName, steps.RegionalRegionDeployType.String())

	for k, v := range stepOutputVariables {
		require.Equal(t, v, mockPrevStepVars[key][k], "The track output should match the stubbed key value: %s, %s", k, v)
	}
}

type spyExecuteStep struct {
	OutputVars map[string]map[string]string
	StepName   string
}

func TestExecuteDeployTrackRegion_ShouldPassRegionalVariables(t *testing.T) {
	primaryOutChan := make(chan tracks.RegionExecution, 1)
	primaryInChan := make(chan tracks.RegionExecution, 1)

	trackOutputVars := []spyExecuteStep{}

	stubStepP1OutputVars := map[string]interface{}{
		"var": "var",
	}

	tracks.ExecuteStep = func(stepperFactory steps.StepperFactory, region string, regionDeployType steps.RegionDeployType, entry *logrus.Entry, fs afero.Fs, defaultStepOutputVariables map[string]map[string]string, stepProgression int,
		s steps.Step, out chan<- steps.Step, destroy bool) {
		trackOutputVars = append(trackOutputVars, spyExecuteStep{
			OutputVars: defaultStepOutputVariables,
			StepName:   s.Name,
		})

		if s.ProgressionLevel == 1 {
			s.Output.OutputVariables = stubStepP1OutputVars
			s.Output.Status = steps.Success
			s.Output.StepName = s.Name
		}
		s.Output.RegionDeployType = regionDeployType
		s.Output.Region = region
		out <- s
		return
	}

	regionalExecution := tracks.RegionExecution{
		TrackName:                  "",
		TrackDir:                   "",
		TrackStepProgressionsCount: 2,
		TrackStepsWithTestsCount:   0,
		TrackOrderedSteps: map[int][]steps.Step{
			1: {
				{
					Name:                   "step1_p1",
					ProgressionLevel:       1,
					RegionalResourcesExist: true,
				},
				{
					Name:                   "step2_p1",
					ProgressionLevel:       1,
					RegionalResourcesExist: true,
				},
			},
			2: {
				{
					Name:                   "step_p2",
					RegionalResourcesExist: true,
				},
			},
		},
		Logger:           logger,
		Fs:               fs,
		Output:           tracks.ExecutionOutput{},
		Region:           "",
		RegionDeployType: steps.RegionalRegionDeployType,
		StepperFactory:   nil,
		DefaultStepOutputVariables: map[string]map[string]string{
			"step1_p1": {
				"primaryvarkey": "primaryvarvalue",
			},
		},
	}

	go tracks.ExecuteDeployTrackRegion(primaryInChan, primaryOutChan)
	primaryInChan <- regionalExecution

	primaryTrackExecution := <-primaryOutChan

	require.NotNil(t, primaryTrackExecution)

	expectedStepP1OutputVarsStrings := map[string]map[string]string{
		"step1_p1-regional": {
			"var": "var",
		},
		"step2_p1-regional": {
			"var": "var",
		},
		"step1_p1": {
			"primaryvarkey": "primaryvarvalue",
		},
	}

	var trackOutputVarsSpyStepP2 map[string]map[string]string
	for _, outputVars := range trackOutputVars {
		if outputVars.StepName == "step_p2" {
			trackOutputVarsSpyStepP2 = outputVars.OutputVars
		}
	}

	require.Equal(t, expectedStepP1OutputVarsStrings["step1_p1"], trackOutputVarsSpyStepP2["step1_p1"], "Primary execution output vars should be passed to regional")
	require.Equal(t, expectedStepP1OutputVarsStrings["step1_p1-regional"], trackOutputVarsSpyStepP2["step1_p1-regional"], "Output vars should be passed from previous progression steps to current progression steps")
	require.Equal(t, expectedStepP1OutputVarsStrings["step2_p1-regional"], trackOutputVarsSpyStepP2["step2_p1-regional"], "Output vars should be passed from previous progression steps to current progression steps")

}

func TestExecuteDeployTrackRegion_ShouldNotExecuteSecondProgressionWhenFirstFails(t *testing.T) {
	primaryOutChan := make(chan tracks.RegionExecution, 1)
	primaryInChan := make(chan tracks.RegionExecution, 1)

	executeStepSpy := map[string]steps.Step{}

	tracks.ExecuteStep = func(stepperFactory steps.StepperFactory, region string, regionDeployType steps.RegionDeployType, entry *logrus.Entry, fs afero.Fs, defaultStepOutputVariables map[string]map[string]string, stepProgression int,
		s steps.Step, out chan<- steps.Step, destroy bool) {
		executeStepSpy[s.Name] = s

		s.Output = steps.StepOutput{
			Status: steps.Fail,
		}
		out <- s
		return
	}
	regionalExecution := tracks.RegionExecution{
		Logger:                     logger,
		Fs:                         fs,
		Output:                     tracks.ExecutionOutput{},
		TrackStepProgressionsCount: 2,
		TrackOrderedSteps: map[int][]steps.Step{
			1: {
				{
					ID:        "",
					Name:      "step_p1",
					TrackName: "",
					Dir:       "",
				},
			},
			2: {
				{
					ID:        "",
					Name:      "step_p2",
					TrackName: "",
					Dir:       "",
				},
			},
		},
	}

	go tracks.ExecuteDeployTrackRegion(primaryInChan, primaryOutChan)
	primaryInChan <- regionalExecution
	primaryTrackExecution := <-primaryOutChan

	require.NotNil(t, primaryTrackExecution)
	require.Len(t, executeStepSpy, 1, "Should not execute the second progression step with a failure in first progression")
}

func TestExecuteDeployTrackRegion_ShouldSkipWhenPrimaryFails(t *testing.T) {
	primaryOutChan := make(chan tracks.RegionExecution, 1)
	primaryInChan := make(chan tracks.RegionExecution, 1)

	executeStepSpy := map[string]steps.Step{}

	tracks.ExecuteStep = func(stepperFactory steps.StepperFactory, region string, regionDeployType steps.RegionDeployType, entry *logrus.Entry, fs afero.Fs, defaultStepOutputVariables map[string]map[string]string, stepProgression int,
		s steps.Step, out chan<- steps.Step, destroy bool) {
		executeStepSpy[s.Name] = s

		s.Output = steps.StepOutput{
			Status: steps.Fail,
		}
		out <- s
		return
	}

	regionalExecution := tracks.RegionExecution{
		Logger:                     logger,
		Fs:                         fs,
		Output:                     tracks.ExecutionOutput{},
		TrackStepProgressionsCount: 2,
		TrackOrderedSteps: map[int][]steps.Step{
			1: {
				{
					ID:        "",
					Name:      "step_p1",
					TrackName: "",
					Dir:       "",
				},
			},
		},
		PrimaryOutput: tracks.ExecutionOutput{FailureCount: 1},
	}

	go tracks.ExecuteDeployTrackRegion(primaryInChan, primaryOutChan)
	primaryInChan <- regionalExecution
	primaryTrackExecution := <-primaryOutChan

	require.NotNil(t, primaryTrackExecution)
	require.Len(t, executeStepSpy, 0, "Should not execute regional steps when primary region fails")
}

func TestExecuteDeployTrackRegion_ShouldNaWhenRegionalResourcesDoNotExist(t *testing.T) {
	primaryOutChan := make(chan tracks.RegionExecution, 1)
	primaryInChan := make(chan tracks.RegionExecution, 1)

	regionalExecution := tracks.RegionExecution{
		Logger:                     logger,
		Fs:                         fs,
		Output:                     tracks.ExecutionOutput{},
		TrackStepProgressionsCount: 2,
		TrackOrderedSteps: map[int][]steps.Step{
			1: {
				{
					ID:                     "",
					Name:                   "step_p1",
					TrackName:              "",
					Dir:                    "",
					RegionalResourcesExist: false,
				},
			},
		},
		RegionDeployType: steps.RegionalRegionDeployType,
	}

	go tracks.ExecuteDeployTrackRegion(primaryInChan, primaryOutChan)
	primaryInChan <- regionalExecution
	primaryTrackExecution := <-primaryOutChan

	require.NotNil(t, primaryTrackExecution)
	require.Equal(t, steps.Na, primaryTrackExecution.Output.Steps["step_p1"].Output.Status)
}
