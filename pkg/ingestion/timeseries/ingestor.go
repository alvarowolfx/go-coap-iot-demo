package timeseries

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"com.aviebrantz.coap-demo/pkg/core/store/historical"
	"github.com/apex/log"
	"gocloud.dev/pubsub"
)

type TimeseriesDataIngestor struct {
	dataSub *pubsub.Subscription
	tsStore historical.TimeSeriesStore
	logger  *log.Entry
}

func NewIngestor(dataSub *pubsub.Subscription, tsStore historical.TimeSeriesStore) *TimeseriesDataIngestor {
	logger := log.WithField("module", "timeseries-ingestor")
	return &TimeseriesDataIngestor{
		dataSub: dataSub,
		tsStore: tsStore,
		logger:  logger,
	}
}

func (tsi TimeseriesDataIngestor) Start() {
	for {
		ctx := context.Background()
		msg, err := tsi.dataSub.Receive(ctx)
		if err != nil {
			tsi.logger.Infof("Receiving message: %v", err)
			break
		}

		deviceID := msg.Metadata["deviceID"]
		var reportedTime time.Time
		timeInt, err := strconv.ParseInt(msg.Metadata["time"], 10, 64)
		if err != nil {
			reportedTime = time.Now()
		} else {
			reportedTime = time.Unix(timeInt, 0)
		}

		tsi.logger.Infof("Got message: %s - %v - %q\n", deviceID, reportedTime, msg.Body)

		var datapoint map[string]interface{}
		err = json.Unmarshal(msg.Body, &datapoint)
		if err != nil {
			tsi.logger.Warnf("Invalid msg format :%v", err)
			// Drop msg
			msg.Ack()
			return
		}

		err = tsi.tsStore.InsertDataPoint(ctx, "device", deviceID, reportedTime, datapoint)

		if err != nil {
			tsi.logger.Errorf("err insert device history :%v", err)
			msg.Nack()
			return
		}

		// Messages must always be acknowledged with Ack.
		msg.Ack()
	}
}
