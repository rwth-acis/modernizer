package weaviate

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
)

func RetrieveSchema() ([]byte, error) {

	client, err := loadClient()
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

func CreateSchema(className string, description string) error {

	client, err := loadClient()
	if err != nil {
		return err
	}

	exists, err := client.Schema().ClassExistenceChecker().WithClassName(className).Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		classObj := &models.Class{
			Class:       className,
			Description: description,
			Vectorizer:  "none",
			Properties: []*models.Property{
				{
					DataType:    []string{"string"},
					Description: description,
					Name:        className,
				},
			},
		}

		err = client.Schema().ClassCreator().WithClass(classObj).Do(context.Background())
		if err != nil {
			return err
		}
		log.Println("created class")
	} else {
		log.Println("class already exists")
	}

	return nil
}

func CreateObject(vector []float32, body string, class string) error {

	client, err := loadClient()
	if err != nil {
		return err
	}

	dataSchema := map[string]interface{}{
		strings.ToLower(class): body,
	}

	_, err = client.Data().Creator().
		WithClassName(class).
		WithProperties(dataSchema).
		WithVector(vector).
		Do(context.Background())

	if err != nil {
		return err
	}

	return nil
}

func loadClient() (*weaviate.Client, error) {
	cfg := weaviate.Config{
		Host:       os.Getenv("WEAVIATE_HOST"), // Replace with your endpoint
		Scheme:     "http",
		AuthConfig: auth.ApiKey{Value: os.Getenv("WEAVIATE_KEY")},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}
