package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
)

type FigmaResponse struct {
	Comments []Comment `json:"comments"`
}

type Comment struct {
	ID         string     `json:"id"`
	FileKey    string     `json:"file_key"`
	ParentID   string     `json:"parent_id"`
	User       *User      `json:"user"`
	CreatedAt  *time.Time `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	OrderID    string     `json:"order_id"`
	Message    string     `json:"message"`
}

type User struct {
	Handle string `json:"handle"`
	ImgURL string `json:"img_url"`
	ID     string `json:"id"`
}

// Resolved counts the number of comments marked as resolved
func (f FigmaResponse) Resolved() (resolved int) {
	for _, c := range f.Comments {
		if c.ResolvedAt != nil {
			resolved++
		}
	}
	return
}

const (
	projectID = "atb-mobility-platform"
	figmaURL  = "https://api.figma.com/v1/files"
)

var (
	bc          *bigquery.Client
	ctx         context.Context
	figmaSecret string
)

func init() {
	ctx = context.Background()
	figmaSecret = os.Getenv("FIGMA_SECRET")

	var err error
	bc, err = bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatal(err)
	}
}

func FigmaComments(w http.ResponseWriter, r *http.Request) {
	// Get file from query params
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "no file specified", http.StatusBadRequest)
		return
	}

	// Build request
	url := fmt.Sprintf("%s/%s/comments", figmaURL, file)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		http.Error(w, "req: "+err.Error(), http.StatusInternalServerError)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Figma-Token", figmaSecret)

	// Send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "http: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Verify response
	if res.StatusCode != http.StatusOK {
		http.Error(w, "invalid response: "+res.Status, http.StatusInternalServerError)
		return
	}

	// Read body
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "read: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Decode response
	f := FigmaResponse{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		http.Error(w, "unmarshal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate BigQuery insert statement
	ins := generateInsertStatement(file, time.Now(), len(f.Comments), f.Resolved())
	q := bc.Query(ins)
	_, err = q.Run(ctx)
	if err != nil {
		http.Error(w, "bigquery.Run: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusOK)
	return
}

func generateInsertStatement(file string, date time.Time, comments, resolved int) string {
	return fmt.Sprintf("INSERT INTO"+
		"`atb-mobility-platform.figma_comments_okr.%s`"+
		"VALUES(DATE(%d, %d, %d), %d, %d)", file, date.Year(), date.Month(), date.Day(), comments, resolved)
}
