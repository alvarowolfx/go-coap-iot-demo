package api

import (
	"io"
	"log"

	"github.com/gofiber/fiber"
	"gocloud.dev/docstore"
)

type ApiServer struct {
	devicesColl *docstore.Collection
}

func NewServer(devicesColl *docstore.Collection) *ApiServer {
	return &ApiServer{
		devicesColl: devicesColl,
	}
}

func (as *ApiServer) getDevices(ctx *fiber.Ctx) {
	c := ctx.Context()
	iter := as.devicesColl.Query().Get(c)
	defer iter.Stop()

	devices := make([]map[string]interface{}, 0)
	for {
		device := make(map[string]interface{})
		err := iter.Next(c, device)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("err querying devices :%v \n", err)
			break
		} else {
			devices = append(devices, device)
		}
		log.Println("Device iterator")
	}

	ctx.JSON(devices)
}

func (as *ApiServer) getDevice(ctx *fiber.Ctx) {
	c := ctx.Context()
	deviceID := ctx.Params("deviceID")
	device := make(map[string]interface{})
	device["deviceID"] = deviceID

	err := as.devicesColl.Get(c, device)
	if err != nil {
		ctx.Status(fiber.ErrBadRequest.Code)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	ctx.JSON(device)
}

func (as *ApiServer) Start() {
	app := fiber.New()

	app.Get("/devices", as.getDevices)
	app.Get("/devices/:deviceID", as.getDevice)

	app.Listen(":8080")
}
