package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e9169/kopilot/pkg/k8s"
)

const (
	testRuleIDCKS001 = "CKS-001"
	testRuleIDBP001  = "BP-001"
	testWorkloadAppA = "default/Deployment/app-a"
	testWorkloadAppB = "default/Deployment/app-b"
)

// ── sanitizeGradeIcon ─────────────────────────────────────────────────────────

func TestSanitizeGradeIcon(t *testing.T) {
	tests := []struct {
		grade string
		want  string
	}{
		{"A", "🟢"},
		{"B", "🟡"},
		{"C", "🟠"},
		{"D", "🔴"},
		{"F", "💀"},
		{"", "💀"},
		{"Z", "💀"},
	}
	for _, tt := range tests {
		t.Run("grade_"+tt.grade, func(t *testing.T) {
			if got := sanitizeGradeIcon(tt.grade); got != tt.want {
				t.Errorf("sanitizeGradeIcon(%q) = %q, want %q", tt.grade, got, tt.want)
			}
		})
	}
}

// ── sanitizeScoreIcon ─────────────────────────────────────────────────────────

func TestSanitizeScoreIcon(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "✅"},
		{99, "🟢"}, // workloadGrade → A
		{90, "🟢"}, // workloadGrade → A
		{89, "🟡"}, // workloadGrade → B
		{75, "🟡"}, // workloadGrade → B
		{74, "🟠"}, // workloadGrade → C
		{60, "🟠"}, // workloadGrade → C
		{59, "🔴"}, // workloadGrade → D
		{40, "🔴"}, // workloadGrade → D
		{39, "💀"}, // workloadGrade → F
		{0, "💀"},  // workloadGrade → F
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%d", tt.score), func(t *testing.T) {
			if got := sanitizeScoreIcon(tt.score); got != tt.want {
				t.Errorf("sanitizeScoreIcon(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

// ── workloadScore ─────────────────────────────────────────────────────────────

func TestWorkloadScore(t *testing.T) {
	tests := []struct {
		name     string
		findings []k8s.SanitizeFinding
		want     int
	}{
		{"nil findings", nil, 100},
		{"empty findings", []k8s.SanitizeFinding{}, 100},
		{"single critical penalty 10", []k8s.SanitizeFinding{{Penalty: 10}}, 90},
		{"two majors penalty 5 each", []k8s.SanitizeFinding{{Penalty: 5}, {Penalty: 5}}, 90},
		{"penalty exactly 100", []k8s.SanitizeFinding{{Penalty: 100}}, 0},
		{"penalty over 100 clamps to 0", func() []k8s.SanitizeFinding {
			var f []k8s.SanitizeFinding
			for i := 0; i < 12; i++ {
				f = append(f, k8s.SanitizeFinding{Penalty: 10})
			}
			return f
		}(), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workloadScore(tt.findings); got != tt.want {
				t.Errorf("workloadScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ── workloadGrade ─────────────────────────────────────────────────────────────

func TestWorkloadGrade(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "A"},
		{90, "A"},
		{89, "B"},
		{75, "B"},
		{74, "C"},
		{60, "C"},
		{59, "D"},
		{40, "D"},
		{39, "F"},
		{0, "F"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%d", tt.score), func(t *testing.T) {
			if got := workloadGrade(tt.score); got != tt.want {
				t.Errorf("workloadGrade(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

// ── stripNamespaceFromWorkload ────────────────────────────────────────────────

func TestStripNamespaceFromWorkload(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"default/Deployment/my-app", "Deployment/my-app"},
		{"kube-system/DaemonSet/fluentd", "DaemonSet/fluentd"},
		{"Deployment/my-app", "my-app"}, // first slash removed
		{"no-slash", "no-slash"},        // no slash → unchanged
		{"", ""},                        // empty
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stripNamespaceFromWorkload(tt.input); got != tt.want {
				t.Errorf("stripNamespaceFromWorkload(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ── filterSanitizeFindings ────────────────────────────────────────────────────

func TestFilterSanitizeFindings(t *testing.T) {
	findings := []k8s.SanitizeFinding{
		{RuleID: testRuleIDCKS001, Severity: k8s.SanitizeCritical},
		{RuleID: testRuleIDBP001, Severity: k8s.SanitizeMajor},
		{RuleID: "BP-007", Severity: k8s.SanitizeMinor},
		{RuleID: "CKS-002", Severity: k8s.SanitizeCritical},
	}

	criticals := filterSanitizeFindings(findings, k8s.SanitizeCritical)
	if len(criticals) != 2 {
		t.Errorf("filterSanitizeFindings(critical) = %d, want 2", len(criticals))
	}

	majors := filterSanitizeFindings(findings, k8s.SanitizeMajor)
	if len(majors) != 1 {
		t.Errorf("filterSanitizeFindings(major) = %d, want 1", len(majors))
	}

	minors := filterSanitizeFindings(findings, k8s.SanitizeMinor)
	if len(minors) != 1 {
		t.Errorf("filterSanitizeFindings(minor) = %d, want 1", len(minors))
	}
}

func TestFilterSanitizeFindingsEmpty(t *testing.T) {
	got := filterSanitizeFindings(nil, k8s.SanitizeCritical)
	if len(got) != 0 {
		t.Errorf("filterSanitizeFindings(nil) = %d, want 0", len(got))
	}
}

// ── groupFindingsByWorkload ───────────────────────────────────────────────────

func TestGroupFindingsByWorkload(t *testing.T) {
	findings := []k8s.SanitizeFinding{
		{Workload: testWorkloadAppA, RuleID: testRuleIDBP001},
		{Workload: testWorkloadAppB, RuleID: "BP-002"},
		{Workload: testWorkloadAppA, RuleID: "BP-003"},
	}

	groups := groupFindingsByWorkload(findings)
	if len(groups) != 2 {
		t.Errorf("groupFindingsByWorkload() = %d groups, want 2", len(groups))
	}
	if len(groups[testWorkloadAppA]) != 2 {
		t.Errorf("app-a has %d findings, want 2", len(groups[testWorkloadAppA]))
	}
	if len(groups[testWorkloadAppB]) != 1 {
		t.Errorf("app-b has %d findings, want 1", len(groups[testWorkloadAppB]))
	}
}

// ── sortedWorkloadKeysByScore ─────────────────────────────────────────────────

func TestSortedWorkloadKeysByScore(t *testing.T) {
	// workload-b: penalty 20 → score 80 (worst, should be first)
	// workload-a: penalty 10 → score 90
	// workload-c: penalty 5  → score 95 (best, should be last)
	m := map[string][]k8s.SanitizeFinding{
		"ns/Deployment/workload-a": {{Penalty: 10}},
		"ns/Deployment/workload-b": {{Penalty: 10}, {Penalty: 10}},
		"ns/Deployment/workload-c": {{Penalty: 5}},
	}

	keys := sortedWorkloadKeysByScore(m)
	if len(keys) != 3 {
		t.Fatalf("sortedWorkloadKeysByScore() returned %d keys, want 3", len(keys))
	}
	if keys[0] != "ns/Deployment/workload-b" {
		t.Errorf("first key (worst) = %q, want workload-b", keys[0])
	}
	if keys[2] != "ns/Deployment/workload-c" {
		t.Errorf("last key (best) = %q, want workload-c", keys[2])
	}
}

func TestSortedWorkloadKeysByScoreAlphaOnTie(t *testing.T) {
	// Same penalty → sorted alphabetically
	m := map[string][]k8s.SanitizeFinding{
		"ns/Deployment/zebra": {{Penalty: 5}},
		"ns/Deployment/alpha": {{Penalty: 5}},
	}

	keys := sortedWorkloadKeysByScore(m)
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}
	if keys[0] != "ns/Deployment/alpha" {
		t.Errorf("on tie first key = %q, want alpha (alphabetically first)", keys[0])
	}
}

// ── writeSanitizeFindingGroup ─────────────────────────────────────────────────

func TestWriteSanitizeFindingGroup(t *testing.T) {
	var sb strings.Builder
	findings := []k8s.SanitizeFinding{
		{RuleID: testRuleIDCKS001, Container: "app", Message: "privileged container"},
		{RuleID: testRuleIDBP001, Container: "", Message: "no livenessProbe"},
	}
	writeSanitizeFindingGroup(&sb, "CRITICAL", findings)
	out := sb.String()

	if !strings.Contains(out, "[CRITICAL]") {
		t.Error("output missing [CRITICAL] label")
	}
	if !strings.Contains(out, testRuleIDCKS001) {
		t.Error("output missing rule ID CKS-001")
	}
	if !strings.Contains(out, "[app]") {
		t.Error("output missing container name [app]")
	}
	if !strings.Contains(out, testRuleIDBP001) {
		t.Error("output missing rule ID BP-001")
	}
	// A finding with no container should not emit "[]"
	if strings.Contains(out, "[]") {
		t.Error("output should not emit empty container brackets []")
	}
}

func TestWriteSanitizeFindingGroupEmpty(t *testing.T) {
	var sb strings.Builder
	writeSanitizeFindingGroup(&sb, "MINOR", nil)
	if sb.Len() != 0 {
		t.Errorf("writeSanitizeFindingGroup with no findings wrote %d bytes, want 0", sb.Len())
	}
}

// ── formatSanitizeResult ──────────────────────────────────────────────────────

func TestFormatSanitizeResult(t *testing.T) {
	report := &k8s.SanitizeResult{
		Context:        "test-cluster",
		Score:          85,
		Grade:          "B",
		TotalWorkloads: 1,
		TotalFindings:  1,
		CriticalCount:  0,
		MajorCount:     1,
		MinorCount:     0,
		Namespaces: []k8s.NamespaceSanitizeScore{
			{
				Namespace: "production",
				Score:     85,
				Grade:     "B",
				Findings: []k8s.SanitizeFinding{
					{
						RuleID:    testRuleIDBP001,
						Severity:  k8s.SanitizeMajor,
						Workload:  "production/Deployment/my-app",
						Container: "app",
						Message:   "no livenessProbe",
						Penalty:   5,
					},
				},
			},
		},
	}

	result := formatSanitizeResult(report)

	if !strings.Contains(result, "test-cluster") {
		t.Error("formatSanitizeResult output missing context name")
	}
	if !strings.Contains(result, "CLUSTER GRADE: B") {
		t.Error("formatSanitizeResult output missing grade")
	}
	if !strings.Contains(result, "production") {
		t.Error("formatSanitizeResult output missing namespace")
	}
	if !strings.Contains(result, testRuleIDBP001) {
		t.Error("formatSanitizeResult output missing finding rule ID")
	}
	if !strings.Contains(result, "MAJOR") {
		t.Error("formatSanitizeResult output missing MAJOR severity label")
	}
}

func TestFormatSanitizeResultNoFindings(t *testing.T) {
	report := &k8s.SanitizeResult{
		Context:        "clean-cluster",
		Score:          100,
		Grade:          "A",
		TotalWorkloads: 1,
		TotalFindings:  0,
		Namespaces: []k8s.NamespaceSanitizeScore{
			{
				Namespace: "default",
				Score:     100,
				Grade:     "A",
				Findings:  []k8s.SanitizeFinding{},
			},
		},
	}

	result := formatSanitizeResult(report)
	if !strings.Contains(result, "No findings") {
		t.Error("formatSanitizeResult with no findings should include 'No findings' message")
	}
	if !strings.Contains(result, "clean-cluster") {
		t.Error("formatSanitizeResult output missing context name")
	}
}
