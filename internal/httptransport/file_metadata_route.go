package httptransport

import (
	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) fileMetadataRoute() {
	ht.fiber.Get("/:file/metadata", func(c *fiber.Ctx) error {
		metadata, err := ht.fileHostingService.GetFileMetadata(c.UserContext(), c.Params("file"))
		if err != nil {
			return err
		}
		return c.JSON(metadata)
	})
}
