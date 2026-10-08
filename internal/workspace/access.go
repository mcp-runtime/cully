package workspace

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

func (t *Team) project(actor, id string, write bool) (*Project, error) {
	p := t.Projects[id]
	if t.Members[actor] == "" || p == nil || p.Members[actor] == "" || (write && p.Members[actor] == "viewer") {
		return nil, ErrForbidden
	}
	return p, nil
}
func NewTeam(actor, name string) (*Team, error) {
	if err := required(actor, name); err != nil {
		return nil, err
	}
	return &Team{ID: uuid.NewString(), Name: name, Members: map[string]string{actor: "admin"}, Projects: map[string]*Project{}, Tasks: map[string]*Task{}, Learnings: map[string]*Learning{}, Playbooks: map[string]*Playbook{}}, nil
}

func manageTeamMember(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	if t.Members[actor] != "admin" {
		return out, "", ErrForbidden
	}
	if err := required(v.Principal); err != nil {
		return out, "", err
	}
	if v.Role != "admin" && v.Role != "member" && v.Role != "remove" {
		return out, "", ErrInvalid
	}
	if t.Members[v.Principal] == "admin" && v.Role != "admin" {
		n := 0
		for _, role := range t.Members {
			if role == "admin" {
				n++
			}
		}
		if n <= 1 {
			return out, "", fmt.Errorf("%w: keep at least one admin", ErrInvalid)
		}
	}
	if len(t.Members) >= 100 && t.Members[v.Principal] == "" {
		return out, "", ErrInvalid
	}
	if v.Role == "remove" {
		for _, project := range t.Projects {
			if project.Members[v.Principal] != "maintainer" {
				continue
			}
			n := 0
			for _, role := range project.Members {
				if role == "maintainer" {
					n++
				}
			}
			if n <= 1 {
				return out, "", fmt.Errorf("%w: keep at least one maintainer on each project", ErrInvalid)
			}
		}
		delete(t.Members, v.Principal)
		for _, project := range t.Projects {
			delete(project.Members, v.Principal)
		}
	} else {
		t.Members[v.Principal] = v.Role
	}

	return out, v.Principal, nil
}
func createProject(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	if t.Members[actor] != "admin" {
		return out, "", ErrForbidden
	}
	if err := required(v.Name, v.Repository); err != nil {
		return out, "", err
	}
	if len(t.Projects) >= 50 {
		return out, "", ErrInvalid
	}
	p = &Project{ID: uuid.NewString(), Name: v.Name, Repository: v.Repository, Members: map[string]string{actor: "maintainer"}, GuidanceVersion: 1}
	t.Projects[p.ID] = p
	out.Project = p
	return out, p.ID, nil
}
func manageProjectMember(t *Team, actor string, p *Project, v Input, now time.Time) (Result, string, error) {
	out := Result{TeamID: t.ID}

	if p.Members[actor] != "maintainer" && t.Members[actor] != "admin" {
		return out, "", ErrForbidden
	}
	if t.Members[v.Principal] == "" {
		return out, "", ErrForbidden
	}
	if !slices.Contains([]string{"maintainer", "member", "viewer", "remove"}, v.Role) {
		return out, "", ErrInvalid
	}
	if p.Members[v.Principal] == "maintainer" && v.Role != "maintainer" {
		n := 0
		for _, role := range p.Members {
			if role == "maintainer" {
				n++
			}
		}
		if n <= 1 {
			return out, "", ErrInvalid
		}
	}
	if v.Role == "remove" {
		delete(p.Members, v.Principal)
	} else {
		p.Members[v.Principal] = v.Role
	}

	return out, v.Principal, nil
}
