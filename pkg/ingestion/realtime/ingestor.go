package realtime

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"com.aviebrantz.coap-demo/pkg/core/store/devices"
	"github.com/apex/log"
	"gocloud.dev/pubsub"
)

type RealtimeDataIngestor struct {
	dataSub     *pubsub.Subscription
	deviceStore devices.DeviceStore
	logger      *log.Entry
}

func NewIngestor(dataSub *pubsub.Subscription, deviceStore devices.DeviceStore) *RealtimeDataIngestor {
	logger := log.WithField("module", "realtime-ingestor")
	return &RealtimeDataIngestor{
		dataSub:     dataSub,
		deviceStore: deviceStore,
		logger:      logger,
	}
}

func (rti RealtimeDataIngestor) Start() {
	// Loop on received messages.
	for {
		ctx := context.Background()
		msg, err := rti.dataSub.Receive(ctx)
		if err != nil {
			rti.logger.Warnf("err receiving message: %v", err)
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

		rti.logger.Infof("Got message: %s - %v - %q\n", deviceID, reportedTime, msg.Body)

		var updates map[string]interface{}
		err = json.Unmarshal(msg.Body, &updates)
		if err != nil {
			rti.logger.Warnf("Invalid msg format :%v", err)
			// Drop msg
			msg.Ack()
			return
		}

		err = rti.deviceStore.UpsertDevice(ctx, deviceID, reportedTime, updates)

		if err != nil {
			rti.logger.Errorf("err update device :%v", err)
			msg.Nack()
			return
		}

		// Messages must always be acknowledged with Ack.
		msg.Ack()
	}
}
