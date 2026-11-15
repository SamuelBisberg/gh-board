// Package board models the GitHub Project Board: the six Statuses, how a
// Pull Request's state maps onto them, and the operations on Cards.
package board

// Status is the Board column a Card sits in.
type Status string

const (
	Todo             Status = "Todo"
	InProgress       Status = "In Progress"
	NeedsReview      Status = "Needs Review"
	ChangesRequested Status = "Changes Requested"
	ApprovedMerged   Status = "Approved & Merged"
	Closed           Status = "Closed"
)

// All lists the Statuses in Board column order.
var All = []Status{Todo, InProgress, NeedsReview, ChangesRequested, ApprovedMerged, Closed}

// Derive maps a Pull Request's state to the Status its Card should have.
// ok is false for a PR closed without merging: the Board's own workflow
// owns Closed, so the tool ignores those.
func Derive(isDraft bool, state, reviewDecision string) (s Status, ok bool) {
	switch state {
	case "MERGED":
		return ApprovedMerged, true
	case "OPEN":
		if isDraft {
			return InProgress, true
		}
		switch reviewDecision {
		case "CHANGES_REQUESTED":
			return ChangesRequested, true
		case "APPROVED":
			return ApprovedMerged, true
		}
		return NeedsReview, true
	}
	return "", false
}
