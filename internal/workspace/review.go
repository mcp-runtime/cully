package workspace

import (
	"fmt"
	"slices"
	"time"
)

func approveTask(actor string, p *Project, task *Task, v Input, now time.Time) error {

	if p.Members[actor] != "maintainer" || task.Owner == actor {
		return ErrForbidden
	}
	// A reviewer cannot approve an attempt they contributed to.
	for _, a := range task.Attempts {
		if a.Initiator == actor {
			return ErrForbidden
		}
	}
	if task.State != "review" || v.Revision != task.Revision || task.ReviewGuidanceVersion != p.GuidanceVersion {
		return ErrConflict
	}
	if err := readyForReview(task, now); err != nil {
		return err
	}
	task.State = "done"
	task.ApprovedBy = actor
	task.ApprovedVersion = task.Version + 1
	return nil
}
func readyForReview(t *Task, now time.Time) error {
	if err := validateEvidence(t, now); err != nil {
		return err
	}
	for i := range t.Criteria {
		found := false
		for _, e := range t.Evidence {
			if e.Criterion == i {
				if e.Status != "pass" {
					return fmt.Errorf("%w: failed criterion", ErrInvalid)
				}
				found = true
			}
		}
		if !found {
			return fmt.Errorf("%w: missing criterion evidence", ErrInvalid)
		}
	}
	return nil
}

// Submission preserves gaps and failures for review; acceptance requires
// complete passing evidence. A report still has to be structurally valid.
func validateEvidence(t *Task, now time.Time) error {
	if required(t.Revision, t.Artifact) != nil {
		return fmt.Errorf("%w: artifact and revision required", ErrInvalid)
	}
	for _, e := range t.Evidence {
		if e.Criterion < 0 || e.Criterion >= len(t.Criteria) || required(e.Check) != nil || !slices.Contains([]string{"pass", "fail"}, e.Status) || e.Revision != t.Revision || e.ObservedAt.IsZero() || e.ObservedAt.After(now) {
			return fmt.Errorf("%w: evidence must name a criterion, check, revision and past timestamp", ErrInvalid)
		}
	}
	return nil
}
