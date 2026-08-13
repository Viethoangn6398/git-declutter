package safety

// BranchStatus is the user-facing classification for a local branch.
type BranchStatus string

const (
	StatusSafe      BranchStatus = "safe"
	StatusReview    BranchStatus = "review"
	StatusKeep      BranchStatus = "keep"
	StatusProtected BranchStatus = "protected"
)

func (s BranchStatus) Label() string {
	switch s {
	case StatusSafe:
		return "SAFE"
	case StatusReview:
		return "REVIEW"
	case StatusKeep:
		return "KEEP"
	case StatusProtected:
		return "PROTECTED"
	default:
		return string(s)
	}
}

// SafeForAutomaticDeletion reports whether a status may enter the
// automatic deletion path. KEEP, REVIEW, and PROTECTED must never.
func (s BranchStatus) SafeForAutomaticDeletion() bool {
	return s == StatusSafe
}
