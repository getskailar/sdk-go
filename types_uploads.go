package skailar

// ImageContentType is the MIME type of an image uploaded via
// [ImageUploadsService.Create].
type ImageContentType string

const (
	// ImagePNG is image/png.
	ImagePNG ImageContentType = "image/png"
	// ImageJPEG is image/jpeg.
	ImageJPEG ImageContentType = "image/jpeg"
	// ImageGIF is image/gif.
	ImageGIF ImageContentType = "image/gif"
	// ImageWebP is image/webp.
	ImageWebP ImageContentType = "image/webp"
)

// FileContentType is the MIME type of a document uploaded via
// [FileUploadsService.Create].
type FileContentType string

const (
	// FilePDF is application/pdf.
	FilePDF FileContentType = "application/pdf"
	// FileText is text/plain.
	FileText FileContentType = "text/plain"
)

// UploadRequest is the wire body for the upload endpoints.
type UploadRequest struct {
	// Base64 is the base64-encoded payload, without a data: prefix.
	Base64 string `json:"base64"`
	// ContentType is the MIME type of the payload.
	ContentType string `json:"content_type"`
}

// UploadResponse is the response of the upload endpoints.
type UploadResponse struct {
	// URL is the Skailar-relative URL of the stored asset, ready to embed in
	// subsequent calls.
	URL string `json:"url"`
	// ContentType is the stored MIME type.
	ContentType string `json:"content_type"`
}
