package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"math/rand"
	"time"
)

type PromptProperties struct {
	Code        string                   `json:"code"`
	HasResponse []map[string]interface{} `json:"hasResponse"`
	Instruct    string                   `json:"instruct"`
	Rank        int                      `json:"rank"`
}

type ResponseData struct {
	ID       string `json:"id"`
	Response string `json:"response"`
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

func RetrieveBestResponse(code string) (ResponseData, error) {

	responses, err := RetrieveResponsesRankDesc(code)
	if err != nil {
		return ResponseData{}, err
	}

	getPrompt, ok := responses.Data["Get"].(map[string]interface{})
	if !ok {
		return ResponseData{}, errors.New("unexpected response format: 'Get' field not found or not a map")
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok || len(promptData) == 0 {
		return ResponseData{}, errors.New("unexpected response format: 'Prompt' field not found or empty list")
	}

	var highestRank int
	var highestRankPrompts []map[string]interface{}

	for _, prompt := range promptData {
		promptMap, ok := prompt.(map[string]interface{})
		if !ok {
			return ResponseData{}, errors.New("unexpected response format: prompt data is not a map")
		}

		rankInterface, ok := promptMap["rank"]
		if !ok {
			return ResponseData{}, errors.New("rank field not found in prompt data")
		}

		rank, ok := rankInterface.(float64)
		if !ok {
			return ResponseData{}, errors.New("rank field is not a number")
		}

		rankInt := int(rank)

		if rankInt > highestRank {
			highestRank = rankInt
			highestRankPrompts = []map[string]interface{}{promptMap}
		} else if rankInt == highestRank {
			highestRankPrompts = append(highestRankPrompts, promptMap)
		}
	}

	if len(highestRankPrompts) > 0 {

		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)
		randomIndex := rng.Intn(len(highestRankPrompts))
		selectedPrompt := highestRankPrompts[randomIndex]

		response, err := ExtractResponse(selectedPrompt)
		if err != nil {
			return ResponseData{}, err
		}

		id, err := ExtractID(selectedPrompt)
		if err != nil {
			return ResponseData{}, err
		}

		responseData := ResponseData{
			ID:       id,
			Response: response,
		}

		return responseData, nil
	}

	return ResponseData{}, errors.New("no prompt found")

}

func RetrieveRandomResponse(code string) (ResponseData, error) {

	responses, err := RetrieveResponsesRankDesc(code)
	if err != nil {
		return ResponseData{}, err
	}

	getPrompt, ok := responses.Data["Get"].(map[string]interface{})
	if !ok {
		return ResponseData{}, errors.New("unexpected response format: 'Get' field not found or not a map")
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok || len(promptData) == 0 {
		return ResponseData{}, errors.New("unexpected response format: 'Prompt' field not found or empty list")
	}

	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	randomIndex := rng.Intn(len(promptData))
	selectedPrompt := promptData[randomIndex]

	selectedPromptMap, ok := selectedPrompt.(map[string]interface{})
	if !ok {
		return ResponseData{}, errors.New("unexpected response format: selected prompt data is not a map")
	}

	response, err := ExtractResponse(selectedPromptMap)
	if err != nil {
		return ResponseData{}, err
	}

	id, err := ExtractID(selectedPromptMap)
	if err != nil {
		return ResponseData{}, err
	}

	responseData := ResponseData{
		ID:       id,
		Response: response,
	}

	return responseData, nil
}

func RetrieveResponsesRankDesc(code string) (*models.GraphQLResponse, error) {

	client, err := loadClient()
	if err != nil {
		return nil, err
	}

	fields := []graphql.Field{
		{Name: "hasResponse", Fields: []graphql.Field{
			{Name: "... on Response", Fields: []graphql.Field{
				{Name: "response"},
			}},
		}},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
		}},
	}

	where := filters.Where().
		WithPath([]string{"code"}).
		WithOperator(filters.Like).
		WithValueText(code)

	ctx := context.Background()
	result, err := client.GraphQL().Get().
		WithClassName("Prompt").
		WithFields(fields...).
		WithWhere(where).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ExtractID(selectedPrompt map[string]interface{}) (string, error) {
	hasAdditionalInterface, ok := selectedPrompt["_additional"]
	if !ok {
		return "", errors.New("_additional field not found in prompt data")
	}

	additionalMap, ok := hasAdditionalInterface.(map[string]interface{})
	if !ok {
		return "", errors.New("_additional field is not a map in prompt data")
	}

	idInterface, ok := additionalMap["id"]
	if !ok {
		return "", errors.New("id field not found in _additional data")
	}

	id, ok := idInterface.(string)
	if !ok {
		return "", errors.New("id field is not a string in _additional data")
	}

	return id, nil
}

func ExtractResponse(selectedPromptMap map[string]interface{}) (string, error) {
	hasResponse, ok := selectedPromptMap["hasResponse"].([]interface{})
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

	return response, nil
}
