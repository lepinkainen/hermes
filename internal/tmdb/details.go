package tmdb

import (
	"context"
	"fmt"
	"net/url"
)

// GetMovieDetails fetches detailed information for a movie by ID.
func (c *Client) GetMovieDetails(ctx context.Context, movieID int) (map[string]any, error) {
	endpoint := fmt.Sprintf("%s/movie/%d?api_key=%s", c.baseURL, movieID, url.QueryEscape(c.apiKey))
	return c.getJSONMap(ctx, endpoint)
}

// GetTVDetails fetches detailed information for a TV show by ID.
func (c *Client) GetTVDetails(ctx context.Context, tvID int, appendToResponse string) (map[string]any, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	if appendToResponse != "" {
		params.Set("append_to_response", appendToResponse)
	}
	endpoint := fmt.Sprintf("%s/tv/%d?%s", c.baseURL, tvID, params.Encode())
	return c.getJSONMap(ctx, endpoint)
}

// GetFullTVDetails fetches full TV show details including external IDs and keywords.
func (c *Client) GetFullTVDetails(ctx context.Context, tvID int) (map[string]any, error) {
	return c.GetTVDetails(ctx, tvID, "external_ids,keywords,content_ratings")
}

// GetFullMovieDetails fetches full movie details including external IDs and keywords.
func (c *Client) GetFullMovieDetails(ctx context.Context, movieID int) (map[string]any, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("append_to_response", "external_ids,keywords,credits")
	endpoint := fmt.Sprintf("%s/movie/%d?%s", c.baseURL, movieID, params.Encode())
	return c.getJSONMap(ctx, endpoint)
}

// GetMetadataByResult fetches metadata for a search result.
func (c *Client) GetMetadataByResult(ctx context.Context, result SearchResult) (*Metadata, error) {
	switch result.MediaType {
	case "movie":
		return c.getMetadataByMovieID(ctx, result.ID, false)
	case "tv":
		return c.getMetadataByTVID(ctx, result.ID, false)
	default:
		return nil, ErrInvalidMediaType
	}
}

// GetMetadataByID fetches metadata by TMDB ID and media type.
func (c *Client) GetMetadataByID(ctx context.Context, mediaID int, mediaType string) (*Metadata, error) {
	return c.getMetadataByID(ctx, mediaID, mediaType, false)
}

func (c *Client) getMetadataByID(ctx context.Context, mediaID int, mediaType string, force bool) (*Metadata, error) {
	switch mediaType {
	case "movie":
		return c.getMetadataByMovieID(ctx, mediaID, force)
	case "tv":
		return c.getMetadataByTVID(ctx, mediaID, force)
	default:
		return nil, ErrInvalidMediaType
	}
}

func (c *Client) getMetadataByMovieID(ctx context.Context, movieID int, force bool) (*Metadata, error) {
	var details map[string]any
	var err error

	if force {
		details, err = c.GetMovieDetails(ctx, movieID)
	} else {
		details, _, err = c.CachedGetMovieDetails(ctx, movieID)
	}
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{
		TMDBID:   movieID,
		TMDBType: "movie",
	}

	if runtime, ok := getInt(details, "runtime"); ok {
		metadata.Runtime = &runtime
	}

	if tags, err := c.buildGenreTags(ctx, "movie", details); err == nil {
		metadata.GenreTags = tags
	}

	return metadata, nil
}

func (c *Client) getMetadataByTVID(ctx context.Context, tvID int, force bool) (*Metadata, error) {
	var details map[string]any
	var err error

	if force {
		details, err = c.GetTVDetails(ctx, tvID, "")
	} else {
		details, _, err = c.CachedGetTVDetails(ctx, tvID, "")
	}
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{
		TMDBID:   tvID,
		TMDBType: "tv",
	}

	if runtime, ok := getEpisodeRuntime(details); ok {
		metadata.Runtime = &runtime
	}
	if episodes, ok := getInt(details, "number_of_episodes"); ok {
		metadata.TotalEpisodes = &episodes
	}
	if status, ok := getString(details, "status"); ok {
		metadata.Status = status
	}

	if tags, err := c.buildGenreTags(ctx, "tv", details); err == nil {
		metadata.GenreTags = tags
	}

	return metadata, nil
}
