package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"com.aviebrantz.coap-demo/api"
	"com.aviebrantz.coap-demo/gateway/coap"
	"com.aviebrantz.coap-demo/ingestion/realtime"
	"com.aviebrantz.coap-demo/ingestion/timeseries"
	"gocloud.dev/docstore"
	"gocloud.dev/pubsub"

	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/docstore/mongodocstore"
	_ "gocloud.dev/pubsub/mempubsub"
)

func main() {
	ctx := context.Background()
	dataTopic, err := pubsub.OpenTopic(ctx, "mem://dataTopic")
	if err != nil {
		log.Fatalf("Err creating data topic :%v", err)
	}
	defer dataTopic.Shutdown(ctx)

	dataSub, err := pubsub.OpenSubscription(ctx, "mem://dataTopic")
	if err != nil {
		log.Fatalf("could not open data topic subscription :%v", err)
	}
	defer dataSub.Shutdown(ctx)

	deviceCollURL := "mongo://iot-coap-platform/devices?id_field=deviceID"
	os.Setenv("MONGO_SERVER_URL", "mongodb://localhost:27017")
	//deviceCollURL := "mem://devices/deviceID"
	devicesColl, err := docstore.OpenCollection(ctx, deviceCollURL)
	if err != nil {
		log.Fatalf("could not open devices collection :%v", err)
	}
	defer devicesColl.Close()

	deviceHistoryCollURL := "mongo://iot-coap-platform/device_history"
	//deviceHistoryCollURL := "mem://device_history/id"
	deviceHistoryColl, err := docstore.OpenCollection(ctx, deviceHistoryCollURL)
	if err != nil {
		log.Fatalf("could not open device history collection :%v", err)
	}
	defer deviceHistoryColl.Close()

	coapGateway := coap.NewGateway(dataTopic)
	realtimeIngestor := realtime.NewIngestor(dataSub, devicesColl)
	timeseriesIngestor := timeseries.NewIngestor(dataSub, deviceHistoryColl)
	apiServer := api.NewServer(devicesColl)

	go coapGateway.Start()
	go realtimeIngestor.Start()
	go timeseriesIngestor.Start()
	go apiServer.Start()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Print("Server Started")
	<-done
	log.Print("Server Stopped")

	// for tcp
	// log.Fatal(coap.ListenAndServe("tcp", ":5688",  r))
	// for tcp-tls
	// log.Fatal(coap.ListenAndServeTLS("tcp", ":5688", &tls.Config{...}, r))
	// for udp-dtls
	// log.Fatal(coap.ListenAndServeDTLS("udp", ":5688", &dtls.Config{...}, r))
}
