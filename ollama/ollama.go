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

func CreateEmbedding(prompt string) ([]float32, error) {
	url := os.Getenv("OLLAMA_URL") + "/api/embeddings"

	data := map[string]interface{}{
		"model":  os.Getenv("OLLAMA_MODEL"),
		"prompt": prompt,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return nil, err
	}

	// Extract the "embedding" object as a float array
	embeddingArray, ok := response["embedding"].([]interface{})
	if !ok {
		log.Println("Error: 'embedding' field is not a float array")
		return nil, err
	}

	// Convert interface array to float32 array
	var embeddingVector []float32
	for _, v := range embeddingArray {
		if floatValue, ok := v.(float64); ok {
			embeddingVector = append(embeddingVector, float32(floatValue))
		} else {
			log.Println("Error: Unable to convert value to float32")
			return nil, err
		}
	}

	return nil, nil
}

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
