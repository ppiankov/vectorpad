package classifier

// Tag identifies the semantic class for a sentence.
type Tag string

const (
	TagConstraint  Tag = "CONSTRAINT"
	TagDecision    Tag = "DECISION"
	TagTentative   Tag = "TENTATIVE"
	TagQuestion    Tag = "QUESTION"
	TagSpeculation Tag = "SPECULATION"
	TagExplanation Tag = "EXPLANATION"
)

// LockPolicy defines how a sentence should be protected from rewrite.
type LockPolicy string

const (
	LockPolicyNone      LockPolicy = "NONE"
	LockPolicyHard      LockPolicy = "HARD"
	LockPolicySoft      LockPolicy = "SOFT"
	LockPolicyModalSpan LockPolicy = "MODAL_SPAN"
)

// LockedSpan marks a byte span that should remain unchanged.
type LockedSpan struct {
	Start int
	End   int
	Text  string
}

// Sentence is a classified sentence with lock metadata.
type Sentence struct {
	Text        string
	Tag         Tag
	LockPolicy  LockPolicy
	LockedSpans []LockedSpan
}
