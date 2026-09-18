package reporting

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport() RunReport {
	return RunReport{
		Result:        "fail",
		Message:       "Executed 1/2 steps successfully with 0 test failure(s) across 1 track(s).  Failed: net/vnet/primary/eastus2.",
		ExecutedSteps: 1,
		TotalSteps:    2,
		FailedSteps:   []string{"net/vnet/primary/eastus2"},
		Steps: []StepReport{
			{
				Track: "storage", Step: "account", Region: "eastus2", RegionDeployType: "primary",
				Status: "SUCCESS", PlanChanges: map[string][]string{"[create]": {"a", "b"}, "[update]": {"c"}},
			},
			{
				Track: "net", Step: "vnet", Region: "eastus2", RegionDeployType: "primary",
				Status: "FAIL", Error: "Error: subnet overlaps existing range",
			},
		},
	}
}

func TestChangeSummary(t *testing.T) {
	cases := []struct {
		name string
		in   map[string][]string
		want string
	}{
		{"nil", nil, "—"},
		{"empty", map[string][]string{}, "—"},
		{"only no-op", map[string][]string{"[no-op]": {"a", "b"}}, "no-op"},
		{"create+update+delete", map[string][]string{"[create]": {"a", "b"}, "[update]": {"c"}, "[delete]": {"d"}}, "+2 ~1 -1"},
		{"replace counts both", map[string][]string{"[delete create]": {"a"}}, "+1 ~0 -1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, changeSummary(c.in))
		})
	}
}

func TestRenderMarkdown_ContainsTableAndErrors(t *testing.T) {
	md := renderMarkdown(sampleReport())

	assert.Contains(t, md, "## runiac run summary")
	assert.Contains(t, md, "❌ **FAIL**")
	assert.Contains(t, md, "| Track | Step | Region | Status | Changes |")
	// step rows
	assert.Contains(t, md, "| storage | account | primary/eastus2 | ✅ SUCCESS | +2 ~1 -0 |")
	assert.Contains(t, md, "| net | vnet | primary/eastus2 | ❌ FAIL | — |")
	// collapsible error detail
	assert.Contains(t, md, "### Errors")
	assert.Contains(t, md, "Error: subnet overlaps existing range")
}

func TestRenderMarkdown_SuccessHasNoErrorsSection(t *testing.T) {
	r := RunReport{
		Result:  "success",
		Message: "Executed 1/1 steps successfully with 0 test failure(s) across 1 track(s).",
		Steps: []StepReport{
			{Track: "storage", Step: "account", Region: "eastus2", RegionDeployType: "primary", Status: "SUCCESS"},
		},
	}
	md := renderMarkdown(r)
	assert.Contains(t, md, "✅ **SUCCESS**")
	assert.NotContains(t, md, "### Errors")
}

func TestWrite_NoOutputDir_ReturnsMarkdownAndWritesNothing(t *testing.T) {
	t.Setenv(OutputDirEnv, "")
	fs := afero.NewMemMapFs()

	md := Write(fs, logrus.NewEntry(logrus.New()), sampleReport())

	assert.Contains(t, md, "## runiac run summary")

	// nothing should have been written to the filesystem
	entries, err := afero.ReadDir(fs, ".")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestWrite_WithOutputDir_WritesJSONAndMarkdown(t *testing.T) {
	dir := "/out/reports"
	t.Setenv(OutputDirEnv, dir)
	fs := afero.NewMemMapFs()

	report := sampleReport()
	md := Write(fs, logrus.NewEntry(logrus.New()), report)
	assert.NotEmpty(t, md)

	// summary.md written and matches returned markdown
	gotMD, err := afero.ReadFile(fs, dir+"/summary.md")
	require.NoError(t, err)
	assert.Equal(t, md, string(gotMD))

	// summary.json written and round-trips to the same structured data
	gotJSON, err := afero.ReadFile(fs, dir+"/summary.json")
	require.NoError(t, err)

	var decoded RunReport
	require.NoError(t, json.Unmarshal(gotJSON, &decoded))
	assert.Equal(t, "fail", decoded.Result)
	assert.Equal(t, 2, decoded.TotalSteps)
	require.Len(t, decoded.Steps, 2)
	assert.Equal(t, []string{"net/vnet/primary/eastus2"}, decoded.FailedSteps)
}

func TestSortSteps_OrdersByTrackStepRegion(t *testing.T) {
	steps := []StepReport{
		{Track: "net", Step: "vnet", Region: "westus2"},
		{Track: "net", Step: "vnet", Region: "eastus2"},
		{Track: "net", Step: "dns", Region: "eastus2"},
		{Track: "app", Step: "web", Region: "eastus2"},
	}
	SortSteps(steps)

	got := make([][3]string, len(steps))
	for i, s := range steps {
		got[i] = [3]string{s.Track, s.Step, s.Region}
	}
	assert.Equal(t, [][3]string{
		{"app", "web", "eastus2"},
		{"net", "dns", "eastus2"},
		{"net", "vnet", "eastus2"},
		{"net", "vnet", "westus2"},
	}, got)
}
