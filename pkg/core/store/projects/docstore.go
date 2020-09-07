package projects

import (
	"context"
	"time"

	"gocloud.dev/docstore"
	"gocloud.dev/gcerrors"
)

type projectDocStore struct {
	coll *docstore.Collection
}

// NewProjectDocStore create a project store using a goacloud.dev/docstore collection
func NewProjectDocStore(projectColl *docstore.Collection) ProjectStore {
	return &projectDocStore{
		coll: projectColl,
	}
}

func (s *projectDocStore) GetProjectByID(ctx context.Context, id string) (*Project, error) {
	projectDoc := make(map[string]interface{})
	projectDoc["projectID"] = id
	err := s.coll.Get(ctx, projectDoc)
	if err != nil {
		code := gcerrors.Code(err)
		if code == gcerrors.NotFound {
			return nil, nil
		}
		return nil, err
	}

	project := &Project{
		ID:   id,
		Data: projectDoc,
	}

	return project, nil
}

func (s *projectDocStore) CreateProject(ctx context.Context, name string) error {
	data := make(map[string]interface{})
	data["created"] = time.Now()
	data["projectID"] = name
	data["name"] = name
	return s.coll.Create(ctx, data)
}
