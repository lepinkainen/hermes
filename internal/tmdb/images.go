package tmdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// GetCoverURLByID fetches the cover image URL by TMDB ID and media type.
func (c *Client) GetCoverURLByID(ctx context.Context, mediaID int, mediaType string) (string, error) {
	var details map[string]any
	var err error

	// Use full details endpoints to share cache with GetMetadataByID
	switch mediaType {
	case "movie":
		details, _, err = c.CachedGetFullMovieDetails(ctx, mediaID, false)
	case "tv":
		details, _, err = c.CachedGetFullTVDetails(ctx, mediaID, false)
	default:
		return "", ErrInvalidMediaType
	}
	if err != nil {
		return "", err
	}

	posterPath, _ := getString(details, "poster_path")
	if posterPath == "" {
		return "", ErrNoPoster
	}
	return c.ImageURL(posterPath), nil
}

// ImageURL constructs the full image URL from a poster path.
func (c *Client) ImageURL(posterPath string) string {
	return c.imageBaseURL + posterPath
}

// GetCoverAndMetadataByID fetches both cover URL and metadata by ID.
func (c *Client) GetCoverAndMetadataByID(ctx context.Context, mediaID int, mediaType string) (string, *Metadata, error) {
	cover, err := c.GetCoverURLByID(ctx, mediaID, mediaType)
	if err != nil {
		if errors.Is(err, ErrNoPoster) {
			// still return metadata even without a poster
			meta, metaErr := c.GetMetadataByID(ctx, mediaID, mediaType)
			return "", meta, metaErr
		}
		return "", nil, err
	}
	meta, err := c.GetMetadataByID(ctx, mediaID, mediaType)
	if err != nil {
		return cover, nil, err
	}
	return cover, meta, nil
}

// GetCoverAndMetadataByResult fetches both cover URL and metadata from a search result.
func (c *Client) GetCoverAndMetadataByResult(ctx context.Context, result SearchResult) (string, *Metadata, error) {
	cover := c.ImageURL(result.PosterPath)
	meta, err := c.GetMetadataByResult(ctx, result)
	if err != nil {
		return cover, nil, err
	}
	return cover, meta, nil
}

// DownloadAndResizeImage downloads an image and resizes it to the specified width.
func (c *Client) DownloadAndResizeImage(ctx context.Context, imageURL, savePath string, maxWidth int) error {
	if maxWidth <= 0 {
		maxWidth = defaultMaxWidth
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d downloading image", resp.StatusCode)
	}

	img, err := imaging.Decode(resp.Body, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	width := img.Bounds().Dx()
	if width > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
		return err
	}

	return imaging.Save(img, savePath, imaging.JPEGQuality(85))
}
