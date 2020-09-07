package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"

	"com.aviebrantz.coap-demo/pkg/api"
	"com.aviebrantz.coap-demo/pkg/config"
	"com.aviebrantz.coap-demo/pkg/core/store/devices"
	"com.aviebrantz.coap-demo/pkg/core/store/historical"
	"com.aviebrantz.coap-demo/pkg/core/store/projects"
	"com.aviebrantz.coap-demo/pkg/gateway/coap"
	"com.aviebrantz.coap-demo/pkg/ingestion/realtime"
	"com.aviebrantz.coap-demo/pkg/ingestion/timeseries"
	"gocloud.dev/pubsub"

	bolt "go.etcd.io/bbolt"

	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/docstore/mongodocstore"
	_ "gocloud.dev/pubsub/mempubsub"
)

var (
	dataTopic *pubsub.Topic
)

func setupDataTopic(ctx context.Context) error {
	if dataTopic != nil {
		return nil
	}

	var err error
	dataTopic, err = pubsub.OpenTopic(ctx, "mem://dataTopic")
	if err != nil {
		return err
	}

	return nil
}

func setupDataSub(ctx context.Context) (*pubsub.Subscription, error) {
	dataSub, err := pubsub.OpenSubscription(ctx, "mem://dataTopic")
	if err != nil {
		return nil, err
	}
	return dataSub, nil
}

func shutdownTopic(ctx context.Context, topic *pubsub.Topic) {
	if topic == nil {
		return
	}
	topic.Shutdown(ctx)
}

func shutdownSub(ctx context.Context, sub *pubsub.Subscription) {
	if sub == nil {
		return
	}
	sub.Shutdown(ctx)
}

func getDocStoreUrl(coll, id string) string {
	baseDocStoreURL := "mongo://iot-coap-platform/"
	url := baseDocStoreURL + coll
	if id != "" {
		url += "?id_field=" + id
	}
	return url
	//baseDocStoreURL := "mem://devices/deviceID"
	//return baseDocStoreURL + coll + "/" + id
}

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("err loading config file: %v", err)
	}

	ctx := context.Background()
	err = setupDataTopic(ctx)
	if err != nil {
		log.Fatalf("Err creating data topic :%v", err)
	}
	defer shutdownTopic(ctx, dataTopic)

	realtimeIngestorSub, err := setupDataSub(ctx)
	if err != nil {
		log.Fatalf("could not open data topic subscription :%v", err)
	}
	defer shutdownSub(ctx, realtimeIngestorSub)

	tsIngestorSub, err := setupDataSub(ctx)
	if err != nil {
		log.Fatalf("could not open data topic subscription :%v", err)
	}
	defer shutdownSub(ctx, tsIngestorSub)

	os.Setenv("MONGO_SERVER_URL", "mongodb://localhost:27017")
	/*deviceCollURL := getDocStoreUrl("devices", "deviceID")
	devicesColl, err := docstore.OpenCollection(ctx, deviceCollURL)
	if err != nil {
		log.Fatalf("could not open devices collection :%v", err)
	}
	defer devicesColl.Close()

	projectsCollURL := getDocStoreUrl("projects", "id")
	projectsColl, err := docstore.OpenCollection(ctx, projectsCollURL)
	if err != nil {
		log.Fatalf("could not open project collection :%v", err)
	}
	defer projectsColl.Close()

	deviceHistoryCollURL := getDocStoreUrl("device_history", "")
	deviceHistoryColl, err := docstore.OpenCollection(ctx, deviceHistoryCollURL)
	if err != nil {
		log.Fatalf("could not open device history collection :%v", err)
	}
	defer deviceHistoryColl.Close()
	*/

	//deviceStore := devices.NewDeviceDocStore(devicesColl)
	//projectStore := projects.NewProjectDocStore(projectsColl)

	db, err := bolt.Open(config.StorageConfig.URL, 0600, nil)
	if err != nil {
		log.Fatalf("could not open device local store: %v", err)
	}

	deviceStore := devices.NewDeviceLocalStore(db)
	projectStore := projects.NewProjectLocalStore(db)
	timeseriesStore := historical.NewTimeSeriesLocalStore(db)

	for _, cfg := range config.GatewayConfigs {
		if cfg.Protocol == "coap" {
			gateway := coap.NewGateway(dataTopic, &cfg)
			go gateway.Start()
		}
	}

	realtimeIngestor := realtime.NewIngestor(realtimeIngestorSub, deviceStore)
	timeseriesIngestor := timeseries.NewIngestor(tsIngestorSub, timeseriesStore)
	apiServer := api.NewServer(deviceStore, projectStore, timeseriesStore, config.APIServerConfig)

	go realtimeIngestor.Start()
	go timeseriesIngestor.Start()
	go apiServer.Start()
	//go metrics.StartMetricsExporter()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Server Started")
	<-done
	log.Info("Server Stopped")
}
