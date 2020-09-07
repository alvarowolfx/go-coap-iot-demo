package api

import (
	"time"

	"github.com/gofiber/fiber"
)

func (as *ApiServer) getDeviceHistory(ctx *fiber.Ctx) {

	deviceID := ctx.Params("deviceID")
	end := time.Now()
	start := end.Add(time.Hour * 24 * 7 * -1)

	points, err := as.timeseriesStore.GetDataPointsInRange(
		ctx.Context(),
		"device",
		deviceID,
		start,
		end,
	)

	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	ctx.JSON(points)
}
