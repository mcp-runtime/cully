package workspace

import (
	"slices"
	"sort"
	"strings"
	"time"
)

func listProjects(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	for _, p := range t.Projects {
		if p.Members[actor] != "" {
			out.Projects = append(out.Projects, *p)
		}
	}
	sort.Slice(out.Projects, func(i, j int) bool { return out.Projects[i].ID < out.Projects[j].ID })
	return out, "", nil
}
func listTasks(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	for _, task := range t.Tasks {
		if task.ProjectID == p.ID {
			if v.Action == "inbox" && task.State != "blocked" && task.State != "review" && !(task.State == "active" && len(task.Attempts) > 0 && !now.Before(task.Attempts[len(task.Attempts)-1].LeaseUntil)) {
				continue
			}
			out.Tasks = append(out.Tasks, *task)
		}
	}
	sort.Slice(out.Tasks, func(i, j int) bool { return out.Tasks[i].ID < out.Tasks[j].ID })

	return out, "", nil
}
func listLessons(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	for _, l := range t.Learnings {
		task := t.Tasks[l.TaskID]
		current := task != nil && task.ProjectID == p.ID && task.Version == l.TaskVersion
		if l.ProjectID == p.ID && ((l.Published && current) || l.Author == actor) && (v.Query == "" || strings.Contains(strings.ToLower(l.Lesson+" "+l.AppliesWhen), strings.ToLower(v.Query))) {
			out.Learnings = append(out.Learnings, *l)
		}
	}
	sort.Slice(out.Learnings, func(i, j int) bool { return out.Learnings[i].ID < out.Learnings[j].ID })
	if len(out.Learnings) > 3 {
		out.Learnings = out.Learnings[:3]
	}
	for _, book := range t.Playbooks {
		l := t.Learnings[book.LearningID]
		if book.ProjectID == p.ID && book.Status == "active" && l != nil && l.Published && l.Version == book.SourceVersion && t.Tasks[l.TaskID] != nil && t.Tasks[l.TaskID].Version == l.TaskVersion {
			out.Playbooks = append(out.Playbooks, *book)
		}
	}
	sort.Slice(out.Playbooks, func(i, j int) bool { return out.Playbooks[i].ID < out.Playbooks[j].ID })
	if len(out.Playbooks) > 3 {
		out.Playbooks = out.Playbooks[:3]
	}

	return out, "", nil
}
func getTask(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	task := t.Tasks[v.TaskID]
	if task == nil || task.ProjectID != p.ID {
		return Result{}, "", ErrForbidden
	}
	out := Result{TeamID: t.ID, Task: task}
	if len(task.Attempts) > 0 {
		out.Stale = now.After(task.Attempts[len(task.Attempts)-1].LeaseUntil) && slices.Contains([]string{"active", "blocked"}, task.State)
	}
	return out, task.ID, nil
}
