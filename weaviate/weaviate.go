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

func InitSchema() error {

	client, err := loadClient()
	if err != nil {
		return err
	}

	exists, err := client.Schema().ClassExistenceChecker().WithClassName("Prompt").Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		classObj := &models.Class{
			Class:       "Prompt",
			Description: "This class holds information regarding the prompt, code and count of queries regarding ones codebase",
			Vectorizer:  "none",
			Properties: []*models.Property{
				{
					DataType:    []string{"text"},
					Description: "The specific instruct or question prepended to the code",
					Name:        "instruct",
				},
				{
					DataType:    []string{"text"},
					Description: "The code which is targeted in the prompt",
					Name:        "code",
				},
				{
					DataType:    []string{"int"},
					Description: "number of times that this particular code has been referenced",
					Name:        "count",
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

	exists, err = client.Schema().ClassExistenceChecker().WithClassName("Response").Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		classObj := &models.Class{
			Class:       "Response",
			Description: "This class contains the responses to prompts",
			Vectorizer:  "none",
			Properties: []*models.Property{
				{
					DataType:    []string{"text"},
					Description: "The generated response by the LLM",
					Name:        "response",
				},
				{
					DataType:    []string{"int"},
					Description: "The relative rank for this response against other ones regarding the same code",
					Name:        "rank",
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

func createClass(className, description, vectorizer string, properties []*models.Property) error {

	client, err := loadClient()
	if err != nil {
		return err
	}

	classObj := &models.Class{
		Class:       className,
		Description: description,
		Vectorizer:  vectorizer,
		Properties:  properties,
	}

	// Create or update the class in Weaviate
	err = client.Schema().ClassCreator().WithClass(classObj).Do(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func CreatePromptObject(vector []float32, prompt string, code string, class string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	dataSchema := map[string]interface{}{
		strings.ToLower(class): prompt,
		"code":                 code,
		"count":                1,
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

func CreateResponseObject(vector []float32, response string, class string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	dataSchema := map[string]interface{}{
		strings.ToLower(class): response,
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
