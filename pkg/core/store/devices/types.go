package devices

import (
	"context"
	"time"
)

type DeviceStore interface {
	GetDeviceByID(ctx context.Context, id string) (*Device, error)
	CreateDevice(ctx context.Context, id string, data map[string]interface{}) error
	UpsertDevice(ctx context.Context, id string, updated time.Time, updates map[string]interface{}) error
	RegisterDeviceToProject(ctx context.Context, deviceID, projectID string) error
	ListDevicesForProject(ctx context.Context, projectID string) ([]*Device, error)
}

type Device struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"projectID"`
	Data      map[string]interface{} `json:"data"`
}
