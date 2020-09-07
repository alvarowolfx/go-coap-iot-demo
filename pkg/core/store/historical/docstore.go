package historical

import (
	"context"
	"io"
	"time"

	"gocloud.dev/docstore"
)

type historicalDocStore struct {
	coll *docstore.Collection
}

// NewHistoricalDocStore create a historical store using a goacloud.dev/docstore collection
func NewHistoricalDocStore(coll *docstore.Collection) TimeSeriesStore {
	return &historicalDocStore{
		coll: coll,
	}
}

func (s *historicalDocStore) InsertDataPoint(ctx context.Context, datatype string, id string, reportedTime time.Time, data map[string]interface{}) error {
	data["deviceID"] = id
	data["time"] = reportedTime.Format(time.RFC3339)
	return s.coll.Actions().Create(data).Do(ctx)
}

func (s *historicalDocStore) GetDataPointsInRange(ctx context.Context, datatype string, id string, start time.Time, end time.Time) ([]*DataPoint, error) {

	iter := s.coll.
		Query().
		Where("deviceID", "=", id).
		Where("type", "=", datatype).
		Where("time", ">=", start.Format(time.RFC3339)).
		Where("time", "<=", end.Format(time.RFC3339)).
		OrderBy("time", "desc").
		Get(ctx)

	defer iter.Stop()

	points := make([]*DataPoint, 0)
	for {
		data := make(map[string]interface{})
		err := iter.Next(ctx, &data)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, nil
		} else {
			timeStr := data["time"].(string)
			t, err := time.Parse(timeStr, time.RFC3339)
			if err != nil {
				continue
			}

			point := &DataPoint{
				Time: t,
				Data: data,
			}

			points = append(points, point)
		}
	}

	return points, nil
}
