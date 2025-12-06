package tools

import (
	"browser-agent/internal/infrastructure/browser/rodwrapper"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

var _ Tool = (*BrowserScreenshot)(nil)

type BrowserScreenshot struct {
	page   *rodwrapper.Page
	logger Logger
}

func NewBrowserScreenshotTool(page *rodwrapper.Page, logger Logger) Tool {
	return &BrowserScreenshot{
		page:   page,
		logger: logger,
	}
}

func (t *BrowserScreenshot) Name() string {
	return "screenshot"
}

func (t *BrowserScreenshot) Type() string {
	return "browser"
}

func (t *BrowserScreenshot) Description() string {
	return "Captures a full-page screenshot of the current browser page. Returns the image as a base64-encoded data URL (JPEG format). Useful for visual reference or understanding page layout."
}

func (t *BrowserScreenshot) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *BrowserScreenshot) Call(ctx context.Context, input string) (string, error) {
	t.logger.Logf("Taking full-page screenshot...")

	imgBytes, err := t.page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson.Int(80),
	})
	if err != nil {
		return "", fmt.Errorf("screenshot failed: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", fmt.Errorf("image decode failed: %w", err)
	}

	if img.Bounds().Dx() > 1024 {
		img = imaging.Resize(img, 1024, 0, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return "", fmt.Errorf("jpeg encode failed: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	result := "data:image/jpeg;base64," + b64

	t.logger.Logf("Full-page screenshot captured and encoded (%d KB, dimensions: %dx%d)",
		len(b64)/1024*3/4, img.Bounds().Dx(), img.Bounds().Dy())

	return result, nil
}
