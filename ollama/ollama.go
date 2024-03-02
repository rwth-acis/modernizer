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
	if !ok {
		set = "default"
	}

	instruct, ok := prompt["instruct"].(string)

	log.Printf("ok: %v", ok)
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
		model = "codellama:13b-instruct"
	}

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": completePrompt,
		"stream": false,
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

	PromptID, err := weaviate.CreatePromptObject(instruct, code, "Prompt")
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	log.Printf("PromptID: %s\n", PromptID)

	ResponseID, err := weaviate.CreateResponseObject(response, "Response")
	if err != nil {
		return weaviate.ResponseData{}, err
	}

	err = weaviate.CreateReferences(PromptID, ResponseID)
	if err != nil {
		panic(err)
	}

	responseData := weaviate.ResponseData{
		Response: response,
		PromptID: PromptID,
		Instruct: instruct,
	}

	return responseData, nil
}
