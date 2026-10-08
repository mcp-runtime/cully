package workspace

import "time"

// An operation defines its authorization scope, mutation status and handler
// together. Validation, MCP tool selection and dispatch share this contract.
type operationScope uint8

const (
	globalScope operationScope = iota
	teamScope
	projectScope
	membershipScope
)

type operation struct {
	scope  operationScope
	write  bool
	handle func(*Team, string, *Project, Input, time.Time) (Result, string, error)
}

var operations = map[string]operation{
	"whoami":           {globalScope, false, nil},
	"team_create":      {globalScope, true, nil},
	"projects":         {teamScope, false, listProjects},
	"team_member":      {teamScope, true, manageTeamMember},
	"project_create":   {teamScope, true, createProject},
	"project_member":   {membershipScope, true, manageProjectMember},
	"board":            {projectScope, false, listTasks},
	"inbox":            {projectScope, false, listTasks},
	"lessons":          {projectScope, false, listLessons},
	"task_get":         {projectScope, false, getTask},
	"task_create":      {projectScope, true, createTask},
	"task_claim":       {projectScope, true, mutateTask},
	"task_checkpoint":  {projectScope, true, mutateTask},
	"task_release":     {projectScope, true, mutateTask},
	"task_state":       {projectScope, true, mutateTask},
	"task_submit":      {projectScope, true, mutateTask},
	"task_approve":     {projectScope, true, mutateTask},
	"learning_draft":   {projectScope, true, draftLearning},
	"learning_publish": {projectScope, true, changeLearning},
	"learning_unshare": {projectScope, true, changeLearning},
	"learning_edit":    {projectScope, true, changeLearning},
	"learning_delete":  {projectScope, true, changeLearning},
	"playbook_adopt":   {projectScope, true, adoptPlaybook},
}
