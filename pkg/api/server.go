package api

import (
	"strconv"

	"com.aviebrantz.coap-demo/pkg/config"
	"com.aviebrantz.coap-demo/pkg/core/store/devices"
	"com.aviebrantz.coap-demo/pkg/core/store/historical"
	"com.aviebrantz.coap-demo/pkg/core/store/projects"
	"github.com/gofiber/fiber"
)

type ApiServer struct {
	deviceStore     devices.DeviceStore
	projectStore    projects.ProjectStore
	timeseriesStore historical.TimeSeriesStore
	config          config.APIServerConfig
}

func NewServer(
	deviceStore devices.DeviceStore,
	projectStore projects.ProjectStore,
	timeseriesStore historical.TimeSeriesStore,
	config config.APIServerConfig,
) *ApiServer {
	return &ApiServer{
		deviceStore:     deviceStore,
		projectStore:    projectStore,
		timeseriesStore: timeseriesStore,
		config:          config,
	}
}

func (as *ApiServer) Start() {
	app := fiber.New()

	app.Post("/project", as.createProject)
	app.Post("/:project/devices/:deviceID", as.registerDeviceOnProject)
	//app.Post("/:project/certificates", as.registerRootCert)

	app.Get("/:project/devices", as.getDevicesByProject)
	app.Get("/:project/devices/:deviceID", as.getDeviceByProject)
	app.Get("/:project/devices/:deviceID/history", as.getDeviceHistory)

	app.Listen(":" + strconv.Itoa(as.config.Port))
}
