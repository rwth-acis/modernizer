package ollama

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/rwth-acis/modernizer/redis"
	"github.com/rwth-acis/modernizer/weaviate"
	"io"
	"log"
	"net/http"
	"os"
)

func GenerateResponse(prompt map[string]interface{}) (weaviate.ResponseData, error) {
	url := os.Getenv("OLLAMA_URL") + "/api/generate"

	set, ok := prompt["instructType"].(string)

	log.Printf("instruction type: %s\n", set)

	if !ok {
		set = "default"
	}

	gitURL, ok := prompt["gitURL"].(string)

	log.Printf("gitURL: %s\n", gitURL)

	if !ok {
		gitURL = ""
	}

	instruct, ok := prompt["instruct"].(string)
	if !ok {
		instruct, _ = redis.GetSetMember(set)
	}

	log.Printf("Prompt: %s\n", instruct)

	code, ok := prompt["prompt"].(string)
	if !ok {
		return weaviate.ResponseData{}, errors.New("prompt field is not a string")
	}

	log.Printf("Code: %s\n", code)

	completePrompt := instruct + " " + code

	model, ok := prompt["model"].(string)
	if !ok {
		model = "codellama:34b-instruct"
	}

	var context int
	if len(completePrompt) < 2048 {
		context = 2048
	} else {
		context = len(completePrompt) * 2
	}

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": completePrompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_ctx": context,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return weaviate.ResponseData{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	var responseJSON map[string]interface{}
	err = json.Unmarshal(body, &responseJSON)
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	response, ok := responseJSON["response"].(string)
	log.Printf("Reponse: %s\n", response)

	if !ok {
		log.Println("Error: 'response' field is not a string array")
		return weaviate.ResponseData{}, errors.New("invalid response format")
	}

	PromptID, err := weaviate.CreatePromptObject(instruct, code, "Prompt", gitURL)
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	log.Printf("PromptID: %s\n", PromptID)

	ResponseID, err := weaviate.CreateResponseObject(response, "Response")
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	err = weaviate.CreateResponseReferences(PromptID, ResponseID)
	if err != nil {
		panic(err)
	}

	responseData := weaviate.ResponseData{
		Response: response,
		PromptID: PromptID,
		Instruct: instruct,
		GitURL:   gitURL,
	}

	go SemanticMeaning(PromptID, code, true)

	return responseData, nil
}

func SemanticMeaning(promptID string, code string, generateReference bool) string {
	url := os.Getenv("OLLAMA_URL") + "/api/generate"

	requestBody := map[string]interface{}{
		"model":  "semantic-meaning",
		"prompt": "What is the semantic meaning of this code: " + code,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("could not marshal request body: %s\n", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("could not create request: %s\n", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("could not send request: %s\n", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("could not read body")
	}

	var responseJSON map[string]interface{}
	err = json.Unmarshal(body, &responseJSON)
	if err != nil {
		log.Printf("could not unmarshal response: %s\n", err)
	}

	response, ok := responseJSON["response"].(string)
	log.Printf("Reponse of semantic Meaning: %s\n", response)
	if !ok {
		log.Println("Error: 'response' field is not a string array")
	}

	if !generateReference {
		return response
	} else {
		semanticMeaningID, err := weaviate.CreateSemanticMeaningObject(response)
		if err != nil {
			log.Printf("creating semantic meaning object failed: %s\n", err)
		}

		err = weaviate.CreateReferencePromptToSemanticMeaning(promptID, semanticMeaningID)
		if err != nil {
			log.Printf("error creating semantic meaning reference: %s\n", err)
		}

		err = weaviate.CreateReferenceSemanticMeaningToPrompt(semanticMeaningID, promptID)
		if err != nil {
			log.Printf("error creating semantic meaning reference: %s\n", err)
		}
	}

	return ""
}
