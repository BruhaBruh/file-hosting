package httptransport

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) healthRoute() {
	ht.fiber.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})
}
