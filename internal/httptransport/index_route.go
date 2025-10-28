package httptransport

import (
	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) indexRoute() {
	if !ht.config.HTTP().AllowPage() {
		return
	}
	ht.fiber.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})
}
