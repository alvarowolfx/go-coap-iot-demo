package devices

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jeremywohl/flatten"
	"github.com/nqd/flat"
	bolt "go.etcd.io/bbolt"
)

// DeviceLocalStore Saves device data locally on filesystem
type deviceLocalStore struct {
	db *bolt.DB
}

const deviceBucketPrefix = "device_"

func NewDeviceLocalStore(db *bolt.DB) DeviceStore {
	return &deviceLocalStore{
		db: db,
	}
}

func (s *deviceLocalStore) GetDeviceByID(ctx context.Context, id string) (*Device, error) {
	device := &Device{}
	err := s.db.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket([]byte(deviceBucketPrefix + id))
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
		device.ProjectID = ""
		if value, ok := nestedData["projectID"]; ok {
			device.ProjectID = value.(string)
		}

		return nil
	})

	return device, err
}

func (s *deviceLocalStore) CreateDevice(ctx context.Context, id string, data map[string]interface{}) error {
	data["created"] = time.Now()
	data["deviceID"] = id
	err := s.UpsertDevice(ctx, id, time.Now(), data)

	if err != nil {
		return err
	}

	return nil
}

func (s *deviceLocalStore) UpsertDevice(ctx context.Context, id string, updated time.Time, updates map[string]interface{}) error {
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

	buck := tx.Bucket([]byte(deviceBucketPrefix + id))
	if buck == nil {
		updates["created"] = time.Now()
		updates["deviceID"] = id
		buck, err = tx.CreateBucketIfNotExists([]byte(deviceBucketPrefix + id))
		if err != nil {
			return err
		}
	}

	updates["updated"] = updated
	flattenData, err := flatten.Flatten(updates, "", flatten.PathStyle)
	if err != nil {
		return err
	}

	for k, v := range flattenData {
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

func (s *deviceLocalStore) RegisterDeviceToProject(ctx context.Context, deviceID, projectID string) error {
	updates := make(map[string]interface{})
	updates["projectID"] = projectID
	return s.UpsertDevice(ctx, deviceID, time.Now(), updates)
}

func (s *deviceLocalStore) ListDevicesForProject(ctx context.Context, projectID string) ([]*Device, error) {
	devices := make([]*Device, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, buck *bolt.Bucket) error {
			if strings.HasPrefix(string(name), deviceBucketPrefix) {
				v := buck.Get([]byte("projectID"))
				if v == nil {
					return nil
				}
				value := string(v)
				if value == projectID {
					id := strings.Replace(string(name), deviceBucketPrefix, "", -1)
					device, err := s.GetDeviceByID(ctx, id)
					if err != nil {
						return nil
					}
					if device != nil {
						devices = append(devices, device)
					}
				}
			}
			return nil
		})
		return nil
	})
	return devices, err
}
