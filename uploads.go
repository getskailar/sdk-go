package skailar

import "context"

// UploadsService is the storage-uploads resource, accessed as Client.Uploads.
type UploadsService struct {
	// Images is the image-uploads sub-resource.
	Images *ImageUploadsService
	// Files is the document-uploads sub-resource.
	Files *FileUploadsService

	client *Client
}

// ImageUploadsService is the image-uploads resource, accessed as
// Client.Uploads.Images.
type ImageUploadsService struct {
	client *Client
}

// Create uploads a base64-encoded image (without a data: prefix) and returns
// its stored URL, ready to embed as a vision input in a chat completion. This
// is a side-effecting call and is not retried on 5xx responses.
func (s *ImageUploadsService) Create(ctx context.Context, base64Data string, contentType ImageContentType) (*UploadResponse, error) {
	req := UploadRequest{Base64: base64Data, ContentType: string(contentType)}
	var out UploadResponse
	if err := s.client.postJSON(ctx, "v1/uploads/images", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FileUploadsService is the document-uploads resource, accessed as
// Client.Uploads.Files.
type FileUploadsService struct {
	client *Client
}

// Create uploads a base64-encoded document (without a data: prefix) and returns
// its stored URL. This is a side-effecting call and is not retried on 5xx
// responses.
func (s *FileUploadsService) Create(ctx context.Context, base64Data string, contentType FileContentType) (*UploadResponse, error) {
	req := UploadRequest{Base64: base64Data, ContentType: string(contentType)}
	var out UploadResponse
	if err := s.client.postJSON(ctx, "v1/uploads/files", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
