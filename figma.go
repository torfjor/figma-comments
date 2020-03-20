package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// CommentsResponse represents the response returned from the Figma API for the :file/comments endpoint
type CommentsResponse struct {
	Comments []comment `json:"comments"`
}

type comment struct {
	ID         string     `json:"id"`
	FileKey    string     `json:"file_key"`
	ParentID   string     `json:"parent_id"`
	User       *user      `json:"user"`
	CreatedAt  *time.Time `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	OrderID    string     `json:"order_id"`
	Message    string     `json:"message"`
}

type user struct {
	Handle string `json:"handle"`
	ImgURL string `json:"img_url"`
	ID     string `json:"id"`
}

// Resolved counts the number of comments marked as resolved
func (f CommentsResponse) Resolved() (resolved int) {
	for _, c := range f.Comments {
		if c.ResolvedAt != nil {
			resolved++
		}
	}
	return
}

// FigmaClient represents a HTTP REST client for Figma
type FigmaClient struct {
	client  *http.Client
	secret  string
	baseURL string
}

// NewFigmaClient returns a configured FigmaClient
func NewFigmaClient(client *http.Client, secret string) *FigmaClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &FigmaClient{
		client:  client,
		secret:  secret,
		baseURL: "https://api.figma.com/v1/files",
	}
}

// Comments fetches the comments for a file in the Figma API
func (f *FigmaClient) Comments(file string) (*CommentsResponse, error) {
	// Build request
	url := fmt.Sprintf("%s/%s/comments", f.baseURL, file)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Figma-Token", f.secret)

	// Send request
	res, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Verify response
	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	// Read body
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Decode response
	r := CommentsResponse{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
