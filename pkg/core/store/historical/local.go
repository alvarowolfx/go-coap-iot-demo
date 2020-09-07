package historical

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

type localTimeSeriesStore struct {
	db *bolt.DB
}

func NewTimeSeriesLocalStore(db *bolt.DB) TimeSeriesStore {
	return &localTimeSeriesStore{
		db: db,
	}
}

func getBucketName(datatype, id string) string {
	return fmt.Sprintf("history_%s_%s", datatype, id)
}

func (s *localTimeSeriesStore) InsertDataPoint(ctx context.Context, datatype string, id string, reportedTime time.Time, data map[string]interface{}) error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			err = tx.Rollback()
		}
	}()

	buck, err := tx.CreateBucketIfNotExists([]byte(getBucketName(datatype, id)))
	if err != nil {
		return err
	}

	value, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = buck.Put([]byte(reportedTime.Format(time.RFC3339)), value)
	if err != nil {
		return err
	}

	err = tx.Commit()

	if err != nil {
		return err
	}

	return nil
}

func (s *localTimeSeriesStore) GetDataPointsInRange(ctx context.Context, datatype string, id string, start time.Time, end time.Time) ([]*DataPoint, error) {
	points := make([]*DataPoint, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		min := []byte(start.Format(time.RFC3339))
		max := []byte(end.Format(time.RFC3339))

		buck := tx.Bucket([]byte(getBucketName(datatype, id)))
		if buck == nil {
			return nil
		}

		c := buck.Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			t, err := time.Parse(time.RFC3339, string(k))
			if err != nil {
				continue
			}

			data := make(map[string]interface{})
			err = json.Unmarshal(v, &data)
			if err != nil {
				continue
			}

			point := &DataPoint{
				Time: t,
				Data: data,
			}
			points = append(points, point)
		}
		return nil
	})

	return points, err
}
