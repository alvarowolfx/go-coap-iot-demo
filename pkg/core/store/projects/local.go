package projects

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nqd/flat"
	bolt "go.etcd.io/bbolt"
)

type projectLocalStore struct {
	db *bolt.DB
}

const projectBucketPrefix = "project_"

func NewProjectLocalStore(db *bolt.DB) ProjectStore {
	return &projectLocalStore{
		db: db,
	}
}

func (s *projectLocalStore) GetProjectByID(ctx context.Context, id string) (*Project, error) {
	device := &Project{}
	err := s.db.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket([]byte(projectBucketPrefix + id))
		if buck == nil {
			device = nil
			return nil
		}

		data := make(map[string]interface{})
		cur := buck.Cursor()
		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			data[string(k)] = string(v)
		}

		nestedData, err := flat.Unflatten(data, &flat.Options{
			Delimiter: "/",
		})

		if err != nil {
			return err
		}

		device.Data = nestedData
		device.ID = id
		device.Name = id

		return nil
	})

	return device, err
}

func (s *projectLocalStore) CreateProject(ctx context.Context, name string) error {
	data := make(map[string]interface{})
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			err = tx.Rollback()
			if err != nil {
				log.Printf("err rollback : %v\n", err)
			}
		}
	}()

	buck := tx.Bucket([]byte(projectBucketPrefix + name))
	if buck == nil {
		data["created"] = time.Now()
		data["name"] = name
		data["projectID"] = name
		buck, err = tx.CreateBucketIfNotExists([]byte(projectBucketPrefix + name))
		if err != nil {
			return err
		}
	}

	for k, v := range data {
		value := fmt.Sprintf("%v", v)
		err = buck.Put([]byte(k), []byte(value))
		if err != nil {
			return err
		}
	}

	err = tx.Commit()

	if err != nil {
		return err
	}

	return nil
}
