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

type ResponseData struct {
	Response string `json:"response"`
	PromptID string `json:"promptID"`
	Instruct string `json:"instruct"`
}

func GenerateResponse(prompt map[string]interface{}) (ResponseData, error) {
	url := os.Getenv("OLLAMA_URL") + "/api/generate"

	set, ok := prompt["instructType"].(string)
	if !ok {
		set = "default"
	}

	instruct, err := redis.GetSetMember(set)
	if err != nil {
		return ResponseData{}, errors.New("no data available")
	}

	log.Printf("Prompt: %s\n", instruct)

	code, ok := prompt["prompt"].(string)
	if !ok {
		return ResponseData{}, errors.New("prompt field is not a string")
	}

	log.Printf("Code: %s\n", code)

	completePrompt := instruct + " " + code

	requestBody := map[string]interface{}{
		"model":  prompt["model"],
		"prompt": completePrompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return ResponseData{}, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResponseData{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ResponseData{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResponseData{}, err
	}

	var responseJSON map[string]interface{}
	err = json.Unmarshal(body, &responseJSON)
	if err != nil {
		return ResponseData{}, err
	}

	response, ok := responseJSON["response"].(string)
	if !ok {
		log.Println("Error: 'response' field is not a string array")
		return ResponseData{}, errors.New("invalid response format")
	}

	log.Printf("Reponse: %s\n", response)

	PromptID, err := weaviate.CreatePromptObject(instruct, code, "Prompt")
	if err != nil {
		return ResponseData{}, err
	}

	ResponseID, err := weaviate.CreateResponseObject(response, "Response")
	if err != nil {
		return ResponseData{}, err
	}

	err = weaviate.CreateReferences(PromptID, ResponseID)
	if err != nil {
		panic(err)
	}

	responseData := ResponseData{
		Response: response,
		PromptID: PromptID,
		Instruct: instruct,
	}

	return responseData, nil
}
