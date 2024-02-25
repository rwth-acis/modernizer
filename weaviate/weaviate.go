package weaviate

import (
	"context"
	"errors"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
	"log"
	"os"
	"strings"
)

func InitSchema() error {

	client, err := loadClient()
	if err != nil {
		return err
	}

	exists, err := client.Schema().ClassExistenceChecker().WithClassName("Response").Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		classObj := &models.Class{
			Class:       "Response",
			Description: "This class contains the responses to prompts",
			Vectorizer:  "text2vec-transformers",
			ModuleConfig: map[string]interface{}{
				"text2vec-transformers": map[string]interface{}{},
			},
			Properties: []*models.Property{
				{
					DataType:    []string{"text"},
					Description: "The generated response by the LLM",
					Name:        "response",
				},
			},
		}

		err = client.Schema().ClassCreator().WithClass(classObj).Do(context.Background())
		if err != nil {
			return err
		}
		log.Println("created Response class")
	} else {
		log.Println("Response class already exists")
	}

	exists, err = client.Schema().ClassExistenceChecker().WithClassName("Prompt").Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		classObj := &models.Class{
			Class:       "Prompt",
			Description: "This class holds information regarding the prompt, code and count of queries regarding ones codebase",
			Vectorizer:  "text2vec-transformers",
			ModuleConfig: map[string]interface{}{
				"text2vec-transformers": map[string]interface{}{},
			},
			Properties: []*models.Property{
				{
					DataType:    []string{"text"},
					Description: "The specific instruct or question prepended to the code",
					Name:        "instruct",
					ModuleConfig: map[string]interface{}{
						"text2vec-transformers": map[string]interface{}{
							"skip": true,
						},
					},
				},
				{
					DataType:    []string{"text"},
					Description: "The code which is targeted in the prompt",
					Name:        "code",
				},
				{
					DataType: []string{"Response"},
					Name:     "hasResponse",
					ModuleConfig: map[string]interface{}{
						"text2vec-transformers": map[string]interface{}{
							"skip": true,
						},
					},
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
		log.Println("created Prompt class")
	} else {
		log.Println("Prompt class already exists")
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

func CreatePromptObject(prompt string, code string, class string) (string, error) {
	client, err := loadClient()
	if err != nil {
		return "", err
	}

	dataSchema := map[string]interface{}{
		"instruct": prompt,
		"code":     code,
		"rank":     1,
	}

	weaviateObject, err := client.Data().Creator().
		WithClassName(class).
		WithProperties(dataSchema).
		Do(context.Background())
	if err != nil {
		return "", err
	}

	return string(weaviateObject.Object.ID), nil
}

func UpdateRankPrompt(prompt map[string]interface{}, upvote bool) error {
	id, ok := prompt["id"].(string)
	if !ok {
		return errors.New("ID not found in request body")
	}

	log.Printf("%v\n", id)

	client, err := loadClient()
	if err != nil {
		return err
	}

	promptProperties, err := RetrieveProperties(id)
	if err != nil {
		return err
	}

	var rank int
	if upvote {
		rank = promptProperties.Rank + 1
	} else {
		rank = promptProperties.Rank - 1
	}

	err = client.Data().Updater().
		WithMerge().
		WithID(id).
		WithClassName("Prompt").
		WithProperties(map[string]interface{}{
			"rank": rank,
		}).
		Do(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func CreateResponseObject(response string, class string) (string, error) {
	client, err := loadClient()
	if err != nil {
		return "", err
	}

	dataSchema := map[string]interface{}{
		strings.ToLower(class): response,
	}

	weaviateObject, err := client.Data().Creator().
		WithClassName(class).
		WithProperties(dataSchema).
		Do(context.Background())

	if err != nil {
		return "", err
	}

	return string(weaviateObject.Object.ID), nil
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
		Host:       os.Getenv("WEAVIATE_HOST"),
		Scheme:     os.Getenv("WEAVIATE_SCHEME"),
		AuthConfig: auth.ApiKey{Value: os.Getenv("WEAVIATE_KEY")},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func CreateReferences(PromptID string, ResponseID string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	err = client.Data().ReferenceCreator().
		WithClassName("Prompt").
		WithID(PromptID).
		WithReferenceProperty("hasResponse").
		WithReference(client.Data().ReferencePayloadBuilder().
			WithClassName("Response").
			WithID(ResponseID).
			Payload()).
		Do(context.Background())
	if err != nil {
		return err
	}

	return nil
}
