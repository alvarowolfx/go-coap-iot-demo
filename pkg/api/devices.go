package api

import (
	"com.aviebrantz.coap-demo/pkg/core/store/devices"
	"github.com/gofiber/fiber"
)

func (as *ApiServer) getDevicesByProject(ctx *fiber.Ctx) {
	project := ctx.Params("project")
	list, err := as.deviceStore.ListDevicesForProject(ctx.Context(), project)

	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	if list == nil {
		list = make([]*devices.Device, 0)
	}

	ctx.JSON(list)
}

func (as *ApiServer) getDeviceByProject(ctx *fiber.Ctx) {
	c := ctx.Context()
	deviceID := ctx.Params("deviceID")
	project := ctx.Params("project")

	device, err := as.deviceStore.GetDeviceByID(c, deviceID)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	if device == nil {
		ctx.Status(fiber.StatusNotFound)
		ctx.JSON(fiber.Map{"message": "not found"})
		return
	}

	if device.ProjectID != project {
		ctx.Status(fiber.StatusNotFound)
		ctx.JSON(fiber.Map{"message": "not found"})
		return
	}

	ctx.JSON(device)
}
