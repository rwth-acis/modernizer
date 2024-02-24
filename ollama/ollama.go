package ollama

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rwth-acis/modernizer/redis"
	"github.com/rwth-acis/modernizer/weaviate"
	"io"
	"log"
	"net/http"
	"os"
)

func GenerateResponse(prompt map[string]interface{}) (string, error) {
	url := os.Getenv("OLLAMA_URL") + "/api/generate"

	// TODO add possibility to differentiate between system prompt roles/creativity
	// TODO add routes to show and add prompts

	instruct, err := redis.GetSetMember("default")
	if err != nil {
		return "", err
	}

	log.Printf("Prompt: %s\n", instruct)

	code, ok := prompt["prompt"].(string)
	if !ok {
		return "", errors.New("prompt field is not a string")
	}

	log.Printf("Code: %s\n", code)

	completePrompt := instruct + " " + code

	requestBody := map[string]interface{}{
		"model":  prompt["model"],
		"prompt": completePrompt,
		"stream": prompt["stream"],
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var responseJSON map[string]interface{}
	err = json.Unmarshal(body, &responseJSON)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return "", err
	}

	response, ok := responseJSON["response"].(string)
	if !ok {
		log.Println("Error: 'response' field is not a string array")
		return "", errors.New("invalid response format")
	}

	log.Printf("Reponse: %s\n", response)

	PromptID, err := weaviate.CreatePromptObject(instruct, code, "Prompt")
	if err != nil {
		return "", err
	}

	ResponseID, err := weaviate.CreateResponseObject(response, "Response")
	if err != nil {
		return "", err
	}

	err = weaviate.CreateReferences(PromptID, ResponseID)
	if err != nil {
		panic(err)
	}

	return response, nil
}
