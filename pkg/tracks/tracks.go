package tracks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.optum.com/healthcarecloud/terrascale/pkg/cloudaccountdeployment"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.optum.com/healthcarecloud/terrascale/pkg/config"
	"github.optum.com/healthcarecloud/terrascale/pkg/steps"
	"github.optum.com/healthcarecloud/terrascale/pkg/terraform"
)

const (
	PRE_TRACK_NAME = "_pretrack" // The name of the directory for the pretrack
)

// ExecuteTrackFunc facilitates track executions across multiple regions and RegionDeployTypes (e.g. Primary us-east-1 and regional us-*)
type ExecuteTrackFunc func(execution Execution, cfg config.Config, t Track, out chan<- Output)

// ExecuteTrackRegionFunc executes a track within a single region and RegionDeployType (e.g. primary/us-east-1 or regional/us-east-2)
type ExecuteTrackRegionFunc func(in <-chan RegionExecution, out chan<- RegionExecution)

type ExecuteStepFunc func(stepperFactory steps.StepperFactory, region string, regionDeployType steps.RegionDeployType, entry *logrus.Entry, fs afero.Fs, defaultStepOutputVariables map[string]map[string]string, stepProgression int,
	s steps.Step, out chan<- steps.Step, destroy bool)

var DeployTrackRegion ExecuteTrackRegionFunc = ExecuteDeployTrackRegion
var DestroyTrackRegion ExecuteTrackRegionFunc = ExecuteDestroyTrackRegion

var DeployTrack ExecuteTrackFunc = ExecuteDeployTrack
var DestroyTrack ExecuteTrackFunc = ExecuteDestroyTrack

var ExecuteStep ExecuteStepFunc = ExecuteStepImpl

// Tracker is an interface for working with tracks
type Tracker interface {
	GatherTracks(config config.Config) (tracks []Track)
	ExecuteTracks(stepperFactory steps.StepperFactory, config config.Config) (output Stage)
}

// DirectoryBasedTracker implements the Tracker interface
type DirectoryBasedTracker struct {
	Log *logrus.Entry
	Fs  afero.Fs
}

// Track represents a delivery framework track (unit of functionality)
type Track struct {
	Name                        string
	Dir                         string
	StepProgressionsCount       int
	StepsCount                  int
	StepsWithTestsCount         int
	StepsWithRegionalTestsCount int
	RegionalDeployment          bool // If true at least one step is configured to deploy to multiple region
	OrderedSteps                map[int][]steps.Step
	Output                      Output
	DestroyOutput               Output
	IsPreTrack                  bool // If true, this is a PreTrack, meaning it should be run before all other tracks
}

type Output struct {
	Name                       string
	PrimaryStepOutputVariables map[string]map[string]string
	Executions                 []RegionExecution
}

type Execution struct {
	Logger                              *logrus.Entry
	Fs                                  afero.Fs
	Output                              ExecutionOutput
	StepperFactory                      steps.StepperFactory
	DefaultExecutionStepOutputVariables map[string]map[string]map[string]string
	PreTrackOutput                      *Output
}

type RegionExecution struct {
	TrackName                  string
	TrackDir                   string
	TrackStepProgressionsCount int
	TrackStepsWithTestsCount   int
	TrackOrderedSteps          map[int][]steps.Step
	Logger                     *logrus.Entry
	Fs                         afero.Fs
	Output                     ExecutionOutput
	Region                     string
	RegionDeployType           steps.RegionDeployType
	StepperFactory             steps.StepperFactory
	PrimaryOutput              ExecutionOutput // This value is only set when regiondeploytype == regional
	DefaultStepOutputVariables map[string]map[string]string
}

// TrackOutput represents the output from a track execution
type ExecutionOutput struct {
	Name                string
	Dir                 string
	ExecutedCount       int
	SkippedCount        int
	FailureCount        int
	FailedTestCount     int
	Steps               map[string]steps.Step
	FailedSteps         []steps.Step
	StepOutputVariables map[string]map[string]string // Output variables across all steps in the track. A map where K={step name} and V={map[outputVarName: outputVarVal]}
}

// Stage represents the outputs of tracks
type Stage struct {
	Tracks map[string]Track
}

// GatherTracks gets all tracks that should be executed based
// on the directory structure
func (tracker DirectoryBasedTracker) GatherTracks(config config.Config) (tracks []Track) {
	tracksDir := "./tracks"

	items, _ := afero.ReadDir(tracker.Fs, tracksDir)
	for _, item := range items {
		if item.IsDir() {
			t := Track{
				Name:         item.Name(),
				Dir:          fmt.Sprintf("%s/%s", tracksDir, item.Name()),
				OrderedSteps: map[int][]steps.Step{},
			}

			if t.Name == PRE_TRACK_NAME {
				tracker.Log.Debug("Pretrack found")
				t.IsPreTrack = true
			}

			tConfig := viper.New()
			tConfig.SetConfigName("gaia")               // name of config file (without extension)
			tConfig.AddConfigPath(filepath.Join(t.Dir)) // path to look for the config file in
			if err := tConfig.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					// Config file not found, don't record or log error as this configuration file is optional.
					tracker.Log.Debug("Track is not using a gaia.yaml configuration file")
				} else {
					tracker.Log.WithError(err).Error("Error reading configuration file")
				}
			}

			if tConfig.IsSet("enabled") && !tConfig.GetBool("enabled") {
				tracker.Log.Warningf("Skipping track %s. Not enabled in configuration.", t.Name)
				continue
			}

			// if steps are not being targeted and track are, skip the non-targeted tracks
			if len(config.StepWhitelist) == 0 && !config.TargetAll && !(tConfig.IsSet("enabled") && tConfig.GetBool("enabled")) {
				tracker.Log.Warning(fmt.Sprintf("Tracks: Skipping %s", item.Name()))
				continue
			} else {
				tFolders, _ := afero.ReadDir(tracker.Fs, t.Dir)
				stepPrefix := "step"
				highestProgressionLevel := 0

				for _, tFolder := range tFolders {
					tFolderName := tFolder.Name()

					// step folder convention is step{progressionLevel}_{stepName}
					if strings.HasPrefix(tFolderName, stepPrefix) {
						stepName := tFolderName[len(stepPrefix)+2:]
						stepID := fmt.Sprintf("#%s#%s#%s", config.Stage, t.Name, stepName)

						// if step is not targeted, skip.
						if !contains(config.StepWhitelist, stepID) && !config.TargetAll {
							tracker.Log.Warningf("Step %s disabled. Not present in whitelist.", stepID)
							continue
						}

						v := viper.New()
						v.SetConfigName("gaia")                            // name of config file (without extension)
						v.AddConfigPath(filepath.Join(t.Dir, tFolderName)) // path to look for the config file in
						if err := v.ReadInConfig(); err != nil {
							if _, ok := err.(viper.ConfigFileNotFoundError); ok {
								// Config file not found, don't record or log error as this configuration file is optional.
								tracker.Log.Debug("Step is not using a gaia.yaml configuration file")
							} else {
								tracker.Log.WithError(err).Error("Error reading configuration file")
							}
						}

						if v.IsSet("enabled") && !v.GetBool("enabled") {
							tracker.Log.Warningf("Step %s disabled. Not enabled in configuration.", stepID)
							continue
						}

						parsedStringProgression := string(tFolderName[len(stepPrefix)])
						progressionLevel, err := strconv.Atoi(parsedStringProgression)

						if err != nil {
							tracker.Log.Error(err)
						}

						if progressionLevel > highestProgressionLevel {
							highestProgressionLevel = progressionLevel
						}

						gaiaCfg := &steps.GaiaConfig{}

						err = v.Unmarshal(gaiaCfg)

						if err != nil {
							tracker.Log.WithError(err).Error("Error unmarshaling gaia config")
						}

						step := steps.Step{
							ProgressionLevel: progressionLevel,
							Name:             stepName,
							Dir:              filepath.Join(t.Dir, tFolderName),
							DeployConfig:     config,
							TrackName:        t.Name,
							ID:               stepID,
							GaiaConfig:       *gaiaCfg,
						}

						step.TestsExist = fileExists(tracker.Fs, filepath.Join(step.Dir, "tests/tests.test"))
						step.RegionalResourcesExist = exists(tracker.Fs, filepath.Join(step.Dir, "regional"))

						if step.RegionalResourcesExist {
							step.RegionalTestsExist = fileExists(tracker.Fs, filepath.Join(step.Dir, "regional", "tests/tests.test"))
						}

						tracker.Log.Infof("Adding Step %s. Tests Exist: %v. Regional Resources Exist: %v. Regional Tests Exist: %v.", stepID, step.TestsExist, step.RegionalResourcesExist, step.RegionalTestsExist)

						// let track know it needs to execute regionally as well
						if !t.RegionalDeployment && step.RegionalResourcesExist {
							t.RegionalDeployment = true
						}

						t.OrderedSteps[progressionLevel] = append(t.OrderedSteps[progressionLevel], step)
						t.StepsCount++

						if step.TestsExist {
							t.StepsWithTestsCount++
						}

						if step.RegionalTestsExist {
							t.StepsWithRegionalTestsCount++
						}
					}
				}
				t.StepProgressionsCount = highestProgressionLevel
			}
			if t.StepsCount > 0 {
				tracker.Log.Println(fmt.Sprintf("Tracks: Adding %s", item.Name()))
				tracks = append(tracks, t)
			}
		}
	}

	return
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(fs afero.Fs, filename string) bool {
	info, err := fs.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// isEmpty checks if a file or dir exists and is not empty
func exists(fs afero.Fs, filename string) bool {
	info, err := afero.IsEmpty(fs, filename)
	return err == nil && !info
}

// ExecuteTracks executes all tracks in parallel.
// If a _pretrack exists, this is executed before
// all other tracks.
func (tracker DirectoryBasedTracker) ExecuteTracks(stepperFactory steps.StepperFactory, cfg config.Config) (output Stage) {
	output.Tracks = map[string]Track{}
	var tracks = tracker.GatherTracks(cfg) // **All** tracks
	var parallelTracks []Track             // Tracks that should be executed in parallel

	// Pre track
	var preTrackExists bool
	var preTrack Track

	for _, t := range tracks {
		output.Tracks[t.Name] = t
		if t.IsPreTrack {
			preTrackExists = true
			preTrack = t
		} else {
			parallelTracks = append(parallelTracks, t)
		}
	}

	// Execute _pretrack if it exists
	if preTrackExists {
		tracker.Log.Debug("Pre track found. Must execute before all other tracks")

		preTrackChan := make(chan Output)
		preTrackExecution := Execution{
			Logger:                              tracker.Log,
			Fs:                                  tracker.Fs,
			Output:                              ExecutionOutput{},
			StepperFactory:                      stepperFactory,
			DefaultExecutionStepOutputVariables: map[string]map[string]map[string]string{},
		}
		go DeployTrack(preTrackExecution, cfg, preTrack, preTrackChan)
		// Wait for the track to contain an item,
		// indicating the track has completed.
		preTrackOutput := <-preTrackChan
		preTrack.Output = preTrackOutput
		output.Tracks[preTrack.Name] = preTrack
		tracker.Log.Debug("Pre track finished")
		tracker.Log.Debugf("Pre track output: %+v", preTrack.Output)
		for _, exec := range preTrackOutput.Executions {
			for _, step := range exec.Output.Steps {
				if step.Output.Status == steps.Fail {
					tracker.Log.Debugf("Step: %s failed", step.Name)
					tracker.Log.Debug("A step in the pretrack failed. Cannot continue.")
					return
				}
			}
		}
		// TODO: Exit early if pretrack fails
		// TODO: Track number reporting
	}

	// Execute non pre/post tracks in parallel
	numParallelTracks := len(parallelTracks)
	parallelTrackChan := make(chan Output)

	// execute all tracks concurrently
	// within ExecuteDeployTrack, track result will be added to trackChan feeding next loop
	for _, t := range parallelTracks {
		execution := Execution{
			Logger:                              tracker.Log,
			Fs:                                  tracker.Fs,
			Output:                              ExecutionOutput{},
			StepperFactory:                      stepperFactory,
			DefaultExecutionStepOutputVariables: map[string]map[string]map[string]string{},
		}
		// If there is a pretrack, add its outputs
		// to the execution so they are available.
		if preTrackExists {
			execution.PreTrackOutput = &preTrack.Output
		}
		go DeployTrack(execution, cfg, t, parallelTrackChan)
	}

	// wait for all executions to finish (this loop matches above range)
	for tExecution := 0; tExecution < numParallelTracks; tExecution++ {
		// waiting to append <-trackChan Track N times will inherently wait for all above executions to finish
		tOutput := <-parallelTrackChan
		if t, ok := output.Tracks[tOutput.Name]; ok {
			// TODO: is it better to have a pointer for map value?
			t.Output = tOutput
			output.Tracks[tOutput.Name] = t
		}
	}

	// If SelfDestroy or Destroy is set (e.g. during PRs), destroy any resources created by the tracks
	if cfg.SelfDestroy && !cfg.DryRun {
		tracker.Log.Info("Executing destroy...")
		trackDestroyChan := make(chan Output)

		for _, t := range parallelTracks {
			executionStepOutputVariables := map[string]map[string]map[string]string{}

			for _, exec := range output.Tracks[t.Name].Output.Executions {
				executionStepOutputVariables[fmt.Sprintf("%s-%s", exec.RegionDeployType, exec.Region)] = exec.Output.StepOutputVariables
			}

			if tracker.Log.Level == logrus.DebugLevel {
				jsonBytes, _ := json.Marshal(executionStepOutputVariables)

				tracker.Log.Debugf("OUTPUT VARS: %s", string(jsonBytes))
			}

			execution := Execution{
				Logger:                              tracker.Log,
				Fs:                                  tracker.Fs,
				Output:                              ExecutionOutput{},
				StepperFactory:                      stepperFactory,
				DefaultExecutionStepOutputVariables: executionStepOutputVariables,
			}
			// If there is a pretrack, add its outputs
			// to the execution so they are available.
			if preTrackExists {
				execution.PreTrackOutput = &preTrack.Output
			}
			go DestroyTrack(execution, cfg, t, trackDestroyChan)
		}

		// wait for all executions to finish (this loop matches above range)
		for range parallelTracks {
			// waiting to append <-trackDestroyChan Track N times will inherently wait for all above executions to finish
			tDestroyOutout := <-trackDestroyChan

			if t, ok := output.Tracks[tDestroyOutout.Name]; ok {
				// TODO: is it better to have a pointer for map value?
				t.DestroyOutput = tDestroyOutout
				output.Tracks[tDestroyOutout.Name] = t
			}
		}

		// Destroy _pretrack if it exists
		if preTrackExists {
			tracker.Log.Debug("Pre track found. Must be destroyed after all other tracks")
			executionStepOutputVariables := map[string]map[string]map[string]string{}

			tracker.Log.Debugf("Pretrack executions: %+v", output.Tracks[preTrack.Name].Output.Executions)
			for _, exec := range output.Tracks[preTrack.Name].Output.Executions {
				executionStepOutputVariables[fmt.Sprintf("%s-%s", exec.RegionDeployType, exec.Region)] = exec.Output.StepOutputVariables
			}

			destroyPreTrackChan := make(chan Output)
			preTrackDestroyExecution := Execution{
				Logger:                              tracker.Log,
				Fs:                                  tracker.Fs,
				Output:                              ExecutionOutput{},
				StepperFactory:                      stepperFactory,
				DefaultExecutionStepOutputVariables: executionStepOutputVariables,
				PreTrackOutput:                      &preTrack.Output,
			}
			go DestroyTrack(preTrackDestroyExecution, cfg, preTrack, destroyPreTrackChan)
			// Wait for the track to contain an item,
			// indicating the track has been destroyed.
			preTrackDestroyOutput := <-destroyPreTrackChan
			preTrack.DestroyOutput = preTrackDestroyOutput
			tracker.Log.Debug("Pre track destroy finished")
			tracker.Log.Debugf("Pre track destroy output: %+v", preTrack.Output)
			// TODO: Exit early if pretrack fails
			// TODO: Track number reporting
		}
	}

	return
}

// Adds step outputs variables to the track output variables map
// K = Step Name, V = map[StepOutputVarName: StepOutputVarValue]
func AppendTrackOutput(trackOutputVariables map[string]map[string]string, output steps.StepOutput) map[string]map[string]string {

	key := output.StepName

	if output.RegionDeployType == steps.RegionalRegionDeployType {
		key = fmt.Sprintf("%s-%s", key, output.RegionDeployType.String())
	}

	if trackOutputVariables[key] == nil {
		trackOutputVariables[key] = make(map[string]string)
	}

	for k, v := range output.OutputVariables {
		trackOutputVariables[key][k] = terraform.OutputToString(v)
	}

	return trackOutputVariables
}

func AppendPreTrackOutputsToDefaultStepOutputVariables(defaultStepOutputVariables map[string]map[string]string, preTrackOutput *Output, regionDeployType steps.RegionDeployType, region string) map[string]map[string]string {
	for _, execution := range preTrackOutput.Executions {
		if execution.RegionDeployType == regionDeployType && execution.Region == region {
			for step, outputVarMap := range execution.Output.StepOutputVariables {
				for outVarName, outVarVal := range outputVarMap {
					key := fmt.Sprintf("pretrack-%s", step)

					// Check if the key already exists
					if _, ok := defaultStepOutputVariables[key]; ok {
						defaultStepOutputVariables[key][outVarName] = outVarVal
					} else {
						defaultStepOutputVariables[key] = map[string]string{
							outVarName: outVarVal,
						}
					}
				}
			}
		}
	}

	return defaultStepOutputVariables
}

// ExecuteDeployTrack is for executing a single track across regions
func ExecuteDeployTrack(execution Execution, cfg config.Config, t Track, out chan<- Output) {
	logger := execution.Logger.WithFields(logrus.Fields{
		"track":  t.Name,
		"action": "deploy",
	})

	output := Output{
		Name:                       t.Name,
		Executions:                 []RegionExecution{},
		PrimaryStepOutputVariables: map[string]map[string]string{},
	}

	primaryOutChan := make(chan RegionExecution, 1)
	primaryInChan := make(chan RegionExecution, 1)

	primaryRegionExecution := RegionExecution{
		TrackName:                  t.Name,
		TrackDir:                   t.Dir,
		TrackStepProgressionsCount: t.StepProgressionsCount,
		TrackStepsWithTestsCount:   t.StepsWithTestsCount,
		TrackOrderedSteps:          t.OrderedSteps,
		Logger:                     logger,
		Fs:                         execution.Fs,
		Output:                     ExecutionOutput{},
		Region:                     cfg.GetPrimaryRegionByCSP(cfg.CSP),
		RegionDeployType:           steps.PrimaryRegionDeployType,
		StepperFactory:             execution.StepperFactory,
		DefaultStepOutputVariables: map[string]map[string]string{},
	}

	if val, ok := execution.DefaultExecutionStepOutputVariables[fmt.Sprintf("%s-%s", primaryRegionExecution.RegionDeployType, primaryRegionExecution.Region)]; ok {
		primaryRegionExecution.DefaultStepOutputVariables = val
	}

	// Add step outputs for primary steps
	// from the pretrack
	if execution.PreTrackOutput != nil {
		logger.Debug("Adding pretrack step outputs to primary region execution")
		primaryRegionExecution.DefaultStepOutputVariables = AppendPreTrackOutputsToDefaultStepOutputVariables(primaryRegionExecution.DefaultStepOutputVariables, execution.PreTrackOutput, primaryRegionExecution.RegionDeployType, primaryRegionExecution.Region)
		logger.Debugf("PrimaryRegionExecution DefaultStepOutputVariables are: %+v", primaryRegionExecution.DefaultStepOutputVariables)
	}

	go DeployTrackRegion(primaryInChan, primaryOutChan)
	primaryInChan <- primaryRegionExecution

	primaryTrackExecution := <-primaryOutChan
	output.Executions = append(output.Executions, primaryTrackExecution)
	output.PrimaryStepOutputVariables = primaryTrackExecution.Output.StepOutputVariables

	// end early if track has no regional step resources
	if !t.RegionalDeployment {
		logger.Info("Track has no regional resources, completing track.")
		_, err := cloudaccountdeployment.FlushTrack(logger, t.Name)

		if err != nil {
			logger.WithError(err).Error(err)
		}

		out <- output
		return
	}

	targetRegions := cfg.GaiaTargetRegions
	targetRegionsCount := len(targetRegions)
	regionOutChan := make(chan RegionExecution, targetRegionsCount)
	regionInChan := make(chan RegionExecution, targetRegionsCount)

	logger.Infof("Primary region successfully completed, executing regional deployments in %v.", targetRegions)

	for i := 0; i < targetRegionsCount; i++ {
		go DeployTrackRegion(regionInChan, regionOutChan)
	}

	for _, reg := range targetRegions {
		outputVars := map[string]map[string]string{}

		// Like slices, maps hold references to an underlying data structure. If you pass a map to a function that changes the contents of the map, the changes will be visible in the caller.
		// https://golang.org/doc/effective_go.html#maps
		// While map is being used for StepOutputVariables, required to copy value to a new map to avoid regions overwriting each other while inflight regional step variables are added
		for k, v := range primaryTrackExecution.Output.StepOutputVariables {
			outputVars[k] = v
		}

		regionalRegionExecution := RegionExecution{
			TrackName:                  t.Name,
			TrackDir:                   t.Dir,
			TrackStepProgressionsCount: t.StepProgressionsCount,
			TrackStepsWithTestsCount:   t.StepsWithRegionalTestsCount,
			TrackOrderedSteps:          t.OrderedSteps,
			Logger:                     logger,
			Fs:                         execution.Fs,
			Output:                     ExecutionOutput{},
			Region:                     reg,
			RegionDeployType:           steps.RegionalRegionDeployType,
			StepperFactory:             execution.StepperFactory,
			DefaultStepOutputVariables: outputVars,
			PrimaryOutput:              primaryTrackExecution.Output,
		}

		// Add step outputs for primary steps
		// from the pretrack and also the regional
		// outputs from the pretrack
		if execution.PreTrackOutput != nil {
			logger.Debugf("Adding pretrack step outputs to %s region execution", regionalRegionExecution.Region)
			regionalRegionExecution.DefaultStepOutputVariables = AppendPreTrackOutputsToDefaultStepOutputVariables(regionalRegionExecution.DefaultStepOutputVariables, execution.PreTrackOutput, regionalRegionExecution.RegionDeployType, regionalRegionExecution.Region)
			logger.Debugf("RegionalRegionExecution DefaultStepOutputVariables are: %+v", regionalRegionExecution.DefaultStepOutputVariables)
		}

		regionInChan <- regionalRegionExecution
	}

	for i := 0; i < targetRegionsCount; i++ {
		regionTrackOutput := <-regionOutChan
		output.Executions = append(output.Executions, regionTrackOutput)
	}

	stepExecutions, err := cloudaccountdeployment.FlushTrack(logger, t.Name)

	if err != nil {
		logger.WithError(err).Error(err)
	}

	if logger.Level == logrus.DebugLevel {
		json, _ := json.Marshal(stepExecutions)

		logger.Debug(string(json))
	}

	out <- output
}

// ExecuteDestroyTrack is a helper function for destroying a track
func ExecuteDestroyTrack(execution Execution, cfg config.Config, t Track, out chan<- Output) {
	trackLogger := execution.Logger.WithFields(logrus.Fields{
		"track":  t.Name,
		"action": "destroy",
	})

	output := Output{
		Name:       t.Name,
		Executions: []RegionExecution{},
	}

	// TODO(high): need to gather previous step variables before attempting to destroy!

	// start with regional if existing
	if t.RegionalDeployment {
		regionOutChan := make(chan RegionExecution)
		regionInChan := make(chan RegionExecution)

		targetRegions := cfg.GaiaTargetRegions
		targetRegionsCount := len(cfg.GaiaTargetRegions)

		for i := 0; i < targetRegionsCount; i++ {
			go DestroyTrackRegion(regionInChan, regionOutChan)
		}

		for _, reg := range targetRegions {
			regionExecution := RegionExecution{
				TrackName:                  t.Name,
				TrackDir:                   t.Dir,
				TrackStepProgressionsCount: t.StepProgressionsCount,
				TrackOrderedSteps:          t.OrderedSteps,
				Logger:                     trackLogger,
				Fs:                         execution.Fs,
				Output:                     ExecutionOutput{},
				Region:                     reg,
				RegionDeployType:           steps.RegionalRegionDeployType,
				StepperFactory:             execution.StepperFactory,
				DefaultStepOutputVariables: execution.DefaultExecutionStepOutputVariables[fmt.Sprintf("%s-%s", steps.RegionalRegionDeployType, reg)],
			}

			// Add step outputs for primary steps
			// from the pretrack and also the regional
			// outputs from the pretrack
			if execution.PreTrackOutput != nil {
				trackLogger.Debugf("Adding pretrack step outputs to %s region destroy", regionExecution.Region)
				regionExecution.DefaultStepOutputVariables = AppendPreTrackOutputsToDefaultStepOutputVariables(regionExecution.DefaultStepOutputVariables, execution.PreTrackOutput, regionExecution.RegionDeployType, regionExecution.Region)
				trackLogger.Debugf("RegionalRegionExecution DefaultStepOutputVariables for destroy are: %+v", regionExecution.DefaultStepOutputVariables)
			}

			regionInChan <- regionExecution
		}

		for i := 0; i < targetRegionsCount; i++ {
			regionTrackOutput := <-regionOutChan
			output.Executions = append(output.Executions, regionTrackOutput)
		}
	}

	// clean up primary
	primaryOutChan := make(chan RegionExecution, 1)
	primaryInChan := make(chan RegionExecution, 1)

	primaryExecution := RegionExecution{
		TrackName:                  t.Name,
		TrackDir:                   t.Dir,
		TrackStepProgressionsCount: t.StepProgressionsCount,
		TrackOrderedSteps:          t.OrderedSteps,
		Logger:                     trackLogger,
		Fs:                         execution.Fs,
		Output:                     ExecutionOutput{},
		Region:                     cfg.GetPrimaryRegionByCSP(cfg.CSP),
		RegionDeployType:           steps.PrimaryRegionDeployType,
		StepperFactory:             execution.StepperFactory,
		DefaultStepOutputVariables: execution.DefaultExecutionStepOutputVariables[fmt.Sprintf("%s-%s", steps.PrimaryRegionDeployType, cfg.GetPrimaryRegionByCSP(cfg.CSP))],
	}

	// Add step outputs for primary steps
	// from the pretrack
	if execution.PreTrackOutput != nil {
		trackLogger.Debug("Adding pretrack step outputs to primary region destroy")
		primaryExecution.DefaultStepOutputVariables = AppendPreTrackOutputsToDefaultStepOutputVariables(primaryExecution.DefaultStepOutputVariables, execution.PreTrackOutput, primaryExecution.RegionDeployType, primaryExecution.Region)
		trackLogger.Debugf("PrimaryRegionExecution DefaultStepOutputVariables for destroy are: %+v", primaryExecution.DefaultStepOutputVariables)
	}

	go DestroyTrackRegion(primaryInChan, primaryOutChan)
	primaryInChan <- primaryExecution

	primaryTrackOutput := <-primaryOutChan
	output.Executions = append(output.Executions, primaryTrackOutput)

	out <- output
}

func ExecuteDeployTrackRegion(in <-chan RegionExecution, out chan<- RegionExecution) {
	execution := <-in
	logger := execution.Logger.WithFields(logrus.Fields{
		"region":           execution.Region,
		"regionDeployType": execution.RegionDeployType.String(),
	})

	execution.Output = ExecutionOutput{
		Name:                execution.TrackName,
		Dir:                 execution.TrackDir,
		Steps:               map[string]steps.Step{},
		StepOutputVariables: execution.DefaultStepOutputVariables,
	}

	if execution.Output.StepOutputVariables == nil {
		execution.Output.StepOutputVariables = map[string]map[string]string{}
	}

	// define test channel outside of stepProgression loop to allow tests to run in background while steps proceed through progressions
	testOutChan := make(chan steps.StepTestOutput)
	testInChan := make(chan steps.Step)

	// Create testing goroutines.
	for testExecution := 0; testExecution < execution.TrackStepsWithTestsCount; testExecution++ {
		go executeStepTest(logger, execution.Fs, execution.StepperFactory, execution.Region, execution.RegionDeployType, execution.Output.StepOutputVariables, testInChan, testOutChan)
	}

	for progressionLevel := 1; progressionLevel <= execution.TrackStepProgressionsCount; progressionLevel++ {
		sChan := make(chan steps.Step)
		for _, s := range execution.TrackOrderedSteps[progressionLevel] {

			// regional resources do not exist
			if execution.RegionDeployType == steps.RegionalRegionDeployType && !s.RegionalResourcesExist {
				go func(s steps.Step) {
					s.Output.Status = steps.Na
					sChan <- s
				}(s)
				// if any previous failures, skip
			} else if progressionLevel > 1 && execution.Output.FailureCount > 0 {
				go func(s steps.Step, logger *logrus.Entry) {
					slogger := logger.WithFields(logrus.Fields{
						"step": s.Name,
					})

					slogger.Warn("Skipping step due to earlier step failures in this region")

					s.Output.Status = steps.Skipped
					sChan <- s
				}(s, logger)
			} else if execution.PrimaryOutput.FailureCount > 0 {
				go func(s steps.Step, logger *logrus.Entry) {
					slogger := logger.WithFields(logrus.Fields{
						"step": s.Name,
					})

					slogger.Warn("Skipping step due to failures in primary region deployment")

					s.Output.Status = steps.Skipped
					sChan <- s
				}(s, logger)
			} else {
				go ExecuteStep(execution.StepperFactory, execution.Region, execution.RegionDeployType, logger, execution.Fs, execution.Output.StepOutputVariables, progressionLevel, s, sChan, false)
			}
		}

		N := len(execution.TrackOrderedSteps[progressionLevel])
		for i := 0; i < N; i++ {
			s := <-sChan
			if s.Output.Status == steps.Skipped {
				execution.Output.SkippedCount++
			} else {
				execution.Output.ExecutedCount++
			}
			execution.Output.Steps[s.Name] = s
			execution.Output.StepOutputVariables = AppendTrackOutput(execution.Output.StepOutputVariables, s.Output)

			if s.Output.Err != nil || s.Output.Status == steps.Fail {
				execution.Output.FailureCount++
				execution.Output.FailedSteps = append(execution.Output.FailedSteps, s)
			}

			// trigger tests if exist, this number needs to match testing goroutines triggered above
			// further filtering happens after trigger
			if execution.RegionDeployType == steps.RegionalRegionDeployType && s.RegionalTestsExist {
				logger.Debug("Triggering tests")
				testInChan <- s
			} else if execution.RegionDeployType == steps.PrimaryRegionDeployType && s.TestsExist {
				logger.Debug("Triggering tests")
				testInChan <- s
			}
		}
	}

	for testExecution := 0; testExecution < execution.TrackStepsWithTestsCount; testExecution++ {
		s := <-testOutChan

		// add test output to trackOut
		if val, ok := execution.Output.Steps[s.StepName]; ok {
			// TODO: is it better to have a pointer for map value?
			val.TestOutput = s
			execution.Output.Steps[s.StepName] = val
		}

		// TODO: avoid this loop with FailedSteps
		for i := range execution.Output.FailedSteps {
			if execution.Output.FailedSteps[i].Name == s.StepName {
				execution.Output.FailedSteps[i].TestOutput = s
			}
		}

		if s.Err != nil {
			execution.Output.FailedTestCount++
		}
	}

	out <- execution
}

func ExecuteDestroyTrackRegion(in <-chan RegionExecution, out chan<- RegionExecution) {
	execution := <-in

	logger := execution.Logger.WithFields(logrus.Fields{
		"region":           execution.Region,
		"regionDeployType": execution.RegionDeployType.String(),
	})

	execution.Output = ExecutionOutput{
		Name:                execution.TrackName,
		Dir:                 execution.TrackDir,
		Steps:               map[string]steps.Step{},
		StepOutputVariables: execution.DefaultStepOutputVariables,
	}

	for i := execution.TrackStepProgressionsCount; i >= 1; i-- {
		sChan := make(chan steps.Step)
		for progressionLevel, s := range execution.TrackOrderedSteps[i] {
			// if any previous failures, skip
			if (progressionLevel > 1 && execution.Output.FailureCount > 0) || (execution.RegionDeployType == steps.RegionalRegionDeployType && !s.RegionalResourcesExist) {
				go func(s steps.Step) {
					s.Output.Status = steps.Skipped
					sChan <- s
				}(s)
			} else {
				go ExecuteStep(execution.StepperFactory, execution.Region, execution.RegionDeployType, logger, execution.Fs, execution.Output.StepOutputVariables, i, s, sChan, true)
			}
		}
		N := len(execution.TrackOrderedSteps[i])
		for i := 0; i < N; i++ {
			s := <-sChan
			if s.Output.Status == steps.Skipped {
				execution.Output.SkippedCount++
			} else {
				execution.Output.ExecutedCount++
			}
			execution.Output.Steps[s.Name] = s

			if s.Output.Err != nil {
				execution.Output.FailureCount++
				execution.Output.FailedSteps = append(execution.Output.FailedSteps, s)
			}
		}
	}

	out <- execution
	return
}

func ExecuteStepImpl(stepperFactory steps.StepperFactory, region string, regionDeployType steps.RegionDeployType,
	logger *logrus.Entry, fs afero.Fs, defaultStepOutputVariables map[string]map[string]string, stepProgression int,
	s steps.Step, out chan<- steps.Step, destroy bool) {

	stepper := stepperFactory.Get(s)

	exec, err := s.InitExecution(logger, fs, regionDeployType, region, defaultStepOutputVariables)

	// if error initializing, short circuit
	if err != nil {
		s.Output = steps.StepOutput{
			Status:           steps.Fail,
			RegionDeployType: regionDeployType,
			Region:           region,
			StepName:         s.Name,
			StreamOutput:     "",
			Err:              err,
			OutputVariables:  nil,
		}
		out <- s
		return
	}

	var output steps.StepOutput

	if destroy {
		output = stepper.ExecuteStepDestroy(exec)
	} else {
		output = stepper.ExecuteStep(exec)
	}

	s.Output = output

	out <- s
	return
}

func executeStepTest(incomingLogger *logrus.Entry, fs afero.Fs, stepperFactory steps.StepperFactory, region string, regionDeployType steps.RegionDeployType, defaultStepOutputVariables map[string]map[string]string, in <-chan steps.Step, out chan<- steps.StepTestOutput) {
	s := <-in
	tOutput := steps.StepTestOutput{}

	logger := incomingLogger.WithFields(logrus.Fields{
		"step":            s.Name,
		"stepProgression": s.ProgressionLevel,
		"action":          "test",
	})

	logger.Info("Starting Step Tests")

	// only run step tests when they exist and deployment was error free
	if s.Output.Err != nil || s.Output.Status == steps.Fail {
		logger.Warn("Skipping Tests Due to Deployment Error")
	} else if s.DeployConfig.DryRun {
		logger.Info("Skipping Tests for Dry Run")
	} else if s.Output.Status == steps.Skipped {
		logger.Warn("Skipping Tests because step was also skipped")
	} else {
		logger.Info("Triggering Step Tests")
		stepper := stepperFactory.Get(s)
		exec, err := s.InitExecution(logger, fs, regionDeployType, region, defaultStepOutputVariables)

		// if err initializing, short circuit
		if err != nil {
			tOutput = steps.StepTestOutput{
				StepName:     s.Name,
				StreamOutput: "",
				Err:          err,
			}

			out <- tOutput
			return
		}

		tOutput = stepper.ExecuteStepTests(exec)

		if tOutput.Err != nil {
			logger.WithError(tOutput.Err).Error("Error executing tests for step")
		}
	}

	out <- tOutput
	return
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.ToLower(a) == strings.ToLower(e) {
			return true
		}
	}
	return false
}
