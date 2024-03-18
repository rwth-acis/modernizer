package weaviate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

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

	if len(objects) == 0 {
		return PromptProperties{}, fmt.Errorf("no object found with ID: %s", id)
	}

	propertiesJSON, err := json.Marshal(objects[0].Properties)
	if err != nil {
		return PromptProperties{}, err
	}

	var temp struct {
		Code        string                   `json:"code"`
		HasResponse []map[string]interface{} `json:"hasResponse"`
		Instruct    string                   `json:"instruct"`
		Rank        int                      `json:"rank"`
		GitURL      string                   `json:"gitURL"`
	}

	if err := json.Unmarshal(propertiesJSON, &temp); err != nil {
		return PromptProperties{}, err
	}

	var responseText string
	if len(temp.HasResponse) > 0 {
		responseTextBytes, err := json.Marshal(temp.HasResponse)
		if err != nil {
			return PromptProperties{}, err
		}
		responseText = string(responseTextBytes)
	}

	response, err := RetrieveResponseByID(id)
	if err != nil {
		return PromptProperties{}, err
	}
	responseText, ok := response.(string)
	if !ok {
		return PromptProperties{}, fmt.Errorf("response from RetrieveResponseByID is not a string")
	}

	promptProperties := PromptProperties{
		Code:        temp.Code,
		HasResponse: responseText,
		Instruct:    temp.Instruct,
		Rank:        temp.Rank,
		GitURL:      temp.GitURL,
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

func RetrieveResponseByID(id string) (interface{}, error) {
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
		{Name: "rank"},
		{Name: "instruct"},
	}

	withNearObject := &graphql.NearObjectArgumentBuilder{}

	withNearObject.WithID(id)

	ctx := context.Background()
	result, err := client.GraphQL().Get().
		WithClassName("Prompt").
		WithFields(fields...).
		WithLimit(1).
		WithNearObject(withNearObject).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	response, err := ExtractResponseFromGraphQL(result)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func ResponseList(code string) ([]string, error) {
	responses, err := RetrieveResponsesRankDesc(code)
	if err != nil {
		return nil, err
	}

	getPrompt, ok := responses.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected response format: 'Get' field not found or not a map")
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok {
		return nil, errors.New("unexpected response format: 'Prompt' field not found")
	}

	if len(promptData) == 0 {
		return nil, errors.New("no prompt found")
	}

	var RankIDs []string

	for _, prompt := range promptData {
		promptMap, ok := prompt.(map[string]interface{})
		if !ok {
			return nil, errors.New("unexpected response format: prompt data is not a map")
		}

		id, err := ExtractID(promptMap)
		if err != nil {
			return nil, err
		}

		RankIDs = append(RankIDs, id)
	}

	return RankIDs, nil
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
		log.Printf("%v", rankInterface)
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

		instruct, err := ExtractInstruct(selectedPrompt)
		if err != nil {
			return ResponseData{}, err
		}

		responseData := ResponseData{
			PromptID: id,
			Response: response,
			Instruct: instruct,
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

	instruct, err := ExtractInstruct(selectedPromptMap)
	if err != nil {
		return ResponseData{}, err
	}

	responseData := ResponseData{
		PromptID: id,
		Response: response,
		Instruct: instruct,
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
		{Name: "rank"},
		{Name: "instruct"},
	}

	where := filters.Where().
		WithPath([]string{"code"}).
		WithOperator(filters.Like).
		WithValueText(code)

	rankDesc := graphql.Sort{
		Path: []string{"rank"}, Order: graphql.Desc,
	}

	ctx := context.Background()
	result, err := client.GraphQL().Get().
		WithClassName("Prompt").
		WithFields(fields...).
		WithWhere(where).
		WithSort(rankDesc).
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

func ExtractInstruct(selectedPrompt map[string]interface{}) (string, error) {
	hasInstruct, ok := selectedPrompt["instruct"]
	if !ok {
		return "", errors.New("instruct field not found in prompt data")
	}

	instruct, ok := hasInstruct.(string)
	if !ok {
		return "", errors.New("id field is not a string in _additional data")
	}

	return instruct, nil
}

func ExtractResponseFromGraphQL(query *models.GraphQLResponse) (string, error) {
	getPrompt, ok := query.Data["Get"].(map[string]interface{})
	if !ok {
		return "", errors.New("unexpected response format: 'Get' field not found or not a map")
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok || len(promptData) == 0 {
		return "", errors.New("unexpected response format: 'Prompt' field not found or empty list")
	}
	selectedPrompt := promptData[0].(map[string]interface{})
	if !ok {
		return "", errors.New("unexpected response format: selected prompt data is not a map")
	}

	response, err := ExtractResponse(selectedPrompt)
	if err != nil {
		return "", err
	}
	return response, nil

}

func RetrieveHasSemanticMeaning(code string) (string, bool) {
	client, err := loadClient()
	if err != nil {
		log.Printf("Error loading client: %v", err)
	}

	fields := []graphql.Field{
		{Name: "hasSemanticMeaning", Fields: []graphql.Field{
			{Name: "... on SemanticMeaning", Fields: []graphql.Field{
				{Name: "semanticMeaning"},
			}},
		}},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
		}},
	}

	where := filters.Where().
		WithPath([]string{"code"}).
		WithOperator(filters.Equal).
		WithValueText(code)

	ctx := context.Background()
	result, err := client.GraphQL().Get().
		WithClassName("Prompt").
		WithFields(fields...).
		WithLimit(1).
		WithWhere(where).
		Do(ctx)
	if err != nil {
		log.Printf("error: %v", err)
	}

	getPrompt, ok := result.Data["Get"].(map[string]interface{})
	if !ok || len(getPrompt) == 0 {
		return "", false
	}

	promptData, ok := getPrompt["Prompt"].([]interface{})
	if !ok || len(promptData) == 0 {
		return "", false
	}

	selectedPrompt := promptData[0].(map[string]interface{})

	hasSemanticMeaning, _ := selectedPrompt["hasSemanticMeaning"].([]interface{})

	firstSemanticMeaningMap, _ := hasSemanticMeaning[0].(map[string]interface{})

	semanticMeaning, _ := firstSemanticMeaningMap["semanticMeaning"].(string)

	return semanticMeaning, true
}

func GetSimilarSemanticMeaning(code string) ([]string, error) {
	client, err := loadClient()
	if err != nil {
		log.Printf("Error loading client: %v", err)
	}

	fields := []graphql.Field{
		{Name: "hasPrompt", Fields: []graphql.Field{
			{Name: "... on Prompt", Fields: []graphql.Field{
				{Name: "_additional", Fields: []graphql.Field{
					{Name: "id"},
				}},
				{Name: "gitURL"},
			}},
		}},
	}

	withNearText := client.GraphQL().NearTextArgBuilder().
		WithConcepts([]string{code}).
		WithCertainty(0.8)

	result, err := client.GraphQL().Get().
		WithClassName("SemanticMeaning").
		WithFields(fields...).
		WithNearText(withNearText).
		Do(context.Background())

	getMap, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected response format: 'Get' field not found or not a map")
	}

	SemanticMeaningData, ok := getMap["SemanticMeaning"].([]interface{})
	if !ok || len(SemanticMeaningData) == 0 {
		return nil, errors.New("unexpected response format: 'Prompt' field not found or empty list")
	}

	var gitURLs []string

	for _, prompt := range SemanticMeaningData {
		Map, ok := prompt.(map[string]interface{})
		if !ok {
			return nil, errors.New("unexpected response format: prompt data is not a map")
		}

		hasPrompt, ok := Map["hasPrompt"].([]interface{})
		if !ok {
			return nil, errors.New("hasPrompt field not found in prompt data")
		}

		hasPromptMap, ok := hasPrompt[0].(map[string]interface{})
		if !ok {
			return nil, errors.New("hasPrompt field not found in prompt data")
		}

		gitURL, ok := hasPromptMap["gitURL"]
		if !ok {
			return nil, errors.New("rank field not found in prompt data")
		}

		gitURLs = append(gitURLs, gitURL.(string))
	}

	return gitURLs, nil
}
