package devices

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/jeremywohl/flatten"
	"gocloud.dev/docstore"
	"gocloud.dev/gcerrors"
)

type deviceDocStore struct {
	devicesColl *docstore.Collection
}

// NewDeviceDocStore create a device store using a goacloud.dev/docstore collection
func NewDeviceDocStore(devicesColl *docstore.Collection) DeviceStore {
	return &deviceDocStore{
		devicesColl: devicesColl,
	}
}

func (s *deviceDocStore) GetDeviceByID(ctx context.Context, id string) (*Device, error) {
	deviceDoc := make(map[string]interface{})
	deviceDoc["deviceID"] = id
	err := s.devicesColl.Get(ctx, deviceDoc)
	if err != nil {
		code := gcerrors.Code(err)
		if code == gcerrors.NotFound {
			return nil, nil
		}
		return nil, err
	}

	projectID := ""
	if value, ok := deviceDoc["projectID"]; ok {
		projectID = value.(string)
	}

	device := &Device{
		ID:        id,
		ProjectID: projectID,
		Data:      deviceDoc,
	}

	return device, nil
}

func (s *deviceDocStore) CreateDevice(ctx context.Context, id string, data map[string]interface{}) error {
	data["created"] = time.Now()
	data["deviceID"] = id
	return s.devicesColl.Create(ctx, data)
}

func (s *deviceDocStore) UpsertDevice(ctx context.Context, id string, updated time.Time, updates map[string]interface{}) error {

	device, err := s.GetDeviceByID(ctx, id)
	if err != nil {
		return err
	}

	if device == nil {
		err = s.CreateDevice(ctx, id, make(map[string]interface{}))
		if err != nil {
			return err
		}
		device, err = s.GetDeviceByID(ctx, id)
		if err != nil {
			return err
		}
	}

	nestedUpdates, err := flatten.Flatten(updates, "", flatten.DotStyle)
	if err != nil {
		log.Printf("Invalid msg format :%v", err)
		return err
	}

	nestedUpdates["updated"] = updated
	mods := docstore.Mods{}
	for k, v := range nestedUpdates {
		mods[docstore.FieldPath(k)] = v
	}

	err = s.devicesColl.Actions().Update(device.Data, mods).Do(ctx)
	if err != nil {
		log.Printf("err update device :%v", err)
	}
	return nil
}

func (s *deviceDocStore) RegisterDeviceToProject(ctx context.Context, deviceID, projectID string) error {
	updates := make(map[string]interface{})
	updates["projectID"] = projectID
	return s.UpsertDevice(ctx, deviceID, time.Now(), updates)
}

func (s *deviceDocStore) ListDevicesForProject(ctx context.Context, projectID string) ([]*Device, error) {
	iter := s.devicesColl.
		Query().
		Where(docstore.FieldPath("projectID"), "=", projectID).
		Get(ctx)
	defer iter.Stop()

	devices := make([]*Device, 0)
	for {
		device := &Device{}
		err := iter.Next(ctx, device)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, nil
		} else {
			devices = append(devices, device)
		}
	}
	return devices, nil
}
