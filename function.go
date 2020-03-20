package function

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
)

const (
	projectID = "atb-mobility-platform"
)

var (
	figmaClient *FigmaClient
	bc          *bigquery.Client
	ctx         context.Context
	figmaSecret string
	tableName   string
)

func init() {
	ctx = context.Background()
	figmaSecret = os.Getenv("FIGMA_SECRET")
	tableName = os.Getenv("TABLE_NAME")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	figmaClient = NewFigmaClient(client, figmaSecret)

	var err error
	bc, err = bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatal(err)
	}
}

// FigmaComments tries to get the comments that belong to a file in Figma, using the Figma REST API.
func FigmaComments(w http.ResponseWriter, r *http.Request) {
	// Get file from query params
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "no file specified", http.StatusBadRequest)
		return
	}

	// Get comments for file
	c, err := figmaClient.Comments(file)
	if err != nil {
		http.Error(w, "figma: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate BigQuery insert statement
	ins := generateQuery(file, time.Now(), len(c.Comments), c.Resolved())
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

func generateQuery(file string, date time.Time, comments, resolved int) string {
	return fmt.Sprintf(`
		INSERT INTO`+
		"`%s.%s.%s`"+
		`VALUES(%s, %d, %d)`,
		projectID, tableName, file, "CURRENT_TIMESTAMP()", comments, resolved)
}
