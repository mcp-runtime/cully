package workspace

import (
	"time"

	"github.com/google/uuid"
)

func draftLearning(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	task := t.Tasks[v.TaskID]
	if task == nil || task.ProjectID != p.ID {
		return out, "", ErrForbidden
	}
	if task.State != "done" {
		return out, "", ErrConflict
	}
	if err := required(v.Lesson, v.AppliesWhen, v.Limitations); err != nil {
		return out, "", err
	}
	if len(t.Learnings) >= 500 {
		return out, "", ErrInvalid
	}
	l := &Learning{ID: uuid.NewString(), ProjectID: p.ID, TaskID: task.ID, TaskVersion: task.Version, Artifact: task.Artifact, Revision: task.Revision, Author: actor, Lesson: v.Lesson, AppliesWhen: v.AppliesWhen, Limitations: v.Limitations, Version: 1}
	t.Learnings[l.ID] = l
	out.Learning = l

	return out, l.ID, nil
}
func changeLearning(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	l := t.Learnings[v.LearningID]
	if l == nil || l.ProjectID != p.ID {
		return out, "", ErrForbidden
	}
	if !l.Published && l.Author != actor {
		return out, "", ErrForbidden
	}
	if err := conflict(l.Version, v.Version); err != nil {
		return out, "", err
	}
	if l.Author != actor {
		return out, "", ErrForbidden
	}
	if v.Action == "learning_edit" {
		if required(v.Lesson, v.AppliesWhen, v.Limitations) != nil {
			return out, "", ErrInvalid
		}
		l.Lesson = v.Lesson
		l.AppliesWhen = v.AppliesWhen
		l.Limitations = v.Limitations
		task := t.Tasks[l.TaskID]
		if task == nil {
			return out, "", ErrForbidden
		}
		l.TaskVersion = task.Version
		l.Artifact = task.Artifact
		l.Revision = task.Revision
	}
	if v.Action == "learning_publish" && (t.Tasks[l.TaskID] == nil || t.Tasks[l.TaskID].Version != l.TaskVersion) {
		return out, "", ErrConflict
	}
	l.Published = v.Action == "learning_publish"
	l.Version++
	out.Learning = l
	for _, b := range t.Playbooks {
		if b.LearningID == l.ID && b.Status == "active" {
			b.Status = "needs_review"
			b.Version++
			p.GuidanceVersion++
		}
	}
	if v.Action == "learning_delete" {
		delete(t.Learnings, l.ID)
		out.Learning = nil
	}
	return out, l.ID, nil
}
