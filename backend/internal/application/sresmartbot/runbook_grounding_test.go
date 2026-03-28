package sresmartbot

import "testing"

func TestRunbookGroundingIndex_FindsRelevantCitations(t *testing.T) {
	index := newRunbookGroundingIndex()
	draft := interpretationDraftFixture()

	citations := index.FindRelevantCitations(draft, 2)
	if len(citations) == 0 {
		t.Fatal("expected runbook citations")
	}
	for _, citation := range citations {
		if citation.Kind != "runbook" {
			t.Fatalf("expected runbook citation kind, got %q", citation.Kind)
		}
		if !index.IsAllowedRunbookSource(citation.Source) {
			t.Fatalf("expected citation source in allowlist, got %q", citation.Source)
		}
		if citation.Section == "" || citation.Note == "" {
			t.Fatalf("expected populated citation fields, got %+v", citation)
		}
	}
}

func TestValidateCitationContract_FailsWithoutEvidence(t *testing.T) {
	err := validateCitationContract([]AgentCitation{
		{
			Kind:    "runbook",
			Source:  "docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md",
			Section: "Deterministic and approval-safe actions",
			Note:    "Use deterministic approval-safe actions.",
		},
	})
	if err == nil {
		t.Fatal("expected citation contract validation to fail when evidence citation is missing")
	}
}

func TestValidateCitationContract_FailsForNonAllowlistedRunbook(t *testing.T) {
	err := validateCitationContract([]AgentCitation{
		{
			Kind:    "runbook",
			Source:  "docs/random/NOT_ALLOWED.md",
			Section: "Unknown",
			Note:    "unknown",
		},
		{
			Kind:   "evidence",
			Source: "logs.recent",
			Note:   "Error signatures increased.",
		},
	})
	if err == nil {
		t.Fatal("expected citation contract validation to fail for non-allowlisted runbook source")
	}
}
