package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
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

	exists, err := client.Schema().ClassExistenceChecker().WithClassName("Response").Do(context.Background())
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
					DataType: []string{"Response"},
					Name:     "hasResponse",
				},
			},
			Vectorizer: "none",
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

func CreatePromptObject(vector []float32, prompt string, code string, class string) (string, error) {
	client, err := loadClient()
	if err != nil {
		return "", err
	}

	dataSchema := map[string]interface{}{
		"instruct": prompt,
		"code":     code,
	}

	weaviateObject, err := client.Data().Creator().
		WithClassName(class).
		WithProperties(dataSchema).
		WithVector(vector).
		Do(context.Background())
	if err != nil {
		return "", err
	}

	return string(weaviateObject.Object.ID), nil
}

func CreateResponseObject(vector []float32, response string, class string) (string, error) {
	client, err := loadClient()
	if err != nil {
		return "", err
	}

	dataSchema := map[string]interface{}{
		strings.ToLower(class): response,
		"rank":                 1,
	}

	weaviateObject, err := client.Data().Creator().
		WithClassName(class).
		WithProperties(dataSchema).
		WithVector(vector).
		Do(context.Background())

	if err != nil {
		return "", err
	}

	return string(weaviateObject.Object.ID), nil
}

func RetrievePromptCount(code string) (int, error) {
	client, err := loadClient()
	if err != nil {
		return 0, err
	}

	count := graphql.Field{
		Name: "code", Fields: []graphql.Field{
			{Name: "count"},
		},
	}

	where := filters.Where().
		WithPath([]string{"code"}).
		WithOperator(filters.Like).
		WithValueText(code)

	ctx := context.Background()
	result, err := client.GraphQL().Aggregate().
		WithClassName("Prompt").
		WithFields(count).
		WithWhere(where).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	if len(result.Errors) > 0 {
		return 0, errors.New(result.Errors[0].Message)
	}

	getPrompt, ok := result.Data["Aggregate"].(map[string]interface{})
	if !ok {
		return 0, errors.New("unexpected response format: 'Aggregate' field not found or not a map")
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok || len(promptData) == 0 {
		return 0, errors.New("unexpected response format: 'Prompt' field not found or not a list")
	}

	prompt := promptData[0].(map[string]interface{})
	if !ok {
		return 0, errors.New("unexpected response format: prompt data is not a map")
	}

	codeMap, ok := prompt["code"].(map[string]interface{})
	if !ok {
		return 0, errors.New("code field not found in prompt data or not a map")
	}

	countValue, ok := codeMap["count"]
	if !ok {
		return 0, errors.New("count not found in code map")
	}

	countFloat, ok := countValue.(float64)
	if !ok {
		return 0, errors.New("count is not a number")
	}

	return int(countFloat), nil
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
