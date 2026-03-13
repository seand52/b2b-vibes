package holded

import (
	"context"
	"fmt"
)

// ListProducts fetches all products from Holded
func (c *Client) ListProducts(ctx context.Context) ([]Product, error) {
	var products []Product
	if err := c.get(ctx, "/products", &products); err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}
	return products, nil
}

// GetProduct fetches a single product by ID
func (c *Client) GetProduct(ctx context.Context, id string) (*Product, error) {
	var product Product
	if err := c.get(ctx, "/products/"+id, &product); err != nil {
		return nil, fmt.Errorf("getting product %s: %w", id, err)
	}
	return &product, nil
}

// ProductImageData holds image binary data
type ProductImageData struct {
	Filename    string
	Data        []byte
	ContentType string
}

// GetAllProductImages fetches all images for a product
func (c *Client) GetAllProductImages(ctx context.Context, productID string) ([]ProductImageData, error) {
	filenames, err := c.listProductImages(ctx, productID)
	if err != nil {
		return nil, err
	}

	var images []ProductImageData
	for _, filename := range filenames {
		data, contentType, err := c.getProductImage(ctx, productID, filename)
		if err != nil {
			c.logger.Warn("failed to fetch image",
				"product_id", productID,
				"filename", filename,
				"error", err,
			)
			continue
		}
		if data != nil {
			images = append(images, ProductImageData{
				Filename:    filename,
				Data:        data,
				ContentType: contentType,
			})
		}
	}

	return images, nil
}

func (c *Client) listProductImages(ctx context.Context, productID string) ([]string, error) {
	var filenames []string
	if err := c.get(ctx, "/products/"+productID+"/image", &filenames); err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("listing product images: %w", err)
	}
	return filenames, nil
}

func (c *Client) getProductImage(ctx context.Context, productID, filename string) ([]byte, string, error) {
	resp, err := c.resty.R().
		SetContext(ctx).
		Get("/products/" + productID + "/image/" + filename)

	if err != nil {
		return nil, "", fmt.Errorf("fetching image: %w", err)
	}

	if resp.StatusCode() == 404 {
		return nil, "", nil
	}

	if resp.IsError() {
		c.logger.Error("holded API error fetching image",
			"status", resp.StatusCode(),
			"product_id", productID,
			"filename", filename,
		)
		return nil, "", &APIError{
			StatusCode: resp.StatusCode(),
			Message:    "failed to fetch image",
		}
	}

	contentType := resp.Header().Get("Content-Type")
	return resp.Body(), contentType, nil
}
