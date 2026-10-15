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

	"github.com/amatsagu/tempest/discord"
	"github.com/amatsagu/tempest/rest"
	"github.com/amatsagu/tempest/route"
)

// maxImageBytes caps the download so an oversized upload cannot exhaust memory.
const maxImageBytes = 8 << 20 // 8 MiB

// ErrAnswered marks a submission that already answered the interaction through
// the REST client. main's error hook ignores it: it is not a failure, it is how
// a file reply reports success, because the runtime's own responder cannot send
// multipart bytes.
var ErrAnswered = errors.New("image: interaction already answered through REST")

// handleSubmit downloads the uploaded PNG, rotates it and posts it back.
func (c *rotateCommand) handleSubmit(ctx context.Context, itx *discord.ModalInteraction, _ *route.Params) error {
	ids, ok := itx.Data.FileUpload(rotateUploadID)
	if !ok || len(ids) == 0 {
		return errors.New("no PNG was uploaded")
	}
	if itx.Data.Resolved == nil {
		return errors.New("uploaded file did not resolve")
	}

	attachment, ok := itx.Data.Resolved.Attachments[ids[0]]
	if !ok {
		return errors.New("uploaded attachment was not resolved")
	}

	raw, err := downloadImage(ctx, attachment.URL)
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

	client := c.client.REST()
	if client == nil {
		return errors.New("no REST client configured; Config.Token is required")
	}

	response := discord.Response{
		Type: discord.ResponseChannelMessageWithSource,
		Data: discord.ResponseMessageData{
			Flags: discord.MessageFlagEphemeral,
			// The attachment's ID is the index of the matching files[0] part.
			Attachments: []discord.Attachment{{
				ID:       discord.AttachmentID{Snowflake: 0},
				Filename: "rotated.png",
			}},
		},
	}

	// A file reply cannot go through itx.Reply: the runtime's responder sends
	// JSON, and Discord only accepts attachment bytes in a multipart/form-data
	// body. So the PNG is posted straight to the interaction callback endpoint
	// with the shared REST client, which reuses the runtime's authentication,
	// rate limiting and retries.
	_, err = client.RequestWithFiles(ctx, http.MethodPost,
		"/interactions/"+itx.ID.String()+"/"+itx.Token+"/callback",
		response,
		[]rest.File{{Name: "rotated.png", ContentType: "image/png", Data: out.Bytes()}},
		"",
	)
	if err != nil {
		return err
	}

	// The interaction already has its response, so ErrAnswered stops the runtime
	// from sending a second, rejected one.
	return ErrAnswered
}

// downloadImage fetches an attachment with a short timeout and an 8 MiB cap.
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

	// Read one byte past the cap so an oversized file is detected, not silently
	// truncated.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("download attachment: %w", err)
	}
	if len(data) > maxImageBytes {
		return nil, errors.New("attachment exceeds the 8 MiB limit")
	}

	return data, nil
}

// rotate90CW returns src rotated 90 degrees clockwise. The standard library has
// no rotation, so this walks the pixels once into a destination RGBA; width and
// height swap.
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
