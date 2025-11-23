package tmdb

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) buildGenreTags(ctx context.Context, mediaType string, details map[string]any) ([]string, error) {
	rawGenres, ok := details["genres"].([]any)
	if !ok || len(rawGenres) == 0 {
		return nil, nil
	}

	genres, err := c.getGenres(ctx, mediaType)
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(rawGenres))
	for _, raw := range rawGenres {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, ok := getInt(m, "id")
		if !ok {
			continue
		}
		name, ok := genres[id]
		if !ok {
			continue
		}
		tags = append(tags, fmt.Sprintf("%s/%s", mediaType, sanitizeGenreName(name)))
	}

	return tags, nil
}

func (c *Client) getGenres(ctx context.Context, mediaType string) (map[int]string, error) {
	c.mu.RLock()
	if genres, ok := c.genreCache[mediaType]; ok {
		c.mu.RUnlock()
		return genres, nil
	}
	c.mu.RUnlock()

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	endpoint := fmt.Sprintf("%s/genre/%s/list?%s", c.baseURL, mediaType, params.Encode())

	var response struct {
		Genres []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}

	if err := c.getJSON(ctx, endpoint, &response); err != nil {
		return nil, err
	}

	result := make(map[int]string, len(response.Genres))
	for _, g := range response.Genres {
		result[g.ID] = g.Name
	}

	c.mu.Lock()
	c.genreCache[mediaType] = result
	c.mu.Unlock()

	return result, nil
}
