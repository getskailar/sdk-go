package skailar

import "context"

// ImagesService is the image-generation resource, accessed as Client.Images.
type ImagesService struct {
	client *Client
}

// Generate creates one or more images from a prompt. This is a billable,
// side-effecting call and is not retried on 5xx responses.
func (s *ImagesService) Generate(ctx context.Context, req ImageGenerationRequest) (*ImageGenerationResponse, error) {
	var out ImageGenerationResponse
	if err := s.client.postJSON(ctx, "v1/images/generations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
