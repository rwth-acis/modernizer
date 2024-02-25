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
	"time"
)

type PromptProperties struct {
	Code        string                   `json:"code"`
	HasResponse []map[string]interface{} `json:"hasResponse"`
	Instruct    string                   `json:"instruct"`
	Rank        int                      `json:"rank"`
}

func RetrieveProperties(id string) (PromptProperties, error) {

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
