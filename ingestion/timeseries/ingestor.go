package timeseries

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"
)

type TimeseriesDataIngestor struct {
	dataSub           *pubsub.Subscription
	deviceHistoryColl *docstore.Collection
}

func NewIngestor(dataSub *pubsub.Subscription, deviceHistoryColl *docstore.Collection) *TimeseriesDataIngestor {
	return &TimeseriesDataIngestor{
		dataSub:           dataSub,
		deviceHistoryColl: deviceHistoryColl,
	}
}

func (tsi TimeseriesDataIngestor) Start() {
	// Loop on received messages.
	for {
		ctx := context.Background()
		msg, err := tsi.dataSub.Receive(ctx)
		if err != nil {
			// Errors from Receive indicate that Receive will no longer succeed.
			log.Printf("Receiving message: %v", err)
			break
		}
		// Do work based on the message, for example:
		deviceID := msg.Metadata["deviceID"]
		var reportedTime time.Time
		timeInt, err := strconv.ParseInt(msg.Metadata["time"], 10, 64)
		if err != nil {
			reportedTime = time.Now()
		} else {
			reportedTime = time.Unix(timeInt, 0)
		}

		log.Printf("Got message: %s - %v - %q\n", deviceID, reportedTime, msg.Body)

		var datapoint map[string]interface{}
		err = json.Unmarshal(msg.Body, &datapoint)
		if err != nil {
			log.Printf("Invalid msg format :%v", err)
			// Drop msg
			msg.Ack()
			return
		}

		datapoint["deviceID"] = deviceID
		datapoint["time"] = reportedTime

		err = tsi.deviceHistoryColl.Actions().Create(datapoint).Do(ctx)
		if err != nil {
			log.Printf("err insert device history :%v", err)
		}

		// Messages must always be acknowledged with Ack.
		msg.Ack()
	}
}
