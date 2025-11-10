package httptransport

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) deleteFileRoute() {
	ht.fiber.Delete("/file/:file", ht.authorizationMiddleware(), func(c *fiber.Ctx) error {
		err := ht.fileHostingService.DeleteFile(c.UserContext(), c.Params("file"))
		if err != nil {
			return err
		}

		return c.SendStatus(http.StatusNoContent)
	})
}
