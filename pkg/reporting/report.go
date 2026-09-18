// Package reporting renders a concise, machine- and human-readable summary of a
// runiac run (per-step status, errors, and terraform plan changes) so results can
// be surfaced without scrolling the full logs.
package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// OutputDirEnv, when set, is the directory the JSON and Markdown reports are
// written to. When unset, reporting only returns the Markdown for echoing and
// writes no files (keeps the default behaviour side-effect free).
const OutputDirEnv = "RUNIAC_OUTPUT_DIR"

// StepReport is the per-step result captured in the run report.
type StepReport struct {
	Track            string              `json:"track"`
	Step             string              `json:"step"`
	Region           string              `json:"region"`
	RegionDeployType string              `json:"region_deploy_type"`
	Status           string              `json:"status"`
	Error            string              `json:"error,omitempty"`
	PlanChanges      map[string][]string `json:"plan_changes,omitempty"`
}

// RunReport is the full run summary.
type RunReport struct {
	Result        string       `json:"result"`
	Message       string       `json:"message"`
	ExecutedSteps int          `json:"executed_steps"`
	TotalSteps    int          `json:"total_steps"`
	FailedSteps   []string     `json:"failed_steps,omitempty"`
	SkippedSteps  []string     `json:"skipped_steps,omitempty"`
	Steps         []StepReport `json:"steps"`
}

// Write renders the Markdown summary and, when OutputDirEnv is set, also writes
// summary.json and summary.md to that directory. It is best-effort: any IO error
// is logged and swallowed so reporting never changes the run's outcome. The
// Markdown is always returned so the caller can echo it to stdout.
func Write(fs afero.Fs, log *logrus.Entry, report RunReport) string {
	md := renderMarkdown(report)

	dir := os.Getenv(OutputDirEnv)
	if dir == "" {
		return md
	}

	if err := fs.MkdirAll(dir, 0o755); err != nil {
		log.WithError(err).Warnf("run report: unable to create output dir %q", dir)
		return md
	}

	if data, err := json.MarshalIndent(report, "", "  "); err != nil {
		log.WithError(err).Warn("run report: unable to marshal summary.json")
	} else if err := afero.WriteFile(fs, filepath.Join(dir, "summary.json"), data, 0o644); err != nil {
		log.WithError(err).Warn("run report: unable to write summary.json")
	}

	if err := afero.WriteFile(fs, filepath.Join(dir, "summary.md"), []byte(md), 0o644); err != nil {
		log.WithError(err).Warn("run report: unable to write summary.md")
	}

	return md
}

func renderMarkdown(r RunReport) string {
	var b strings.Builder

	icon := "✅"
	if r.Result != "success" {
		icon = "❌"
	}

	b.WriteString("## runiac run summary\n\n")
	b.WriteString(fmt.Sprintf("%s **%s** — %s\n\n", icon, strings.ToUpper(r.Result), r.Message))

	b.WriteString("| Track | Step | Region | Status | Changes |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, s := range r.Steps {
		region := s.Region
		if s.RegionDeployType != "" {
			region = fmt.Sprintf("%s/%s", s.RegionDeployType, s.Region)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			s.Track, s.Step, region, statusIcon(s.Status), changeSummary(s.PlanChanges)))
	}

	// Collapsible error detail only when there are failures.
	var withErrors []StepReport
	for _, s := range r.Steps {
		if s.Error != "" {
			withErrors = append(withErrors, s)
		}
	}
	if len(withErrors) > 0 {
		b.WriteString("\n### Errors\n\n")
		for _, s := range withErrors {
			b.WriteString(fmt.Sprintf("<details><summary>%s / %s / %s/%s</summary>\n\n```\n%s\n```\n</details>\n\n",
				s.Track, s.Step, s.RegionDeployType, s.Region, strings.TrimSpace(s.Error)))
		}
	}

	return b.String()
}

func statusIcon(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCESS":
		return "✅ SUCCESS"
	case "FAIL":
		return "❌ FAIL"
	case "SKIPPED":
		return "⏭️ SKIPPED"
	case "UNSTABLE":
		return "⚠️ UNSTABLE"
	default:
		return status
	}
}

// changeSummary condenses terraform plan actions into a "+create ~update -delete"
// tally. Composite actions (e.g. replace = delete+create) count toward both.
func changeSummary(plan map[string][]string) string {
	if len(plan) == 0 {
		return "—"
	}
	create, update, del := 0, 0, 0
	for action, addrs := range plan {
		n := len(addrs)
		a := strings.ToLower(action)
		if strings.Contains(a, "create") {
			create += n
		}
		if strings.Contains(a, "update") {
			update += n
		}
		if strings.Contains(a, "delete") {
			del += n
		}
	}
	if create == 0 && update == 0 && del == 0 {
		return "no-op"
	}
	return fmt.Sprintf("+%d ~%d -%d", create, update, del)
}

// SortSteps orders steps deterministically (track, step, region) for stable output.
func SortSteps(steps []StepReport) {
	sort.SliceStable(steps, func(i, j int) bool {
		if steps[i].Track != steps[j].Track {
			return steps[i].Track < steps[j].Track
		}
		if steps[i].Step != steps[j].Step {
			return steps[i].Step < steps[j].Step
		}
		return steps[i].Region < steps[j].Region
	})
}
