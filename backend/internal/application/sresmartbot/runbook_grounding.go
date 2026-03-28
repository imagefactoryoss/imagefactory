package sresmartbot

import (
	"fmt"
	"sort"
	"strings"
)

type AgentCitation struct {
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Section string `json:"section,omitempty"`
	Note    string `json:"note"`
}

type runbookSection struct {
	Source   string
	Section  string
	Keywords []string
	Note     string
}

type scoredRunbookSection struct {
	section runbookSection
	score   int
}

type runbookGroundingIndex struct {
	allowlist map[string]struct{}
	sections  []runbookSection
}

func newRunbookGroundingIndex() *runbookGroundingIndex {
	sections := []runbookSection{
		{
			Source:   "docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md",
			Section:  "Deterministic and approval-safe actions",
			Keywords: []string{"approval", "deterministic", "safe action", "triage", "incident", "recovery"},
			Note:     "Use deterministic, approval-gated actions only.",
		},
		{
			Source:   "docs/implementation/ROBOT_SRE_INCIDENT_TAXONOMY_AND_POLICY_MATRIX.md",
			Section:  "Incident taxonomy and response policy",
			Keywords: []string{"taxonomy", "incident_type", "severity", "policy", "domain", "confidence"},
			Note:     "Map symptoms to incident domain/type policy before remediation.",
		},
		{
			Source:   "docs/implementation/SRE_SMART_BOT_ASYNC_BACKLOG_TRANSPORT_PRESSURE_EPIC.md",
			Section:  "Async backlog and transport pressure correlation",
			Keywords: []string{"async_backlog", "messaging_transport", "reconnect", "disconnect", "consumer lag", "outbox"},
			Note:     "Correlate backlog pressure with transport health before escalating worker changes.",
		},
		{
			Source:   "docs/implementation/SRE_SMART_BOT_NATS_CONSUMER_LAG_PRESSURE_EPIC.md",
			Section:  "NATS consumer lag diagnostics",
			Keywords: []string{"messaging_consumers", "nats", "lag", "pending", "consumer"},
			Note:     "Prioritize lagging consumer diagnostics for NATS pressure incidents.",
		},
		{
			Source:   "docs/implementation/SRE_SMART_BOT_EXTERNAL_CLUSTER_DEPLOYMENT_RUNBOOK.md",
			Section:  "External cluster health and rollout checks",
			Keywords: []string{"cluster_overview", "runtime_health", "deployment", "kubernetes", "node"},
			Note:     "Validate cluster/runtime health checkpoints before configuration changes.",
		},
		{
			Source:   "docs/testing/SRE_SMART_BOT_UX_RUN_SHEET.md",
			Section:  "Operator handoff and incident communication",
			Keywords: []string{"handoff", "operator", "summary", "next checks", "communication"},
			Note:     "Capture operator handoff with timeline, next checks, and safe action context.",
		},
	}

	allowlist := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		allowlist[section.Source] = struct{}{}
	}
	return &runbookGroundingIndex{
		allowlist: allowlist,
		sections:  sections,
	}
}

func (i *runbookGroundingIndex) FindRelevantCitations(draft *AgentDraftResponse, limit int) []AgentCitation {
	if i == nil || len(i.sections) == 0 || limit <= 0 {
		return []AgentCitation{fallbackRunbookCitation()}
	}
	query := draftGroundingQuery(draft)
	scored := make([]scoredRunbookSection, 0, len(i.sections))
	for _, section := range i.sections {
		score := scoreRunbookSection(query, section)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredRunbookSection{section: section, score: score})
	}
	if len(scored) == 0 {
		return []AgentCitation{fallbackRunbookCitation()}
	}
	sort.SliceStable(scored, func(a, b int) bool {
		if scored[a].score == scored[b].score {
			return scored[a].section.Source < scored[b].section.Source
		}
		return scored[a].score > scored[b].score
	})

	citations := make([]AgentCitation, 0, limit)
	seenSources := make(map[string]struct{}, limit)
	for _, candidate := range scored {
		if len(citations) >= limit {
			break
		}
		if _, exists := seenSources[candidate.section.Source]; exists {
			continue
		}
		citations = append(citations, AgentCitation{
			Kind:    "runbook",
			Source:  candidate.section.Source,
			Section: candidate.section.Section,
			Note:    candidate.section.Note,
		})
		seenSources[candidate.section.Source] = struct{}{}
	}
	if len(citations) == 0 {
		return []AgentCitation{fallbackRunbookCitation()}
	}
	return citations
}

func (i *runbookGroundingIndex) IsAllowedRunbookSource(source string) bool {
	if i == nil {
		return false
	}
	_, ok := i.allowlist[strings.TrimSpace(source)]
	return ok
}

func fallbackRunbookCitation() AgentCitation {
	return AgentCitation{
		Kind:    "runbook",
		Source:  "docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md",
		Section: "Deterministic and approval-safe actions",
		Note:    "Fallback grounding citation for deterministic, approval-safe operations.",
	}
}

func buildEvidenceCitations(draft *AgentDraftResponse, limit int) []AgentCitation {
	if limit <= 0 {
		return nil
	}
	citations := make([]AgentCitation, 0, limit)
	seen := make(map[string]struct{}, limit)
	if draft != nil {
		for _, hypothesis := range draft.Hypotheses {
			for _, ref := range hypothesis.EvidenceRefs {
				if len(citations) >= limit {
					break
				}
				source := strings.TrimSpace(ref.ToolName)
				if source == "" {
					source = "draft.evidence_ref"
				}
				key := source + "|" + ref.Summary
				if _, exists := seen[key]; exists {
					continue
				}
				citations = append(citations, AgentCitation{
					Kind:   "evidence",
					Source: source,
					Note:   strings.TrimSpace(ref.Summary),
				})
				seen[key] = struct{}{}
			}
			if len(citations) >= limit {
				break
			}
		}
	}
	if draft != nil && len(citations) < limit {
		for _, run := range draft.ToolRuns {
			if len(citations) >= limit {
				break
			}
			note := summarizeToolRun(run)
			key := run.ToolName + "|" + note
			if _, exists := seen[key]; exists {
				continue
			}
			citations = append(citations, AgentCitation{
				Kind:   "evidence",
				Source: strings.TrimSpace(run.ToolName),
				Note:   strings.TrimSpace(note),
			})
			seen[key] = struct{}{}
		}
	}
	if len(citations) == 0 {
		summary := "Deterministic draft summary used as evidence fallback."
		if draft != nil && strings.TrimSpace(draft.Summary) != "" {
			summary = strings.TrimSpace(draft.Summary)
		}
		citations = append(citations, AgentCitation{
			Kind:   "evidence",
			Source: "draft.summary",
			Note:   summary,
		})
	}
	return citations
}

func validateCitationContract(citations []AgentCitation) error {
	index := newRunbookGroundingIndex()
	if len(citations) == 0 {
		return fmt.Errorf("citation validation failed: at least one runbook and one evidence citation are required")
	}
	hasRunbook := false
	hasEvidence := false
	for _, citation := range citations {
		kind := strings.ToLower(strings.TrimSpace(citation.Kind))
		source := strings.TrimSpace(citation.Source)
		note := strings.TrimSpace(citation.Note)
		if source == "" || note == "" {
			return fmt.Errorf("citation validation failed: citation source and note are required")
		}
		switch kind {
		case "runbook":
			hasRunbook = true
			if !index.IsAllowedRunbookSource(source) {
				return fmt.Errorf("citation validation failed: runbook source %q is not in allowlist", source)
			}
		case "evidence":
			hasEvidence = true
		default:
			return fmt.Errorf("citation validation failed: unsupported citation kind %q", citation.Kind)
		}
	}
	if !hasRunbook || !hasEvidence {
		return fmt.Errorf("citation validation failed: both runbook and evidence citations are required")
	}
	return nil
}

func draftGroundingQuery(draft *AgentDraftResponse) string {
	if draft == nil {
		return ""
	}
	parts := make([]string, 0, 16)
	parts = append(parts, strings.ToLower(strings.TrimSpace(draft.Summary)))
	for _, hypothesis := range draft.Hypotheses {
		parts = append(parts, strings.ToLower(strings.TrimSpace(hypothesis.Title)))
		parts = append(parts, strings.ToLower(strings.TrimSpace(hypothesis.Rationale)))
		for _, signal := range hypothesis.SignalsUsed {
			parts = append(parts, strings.ToLower(strings.TrimSpace(signal)))
		}
	}
	for _, step := range draft.InvestigationPlan {
		parts = append(parts, strings.ToLower(strings.TrimSpace(step.Title)))
		parts = append(parts, strings.ToLower(strings.TrimSpace(step.Description)))
	}
	for _, run := range draft.ToolRuns {
		parts = append(parts, strings.ToLower(strings.TrimSpace(run.ToolName)))
	}
	return strings.Join(parts, " ")
}

func scoreRunbookSection(query string, section runbookSection) int {
	if strings.TrimSpace(query) == "" {
		return 0
	}
	score := 0
	for _, keyword := range section.Keywords {
		term := strings.ToLower(strings.TrimSpace(keyword))
		if term == "" {
			continue
		}
		if strings.Contains(query, term) {
			score++
		}
	}
	return score
}
