package weaviate

import (
	"context"
	"encoding/json"
	"os"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
)

func RetrieveSchema() ([]byte, error) {

	cfg := weaviate.Config{
		Host:       os.Getenv("WEAVIATE_HOST"), // Replace with your endpoint
		Scheme:     "http",
		AuthConfig: auth.ApiKey{Value: os.Getenv("WEAVIATE_KEY")},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Retrieve the schema
	allSchemas, err := client.Schema().Getter().Do(context.Background())
	if err != nil {
		return nil, err
	}

	jsonSchema, err := json.MarshalIndent(allSchemas, "", "  ")
	if err != nil {
		return nil, err
	}

	return jsonSchema, nil
}
