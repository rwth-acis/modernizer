package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
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

type PromptProperties struct {
	Code        string                   `json:"code"`
	HasResponse []map[string]interface{} `json:"hasResponse"`
	Instruct    string                   `json:"instruct"`
	Rank        int                      `json:"rank"`
}

func GetProperties(id string) (PromptProperties, error) {

	client, err := loadClient()
	if err != nil {
		return PromptProperties{}, err
	}

	objects, err := client.Data().ObjectsGetter().
		WithID(id).
		WithClassName("Prompt").
		Do(context.Background())
	if err != nil {
		return PromptProperties{}, err
	}

	properties := objects[0].Properties

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return PromptProperties{}, err
	}

	var promptProperties PromptProperties
	err = json.Unmarshal(propertiesJSON, &promptProperties)
	if err != nil {
		return PromptProperties{}, err
	}

	return promptProperties, nil

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

	promptProperties, err := GetProperties(id)
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
		return 0, err
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

func RetrieveResponse(code string) (string, error) {

	client, err := loadClient()
	if err != nil {
		return "", err
	}

	fields := []graphql.Field{
		{Name: "instruct"},
		{Name: "rank"},
		{Name: "hasResponse", Fields: []graphql.Field{
			{Name: "... on Response", Fields: []graphql.Field{
				{Name: "response"},
			}},
		}},
	}

	where := filters.Where().
		WithPath([]string{"code"}).
		WithOperator(filters.Like).
		WithValueText(code)

	byRankDesc := graphql.Sort{
		Path: []string{"rank"}, Order: graphql.Desc,
	}

	ctx := context.Background()
	result, err := client.GraphQL().Get().
		WithClassName("Prompt").
		WithSort(byRankDesc).
		WithFields(fields...).
		WithWhere(where).
		Do(ctx)
	if err != nil {
		panic(err)
	}

	log.Printf("result= %v\n", result)

	getPrompt, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return "", errors.New("unexpected response format: 'Get' field not found or not a map")
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok || len(promptData) == 0 {
		return "", errors.New("unexpected response format: 'Prompt' field not found or empty list")
	}

	// Initialize variables to track the prompt with the highest rank
	var highestRank int
	var highestRankPrompts []map[string]interface{}

	// Iterate through each prompt to find the one with the highest rank
	for _, prompt := range promptData {
		promptMap, ok := prompt.(map[string]interface{})
		if !ok {
			return "", errors.New("unexpected response format: prompt data is not a map")
		}

		rankInterface, ok := promptMap["rank"]
		if !ok {
			return "", errors.New("rank field not found in prompt data")
		}

		rank, ok := rankInterface.(float64)
		if !ok {
			return "", errors.New("rank field is not a number")
		}

		// Convert float64 to int
		rankInt := int(rank)

		if rankInt > highestRank {
			highestRank = rankInt
			highestRankPrompts = []map[string]interface{}{promptMap}
		} else if rankInt == highestRank {
			highestRankPrompts = append(highestRankPrompts, promptMap)
		}
	}

	// If there are prompts with the same highest rank, select one randomly
	if len(highestRankPrompts) > 0 {
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(highestRankPrompts))
		selectedPrompt := highestRankPrompts[randomIndex]

		hasResponse, ok := selectedPrompt["hasResponse"].([]interface{})
		if !ok || len(hasResponse) == 0 {
			return "", errors.New("hasResponse field not found in prompt data or empty list")
		}

		firstResponseMap, ok := hasResponse[0].(map[string]interface{})
		if !ok {
			return "", errors.New("unexpected response format: response data is not a map")
		}

		response, ok := firstResponseMap["response"].(string)
		if !ok {
			return "", errors.New("response field not found in response data or not a string")
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			fmt.Println("Error:", err)
			return "", err
		}

		log.Printf("Selected Response: %v\n", response)

		// Add a newline character to the end of the string
		jsonDataWithNewline := append(jsonData, '\n')

		return string(jsonDataWithNewline), nil
	}

	return "", errors.New("no prompt found")

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
