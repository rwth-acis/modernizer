package weaviate

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate/entities/models"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type ResponseData struct {
	Response string `json:"response"`
	PromptID string `json:"promptID"`
	Instruct string `json:"instruct"`
	GitURL   string `json:"gitURL"`
}

type PromptProperties struct {
	Code        string `json:"code"`
	HasResponse string `json:"hasResponse"`
	Instruct    string `json:"instruct"`
	Rank        int    `json:"rank"`
	GitURL      string `json:"gitURL"`
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

	exists, err = client.Schema().ClassExistenceChecker().WithClassName("SemanticMeaning").Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		classObj := &models.Class{
			Class:       "SemanticMeaning",
			Description: "This class contains the semantic Meaning of the code",
			Vectorizer:  "text2vec-transformers",
			ModuleConfig: map[string]interface{}{
				"text2vec-transformers": map[string]interface{}{},
			},
			Properties: []*models.Property{
				{
					DataType:    []string{"text"},
					Description: "The generated response by the LLM",
					Name:        "semanticMeaning",
				},
			},
		}
		err = client.Schema().ClassCreator().WithClass(classObj).Do(context.Background())
		if err != nil {
			return err
		}
		log.Println("created semanticMeaning class")
	} else {
		log.Println("semanticMeaning class already exists")
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
					DataType: []string{"SemanticMeaning"},
					Name:     "hasSemanticMeaning",
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
					ModuleConfig: map[string]interface{}{
						"text2vec-transformers": map[string]interface{}{
							"skip": true,
						},
					},
				},
				{
					DataType:    []string{"text"},
					Description: "A link to the git blob containing the code with the line and character position of the prompt",
					Name:        "gitURL",
					ModuleConfig: map[string]interface{}{
						"text2vec-transformers": map[string]interface{}{
							"skip": true,
						},
					},
				},
			},
		}

		err = client.Schema().ClassCreator().WithClass(classObj).Do(context.Background())
		if err != nil {
			return err
		}
		log.Println("created Prompt class")

		prop := &models.Property{
			DataType: []string{"Prompt"},
			Name:     "hasPrompt",
			ModuleConfig: map[string]interface{}{
				"text2vec-transformers": map[string]interface{}{
					"skip": true,
				},
			},
		}
		err = client.Schema().PropertyCreator().WithClassName("SemanticMeaning").WithProperty(prop).Do(context.Background())
		if err != nil {

			log.Printf("Error creating property: %v\n", err)
			return err
		}
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

func CreatePromptObject(instruct string, code string, class string, gitURL string) (string, error) {
	client, err := loadClient()
	if err != nil {
		return "", err
	}

	dataSchema := map[string]interface{}{
		"instruct": instruct,
		"code":     code,
		"rank":     1,
		"gitURL":   gitURL,
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

func CreateSemanticMeaningObject(meaning string) (string, error) {
	client, err := loadClient()
	if err != nil {
		return "", err
	}

	dataSchema := map[string]interface{}{
		"semanticMeaning": meaning,
	}

	weaviateObject, err := client.Data().Creator().
		WithClassName("semanticMeaning").
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

func CreateResponseReferences(PromptID string, ResponseID string) error {
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

func CreateReferencePromptToSemanticMeaning(PromptID string, semanticMeaningID string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	err = client.Data().ReferenceReplacer().
		WithClassName("Prompt").
		WithID(PromptID).
		WithReferenceProperty("hasSemanticMeaning").
		WithReferences(&models.MultipleRef{
			client.Data().ReferencePayloadBuilder().
				WithClassName("SemanticMeaning").
				WithID(semanticMeaningID).
				Payload(),
		}).
		Do(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func CreateReferenceSemanticMeaningToPrompt(semanticMeaningID string, PromptID string) error {
	client, err := loadClient()
	if err != nil {
		return err
	}

	err = client.Data().ReferenceCreator().
		WithClassName("SemanticMeaning").
		WithID(semanticMeaningID).
		WithReferenceProperty("hasPrompt").
		WithReference(client.Data().ReferencePayloadBuilder().
			WithClassName("Prompt").
			WithID(PromptID).
			Payload()).
		Do(context.Background())
	if err != nil {
		return err
	}

	return nil
}
