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
							tracker.Log.Warningf("Skipping Step %s. Not present in whitelist.", stepID)
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
							tracker.Log.Warningf("Skipping Step %s. Not enabled in configuration.", stepID)
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

// ExecuteTracks executes all tracks in parallel
func (tracker DirectoryBasedTracker) ExecuteTracks(stepperFactory steps.StepperFactory, cfg config.Config) (output Stage) {
	output.Tracks = map[string]Track{}
	var executingTracks = tracker.GatherTracks(cfg)

	for _, t := range executingTracks {
		output.Tracks[t.Name] = t
	}

	numTracks := len(executingTracks)
	trackChan := make(chan Output)

	// execute all tracks concurrently
	// within ExecuteDeployTrack, track result will be added to trackChan feeding next loop
	for _, t := range executingTracks {
		execution := Execution{
			Logger:                              tracker.Log,
			Fs:                                  tracker.Fs,
			Output:                              ExecutionOutput{},
			StepperFactory:                      stepperFactory,
			DefaultExecutionStepOutputVariables: map[string]map[string]map[string]string{},
		}
		go DeployTrack(execution, cfg, t, trackChan)
	}

	// wait for all executions to finish (this loop matches above range)
	for tExecution := 0; tExecution < numTracks; tExecution++ {
		// waiting to append <-trackChan Track N times will inherently wait for all above executions to finish
		tOutput := <-trackChan
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

		for _, t := range executingTracks {
			executionStepOutputVariables := map[string]map[string]map[string]string{}

			for _, exec := range output.Tracks[t.Name].Output.Executions {
				executionStepOutputVariables[fmt.Sprintf("%s-%s", exec.RegionDeployType, exec.Region)] = exec.Output.StepOutputVariables
			}

			if tracker.Log.Level == logrus.DebugLevel {
				jsonBytes, _ := json.Marshal(executionStepOutputVariables)

				tracker.Log.Debugf("OUTPUT VARS: %s", string(jsonBytes))
			}

			go DestroyTrack(Execution{
				Logger:                              tracker.Log,
				Fs:                                  tracker.Fs,
				Output:                              ExecutionOutput{},
				StepperFactory:                      stepperFactory,
				DefaultExecutionStepOutputVariables: executionStepOutputVariables,
			}, cfg, t, trackDestroyChan)
		}

		// wait for all executions to finish (this loop matches above range)
		for range executingTracks {
			// waiting to append <-trackDestroyChan Track N times will inherently wait for all above executions to finish
			tDestroyOutout := <-trackDestroyChan

			if t, ok := output.Tracks[tDestroyOutout.Name]; ok {
				// TODO: is it better to have a pointer for map value?
				t.DestroyOutput = tDestroyOutout
				output.Tracks[tDestroyOutout.Name] = t
			}
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

	if primaryTrackExecution.Output.FailedTestCount > 0 || primaryTrackExecution.Output.FailureCount > 0 {
		logger.Warn("Primary region deployment encountered errors, skipping regional deployments for this track.")
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
			regionInChan <- RegionExecution{
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

			// if any previous failures, skip
			if (progressionLevel > 1 && execution.Output.FailureCount > 0) || (execution.RegionDeployType == steps.RegionalRegionDeployType && !s.RegionalResourcesExist) {
				go func(s steps.Step) {
					s.Output.Status = steps.Skipped
					sChan <- s
				}(s)
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
