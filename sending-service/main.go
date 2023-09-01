package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/api/idtoken"
)

func httpClientWithIDToken(ctx context.Context, audience string) (*http.Client, error) {
	client, err := idtoken.NewClient(ctx, audience)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func main() {
	receivingServiceURL := os.Getenv("RECEIVING_SERVICE_URL")
	if receivingServiceURL == "" {
		log.Fatal("RECEIVING_SERVICE_URL environment variable is not set")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := httpClientWithIDToken(ctx, receivingServiceURL)
		if err != nil {
			log.Printf("Failed to create authenticated client: %v", err)
			http.Error(w, "Failed to create authenticated client", http.StatusInternalServerError)
			return
		}

		resp, err := client.Get(receivingServiceURL)
		if err != nil {
			log.Printf("Failed to make request: %v", err)
			http.Error(w, "Failed to make request", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Response from receiving service: %s", string(body))
	})

	http.ListenAndServe(":8080", nil)
}
