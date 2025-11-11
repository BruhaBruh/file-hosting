package httptransport

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) fileRoute() {
	ht.fiber.Get("/file/:file", func(c *fiber.Ctx) error {
		file, err := ht.fileHostingService.GetFile(c.UserContext(), c.Params("file"))
		if err != nil {
			return err
		}
		c.Response().Header.Set(fiber.HeaderContentType, file.Metadata.MimeType)
		c.Response().Header.Set(fiber.HeaderContentDisposition, fmt.Sprintf("inline; filename=\"%s\"", file.Metadata.Name))
		for key, value := range file.Metadata.Meta {
			header := fmt.Sprintf("X-Meta-%s", key)
			for i := range value {
				c.Response().Header.Add(header, value[i])
			}
		}

		return c.Send(file.Content)
	})
}
