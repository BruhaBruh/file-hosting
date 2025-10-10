package httptransport

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bruhabruh/file-hosting/internal/domain"
	"github.com/gofiber/fiber/v2"
)

func (ht *HttpTransport) uploadPublicRoute() {
	ht.fiber.Post("/upload", func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return err
		}

		file, err := fileHeader.Open()
		if err != nil {
			return err
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return err
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

		if len(metadata.MimeType) == 0 {
			metadata.MimeType = http.DetectContentType(data)
		}

		if len(metadata.MimeType) == 0 {
			metadata.MimeType = http.DetectContentType(data)
		}

		fileName, err := ht.fileHostingService.UploadFileWithGenerativeName(c.UserContext(), data, metadata)
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
			return err
		}

		file, err := fileHeader.Open()
		if err != nil {
			return err
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return err
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

		if len(metadata.MimeType) == 0 {
			metadata.MimeType = http.DetectContentType(data)
		}

		if len(metadata.MimeType) == 0 {
			metadata.MimeType = http.DetectContentType(data)
		}

		fileName, err := ht.fileHostingService.UploadFile(c.UserContext(), data, metadata)
		if err != nil {
			return err
		}

		link := fmt.Sprintf("%s/%s", ht.config.Origin(), fileName)

		return c.SendString(link)
	})
}
