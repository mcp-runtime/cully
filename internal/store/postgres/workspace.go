package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mcp-runtime/cully/internal/memory"
	"github.com/mcp-runtime/cully/internal/workspace"
)

func workspaceError(err error) error {
	switch {
	case errors.Is(err, workspace.ErrForbidden):
		return memory.ErrForbidden
	case errors.Is(err, workspace.ErrConflict):
		return fmt.Errorf("%w: refresh before retrying", memory.ErrConflict)
	case errors.Is(err, workspace.ErrInvalid):
		return fmt.Errorf("%w: %v", memory.ErrInvalid, err)
	default:
		return memory.ErrUnavailable
	}
}
func (s *Store) workspace(ctx context.Context, actor string, v workspace.Input) (memory.Result, error) {
	if v.Action == "whoami" {
		return memory.Result{Workspace: &workspace.Result{Principal: actor}}, nil
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return memory.Result{}, memory.ErrUnavailable
	}
	defer tx.Rollback(ctx)
	var team *workspace.Team
	var event *workspace.Event
	var out workspace.Result
	now := time.Now().UTC()
	if v.Action == "team_create" {
		team, err = workspace.NewTeam(actor, v.Name)
		if err != nil {
			return memory.Result{}, workspaceError(err)
		}
		out.TeamID = team.ID
		event = &workspace.Event{ID: uuid.NewString(), TeamID: team.ID, Actor: actor, Action: v.Action, TargetID: team.ID, At: now}
	} else {
		var data []byte
		// Authorize before loading the aggregate; the row lock ensures a
		// membership removal and any subsequent write have a single order.
		err = tx.QueryRow(ctx, `SELECT data FROM cully_workspaces WHERE id=$1::uuid AND data->'members' ? $2 FOR UPDATE`, v.TeamID, actor).Scan(&data)
		if errors.Is(err, pgx.ErrNoRows) {
			return memory.Result{}, memory.ErrForbidden
		}
		if err != nil {
			return memory.Result{}, memory.ErrUnavailable
		}
		if err = json.Unmarshal(data, &team); err != nil {
			return memory.Result{}, memory.ErrUnavailable
		}
		out, event, err = team.Apply(actor, v, now)
		if err != nil {
			return memory.Result{}, workspaceError(err)
		}
	}
	if v.Write() {
		data, err := json.Marshal(team)
		if err != nil {
			return memory.Result{}, memory.ErrUnavailable
		}
		if len(data) > 1<<20 {
			return memory.Result{}, fmt.Errorf("%w: workspace storage limit reached", memory.ErrInvalid)
		}
		if v.Action == "team_create" {
			_, err = tx.Exec(ctx, `INSERT INTO cully_workspaces(id,data) VALUES($1::uuid,$2::jsonb)`, team.ID, data)
		} else {
			_, err = tx.Exec(ctx, `UPDATE cully_workspaces SET data=$2::jsonb,updated_at=now() WHERE id=$1::uuid`, team.ID, data)
		}
		if err != nil {
			return memory.Result{}, memory.ErrUnavailable
		}
		if event != nil {
			var projectID any
			if event.ProjectID != "" {
				projectID = event.ProjectID
			}
			_, err = tx.Exec(ctx, `INSERT INTO cully_workspace_events(id,team_id,project_id,actor,action,target_id,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7)`, event.ID, event.TeamID, projectID, event.Actor, event.Action, event.TargetID, event.At)
			if err != nil {
				return memory.Result{}, memory.ErrUnavailable
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return memory.Result{}, memory.ErrUnavailable
	}
	return memory.Result{Workspace: &out}, nil
}
