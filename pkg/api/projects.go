package api

import (
	"github.com/gofiber/fiber"
)

type createProjectRequest struct {
	Name string `json:"name" form:"name"`
}

func (as *ApiServer) createProject(ctx *fiber.Ctx) {

	req := &createProjectRequest{}
	if err := ctx.BodyParser(req); err != nil {
		ctx.
			Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"message": "Missing project name"})
		return
	}

	err := as.projectStore.CreateProject(ctx.Context(), req.Name)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	project, err := as.projectStore.GetProjectByID(ctx.Context(), req.Name)
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	ctx.JSON(project)
}

func (as *ApiServer) registerDeviceOnProject(ctx *fiber.Ctx) {
	deviceID := ctx.Params("deviceID")
	project := ctx.Params("project")

	err := as.deviceStore.RegisterDeviceToProject(ctx.Context(), deviceID, project)

	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		ctx.JSON(fiber.Map{"message": err.Error()})
		return
	}

	ctx.JSON(fiber.Map{"message": "associated"})
}
