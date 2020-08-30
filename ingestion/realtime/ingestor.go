package realtime

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/jeremywohl/flatten"
	"gocloud.dev/docstore"
	"gocloud.dev/gcerrors"
	"gocloud.dev/pubsub"
)

type RealtimeDataIngestor struct {
	dataSub     *pubsub.Subscription
	devicesColl *docstore.Collection
}

func NewIngestor(dataSub *pubsub.Subscription, devicesColl *docstore.Collection) *RealtimeDataIngestor {
	return &RealtimeDataIngestor{
		dataSub:     dataSub,
		devicesColl: devicesColl,
	}
}

func (rti RealtimeDataIngestor) Start() {
	// Loop on received messages.
	for {
		ctx := context.Background()
		msg, err := rti.dataSub.Receive(ctx)
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

		deviceDoc := make(map[string]interface{})
		deviceDoc["deviceID"] = deviceID

		err = rti.devicesColl.Get(ctx, deviceDoc)
		if err != nil {
			code := gcerrors.Code(err)
			// On FailedPrecondition or NotFound, try again.
			if code == gcerrors.NotFound {
				deviceDoc["created"] = time.Now()
				rti.devicesColl.Create(ctx, deviceDoc)
			} else {
				log.Printf("Error getting device %v", err)
				msg.Nack()
				return
			}
		}

		var updates map[string]interface{}
		err = json.Unmarshal(msg.Body, &updates)
		if err != nil {
			log.Printf("Invalid msg format :%v", err)
			// Drop msg
			msg.Ack()
			return
		}

		nestedUpdates, err := flatten.Flatten(updates, "", flatten.DotStyle)
		if err != nil {
			log.Printf("Invalid msg format :%v", err)
			// Drop msg
			msg.Ack()
			return
		}

		nestedUpdates["updated"] = reportedTime
		mods := docstore.Mods{}
		for k, v := range nestedUpdates {
			//newKey := strings.ReplaceAll(k, "/", ".")
			// log.Printf(" Key - %s ", k)
			mods[docstore.FieldPath(k)] = v
		}

		err = rti.devicesColl.Actions().Update(deviceDoc, mods).Do(ctx)
		if err != nil {
			log.Printf("err update device :%v", err)
		}

		// Messages must always be acknowledged with Ack.
		msg.Ack()
	}
}
