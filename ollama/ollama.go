package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	// Print the response body
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

	return embeddingVector, nil
}
