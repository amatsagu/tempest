package discord

// File is one attachment sent with a message. The bytes are buffered so a retry
// can resend the exact same body.
type File struct {
	// Name is the file name the recipient sees, for example "rotated.png".
	Name string
	// ContentType is the media type of Data; empty falls back to
	// application/octet-stream.
	ContentType string
	// Data is the whole file.
	Data []byte
}
