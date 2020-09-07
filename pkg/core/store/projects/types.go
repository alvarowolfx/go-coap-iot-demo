package projects

import (
	"context"
)

type ProjectStore interface {
	GetProjectByID(ctx context.Context, id string) (*Project, error)
	CreateProject(ctx context.Context, name string) error
}

type Project struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}
