package httptransport

import (
	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) filesRoute() {
	ht.fiber.Get("/files", ht.authorizationMiddleware(), func(c *fiber.Ctx) error {
		files, err := ht.fileHostingService.GetFiles(c.UserContext())
		if err != nil {
			return err
		}

		return c.JSON(files)
	})
}
