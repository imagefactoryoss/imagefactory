package project

import (
	"time"

	"github.com/google/uuid"
)

// ProjectEvent defines project event behavior.
type ProjectEvent interface {
	ProjectID() uuid.UUID
	TenantID() uuid.UUID
	ProjectName() string
	OccurredAt() time.Time
	ActorID() *uuid.UUID
}

type ProjectCreated struct {
	projectID   uuid.UUID
	tenantID    uuid.UUID
	projectName string
	visibility  string
	gitRepo     string
	gitBranch   string
	gitProvider string
	actorID     *uuid.UUID
	occurredAt  time.Time
}

func NewProjectCreated(p *Project, actorID *uuid.UUID) *ProjectCreated {
	return &ProjectCreated{
		projectID:   p.ID(),
		tenantID:    p.TenantID(),
		projectName: p.Name(),
		visibility:  string(p.Visibility()),
		gitRepo:     p.GitRepo(),
		gitBranch:   p.GitBranch(),
		gitProvider: p.GitProvider(),
		actorID:     actorID,
		occurredAt:  time.Now().UTC(),
	}
}

func (e *ProjectCreated) ProjectID() uuid.UUID  { return e.projectID }
func (e *ProjectCreated) TenantID() uuid.UUID   { return e.tenantID }
func (e *ProjectCreated) ProjectName() string   { return e.projectName }
func (e *ProjectCreated) Visibility() string    { return e.visibility }
func (e *ProjectCreated) GitRepo() string       { return e.gitRepo }
func (e *ProjectCreated) GitBranch() string     { return e.gitBranch }
func (e *ProjectCreated) GitProvider() string   { return e.gitProvider }
func (e *ProjectCreated) ActorID() *uuid.UUID   { return e.actorID }
func (e *ProjectCreated) OccurredAt() time.Time { return e.occurredAt }

type ProjectUpdated struct {
	projectID   uuid.UUID
	tenantID    uuid.UUID
	projectName string
	visibility  string
	gitRepo     string
	gitBranch   string
	gitProvider string
	actorID     *uuid.UUID
	occurredAt  time.Time
}

func NewProjectUpdated(p *Project, actorID *uuid.UUID) *ProjectUpdated {
	return &ProjectUpdated{
		projectID:   p.ID(),
		tenantID:    p.TenantID(),
		projectName: p.Name(),
		visibility:  string(p.Visibility()),
		gitRepo:     p.GitRepo(),
		gitBranch:   p.GitBranch(),
		gitProvider: p.GitProvider(),
		actorID:     actorID,
		occurredAt:  time.Now().UTC(),
	}
}

func (e *ProjectUpdated) ProjectID() uuid.UUID  { return e.projectID }
func (e *ProjectUpdated) TenantID() uuid.UUID   { return e.tenantID }
func (e *ProjectUpdated) ProjectName() string   { return e.projectName }
func (e *ProjectUpdated) Visibility() string    { return e.visibility }
func (e *ProjectUpdated) GitRepo() string       { return e.gitRepo }
func (e *ProjectUpdated) GitBranch() string     { return e.gitBranch }
func (e *ProjectUpdated) GitProvider() string   { return e.gitProvider }
func (e *ProjectUpdated) ActorID() *uuid.UUID   { return e.actorID }
func (e *ProjectUpdated) OccurredAt() time.Time { return e.occurredAt }

type ProjectDeleted struct {
	projectID   uuid.UUID
	tenantID    uuid.UUID
	projectName string
	actorID     *uuid.UUID
	occurredAt  time.Time
}

func NewProjectDeleted(p *Project, actorID *uuid.UUID) *ProjectDeleted {
	return &ProjectDeleted{
		projectID:   p.ID(),
		tenantID:    p.TenantID(),
		projectName: p.Name(),
		actorID:     actorID,
		occurredAt:  time.Now().UTC(),
	}
}

func (e *ProjectDeleted) ProjectID() uuid.UUID  { return e.projectID }
func (e *ProjectDeleted) TenantID() uuid.UUID   { return e.tenantID }
func (e *ProjectDeleted) ProjectName() string   { return e.projectName }
func (e *ProjectDeleted) ActorID() *uuid.UUID   { return e.actorID }
func (e *ProjectDeleted) OccurredAt() time.Time { return e.occurredAt }
