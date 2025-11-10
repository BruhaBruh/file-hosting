package httptransport

import (
	"fmt"
	"io"
	"strings"

	"github.com/bruhabruh/file-hosting/internal/app/apperr"
	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/bruhabruh/file-hosting/pkg/logging"
	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) uploadPublicRoute() {
	ht.fiber.Post("/upload", func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			logging.L(c.UserContext()).Warn("failed to get file from form", logging.ErrAttr(err))
			return apperr.ErrBadRequest.WithMessage("Fail get file")
		}

		file, err := fileHeader.Open()
		if err != nil {
			logging.L(c.UserContext()).Warn("failed to open file", logging.ErrAttr(err))
			return apperr.ErrBadRequest.WithMessage("Fail to open file")
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			logging.L(c.UserContext()).Warn("failed to read file", logging.ErrAttr(err))
			return apperr.ErrBadRequest.WithMessage("Fail to read file")
		}

		metadata := &domain.FileMetadata{
			Name:     fileHeader.Filename,
			MimeType: fileHeader.Header.Get(fiber.HeaderContentType),
			Meta:     make(map[string][]string),
		}

		for key, value := range c.GetReqHeaders() {
			lowerKey := strings.ToLower(key)
			if strings.HasPrefix(lowerKey, "x-meta-") {
				keyForMeta, _ := strings.CutPrefix(lowerKey, "x-meta-")
				metadata.Meta[keyForMeta] = value
			}
		}

		fileName, _, err := ht.fileHostingService.UploadFileWithGenerativeName(c.UserContext(), content, metadata, c.Query("d"))
		if err != nil {
			return err
		}

		link := fmt.Sprintf("%s/%s", ht.config.Origin(), fileName)

		return c.SendString(link)
	})
}

func (ht *HttpTransport) uploadPrivateRoute() {
	ht.fiber.Post("/upload/:file", ht.authorizationMiddleware(), func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			logging.L(c.UserContext()).Warn("failed to get file from form", logging.ErrAttr(err))
			return apperr.ErrBadRequest.WithMessage("Fail get file")
		}

		file, err := fileHeader.Open()
		if err != nil {
			logging.L(c.UserContext()).Warn("failed to open file", logging.ErrAttr(err))
			return apperr.ErrBadRequest.WithMessage("Fail to open file")
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			logging.L(c.UserContext()).Warn("failed to read file", logging.ErrAttr(err))
			return apperr.ErrBadRequest.WithMessage("Fail to read file")
		}

		metadata := &domain.FileMetadata{
			Name:     c.Params("file"),
			MimeType: fileHeader.Header.Get(fiber.HeaderContentType),
			Meta:     make(map[string][]string),
		}

		for key, value := range c.GetReqHeaders() {
			lowerKey := strings.ToLower(key)
			if strings.HasPrefix(lowerKey, "x-meta-") {
				keyForMeta, _ := strings.CutPrefix(lowerKey, "x-meta-")
				metadata.Meta[keyForMeta] = value
			}
		}

		fileName, _, err := ht.fileHostingService.UploadFile(c.UserContext(), content, metadata, c.Query("d"))
		if err != nil {
			return err
		}

		link := fmt.Sprintf("%s/%s", ht.config.Origin(), fileName)

		return c.SendString(link)
	})
}
