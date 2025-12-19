package httptransport

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) fileRoute() {
	ht.fiber.Get("/file/:file", func(c *fiber.Ctx) error {
		metadata, err := ht.fileHostingService.GetFileMetadata(c.UserContext(), c.Params("file"))
		if err != nil {
			return err
		}

		etag := `"` + metadata.Sha1 + `"`
		if inm := c.Get(fiber.HeaderIfNoneMatch); inm != "" {
			if inm == "*" || strings.Contains(inm, etag) {
				return c.SendStatus(fiber.StatusNotModified)
			}
		}

		file, err := ht.fileHostingService.GetFile(c.UserContext(), c.Params("file"))
		if err != nil {
			return err
		}

		c.Response().Header.Set(fiber.HeaderETag, etag)
		c.Response().Header.Set(
			fiber.HeaderCacheControl,
			"public, max-age=3600",
		)
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
