package httptransport

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) fileRoute() {
	ht.fiber.Get("/file/:file", func(c *fiber.Ctx) error {
		data, metadata, err := ht.fileHostingService.GetFile(c.UserContext(), c.Params("file"))
		if err != nil {
			return err
		}
		c.Response().Header.Set(fiber.HeaderContentType, metadata.MimeType)
		c.Response().Header.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", metadata.Name))
		for key, value := range metadata.Meta {
			header := fmt.Sprintf("X-Meta-%s", key)
			for i := range value {
				c.Response().Header.Add(header, value[i])
			}
		}

		return c.Send(data)
	})
}
