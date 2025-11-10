package httptransport

import (
	"net/http"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

type fileRename struct {
	Name string `json:"name"`
}

func (ht *HttpTransport) renameFileRoute() {
	ht.fiber.Patch("/file/:file", ht.authorizationMiddleware(), func(c *fiber.Ctx) error {
		rawBody := c.BodyRaw()

		var fileRename fileRename
		if err := json.Unmarshal(rawBody, &fileRename); err != nil {
			return apperr.ErrBadRequest.WithMessage("invalid json")
		}

		err := ht.fileHostingService.RenameFile(c.UserContext(), c.Params("file"), fileRename.Name)
		if err != nil {
			return err
		}

		return c.SendStatus(http.StatusNoContent)
	})
}
