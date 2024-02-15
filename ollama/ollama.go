package ollama

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rwth-acis/modernizer/weaviate"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
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

	//TODO add possibility to differentiate between system prompt roles/creativity
	//TODO add routes to show and add prompts

	var promptDB = []string{
		"Explain me this:",
		"How does the following code work?",
		"Explain me like I am five what this code does?",
		"You are a senior developer and are responsible for providing thoughtful and concise documentation. Write documentation for the following code:",
		"Explain me what this piece of code does like angry Linux Torvalds on Linux kernel code reviews:",
		"Can you break down the underlying concepts behind this code snippet?",
		"Describe the architectural decisions made in this code.",
		"How does this code align with best practices in software engineering?",
		"Discuss potential optimizations and performance improvements for this code.",
		"What security considerations should be taken into account when using this code?",
		"Consider how this code could be adapted for different use cases.",
		"Examine the impact of different input data on the code's behavior.",
		"Evaluate the scalability of this code for large-scale applications.",
		"How many WTFs per minute does this code generate?",
		"Explain this code as if you were a wizard casting a spell.",
		"Pretend you're a detective solving a mystery related to this code.",
		"Imagine this code as a recipe for a bizarre culinary dish.",
		"Describe this code using only emojis and internet slang.",
		"Interpret this code through the lens of a conspiracy theorist uncovering hidden messages.",
		"As the responsible senior engineer, what technical debt does this code have?",
	}

	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	randomIndex := rng.Intn(len(promptDB))

	chosenPrompt := promptDB[randomIndex]
	log.Printf("Prompt: %s\n", chosenPrompt)

	code, ok := prompt["prompt"].(string)
	if !ok {
		return "", errors.New("prompt field is not a string")
	}

	log.Printf("Code: %s\n", code)

	// Prepend the random sentence to the code
	completePrompt := chosenPrompt + " " + code

	// Create the JSON request body
	requestBody := map[string]interface{}{
		"model":  prompt["model"],
		"prompt": completePrompt,
		"stream": prompt["stream"],
	}

	// Convert the request body to JSON
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

	vector, err := CreateEmbedding(code)
	if err != nil {
		return "", err
	}

	PromptID, err := weaviate.CreatePromptObject(vector, chosenPrompt, code, "Prompt")
	if err != nil {
		return "", err
	}

	vector, err = CreateEmbedding(response)
	if err != nil {
		return "", err
	}

	ResponseID, err := weaviate.CreateResponseObject(vector, response, "Response")
	if err != nil {
		return "", err
	}

	err = weaviate.CreateReferences(PromptID, ResponseID)
	if err != nil {
		panic(err)
	}

	return response, nil
}
