package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/amatsagu/tempest"
)

// maxImageBytes caps the download so an oversized upload cannot exhaust memory.
const maxImageBytes = 8 << 20 // 8 MiB

// handleSubmit downloads the uploaded PNG, rotates it, and posts it back.
func (c *rotateCommand) handleSubmit(itx *tempest.ModalInteraction) error {
	attachments := itx.GetFileUploadAttachments(rotateUploadID)
	if len(attachments) == 0 {
		return errors.New("no PNG was uploaded")
	}

	// Acknowledge immediately; the download and rotation can take longer than
	// Discord's three second response window.
	if err := itx.Defer(true); err != nil {
		return err
	}

	raw, err := downloadImage(context.Background(), attachments[0].URL)
	if err != nil {
		return err
	}

	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decode PNG: %w", err)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, rotate90CW(src)); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}

	files := []tempest.File{{
		Name:   "rotated.png",
		Reader: bytes.NewReader(out.Bytes()),
	}}

	return sendModalFile(itx, tempest.ResponseMessageData{Content: "Rotated image."}, files)
}

// sendModalFile patches the deferred modal response with a multipart body. Master
// has no ModalInteraction.WithFiles helper, so this uses the REST client directly.
func sendModalFile(itx *tempest.ModalInteraction, data tempest.ResponseMessageData, files []tempest.File) error {
	endpoint := "/webhooks/" + itx.ApplicationID.String() + "/" + itx.Token + "/messages/@original"
	return itx.BaseClient.Rest.RequestWithFiles(http.MethodPatch, endpoint, data, files, nil)
}

func downloadImage(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download attachment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download attachment: status %d", resp.StatusCode)
	}

	// Read one byte past the cap so an oversized file is detected, not truncated.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("download attachment: %w", err)
	}

	if len(data) > maxImageBytes {
		return nil, errors.New("attachment exceeds the 8 MiB limit")
	}

	return data, nil
}

// rotate90CW rotates src 90 degrees clockwise, swapping width and height.
func rotate90CW(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, height, width))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Clockwise: the source top-left becomes the destination top-right.
			dst.Set(height-1-y, x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}
