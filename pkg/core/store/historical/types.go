package historical

import (
	"context"
	"time"
)

type TimeSeriesStore interface {
	InsertDataPoint(ctx context.Context, datatype string, id string, time time.Time, data map[string]interface{}) error
	GetDataPointsInRange(ctx context.Context, datatype string, id string, start time.Time, end time.Time) ([]*DataPoint, error)
}

type DataPoint struct {
	Time time.Time              `json:"time"`
	Data map[string]interface{} `json:"data"`
}
