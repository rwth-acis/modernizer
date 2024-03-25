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
		"model":  "codellama:13b-instruct",
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

	PromptID, err := weaviate.CreatePromptObject(instruct, set, code, "Prompt", gitURL)
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
	url := os.Getenv("OLLAMA_URL") + "/api/chat"

	requestBody := map[string]interface{}{
		"model":  "semantic-meaning",
		"stream": false,
		"messages": []map[string]string{
			{"role": "user", "content": "def add(a, b):\n    return a + b"},
			{"role": "assistant", "content": "add two numbers"},
			{"role": "user", "content": "public class ArithmeticFunctions {\n    public static double divide(double a, double b) {\n        if (b == 0) {\n            throw new ArithmeticException(\"Cannot divide by zero\");\n        }\n        return a / b;\n    }\n}"},
			{"role": "assistant", "content": "multiply two numbers"},
			{"role": "user", "content": "#include <stdio.h>\n\nint fibonacci(int n) {\n    if (n <= 1)\n        return n;\n    else\n        return fibonacci(n - 1) + fibonacci(n - 2);\n}\n\nint main() {\n    int n, i;\n\n    printf(\"Enter the number of terms: \");\n    scanf(\"%d\", &n);\n\n    printf(\"Fibonacci Series: \");\n    for (i = 0; i < n; i++) {\n        printf(\"%d \", fibonacci(i));\n    }\n\n    return 0;\n}\n"},
			{"role": "assistant", "content": "calculate the fibonacci sequence"},
			{"role": "user", "content": "func proxy(c *gin.Context) {\n\tremote, err := url.Parse(os.Getenv(\"OLLAMA_URL\"))\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tproxy := httputil.NewSingleHostReverseProxy(remote)\n\tproxy.Director = func(req *http.Request) {\n\t\treq.Header = c.Request.Header\n\t\treq.Host = remote.Host\n\t\treq.URL.Scheme = remote.Scheme\n\t\treq.URL.Host = remote.Host\n\t\treq.URL.Path = c.Param(\"proxyPath\")\n\t}\n\n\tproxy.ServeHTTP(c.Writer, c.Request)\n}"},
			{"role": "assistant", "content": "reverse proxy"},
			{"role": "user", "content": "func CreatePromptObject(instruct string, code string, class string, gitURL string) (string, error) {\n\tclient, err := loadClient()\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\n\tdataSchema := map[string]interface{}{\n\t\t\"instruct\": instruct,\n\t\t\"code\":     code,\n\t\t\"rank\":     1,\n\t\t\"gitURL\":   gitURL,\n\t}\n\n\tweaviateObject, err := client.Data().Creator().\n\t\tWithClassName(class).\n\t\tWithProperties(dataSchema).\n\t\tDo(context.Background())\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\n\treturn string(weaviateObject.Object.ID), nil\n}"},
			{"role": "assistant", "content": "create a weaviate object"},
			{"role": "user", "content": code},
		},
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

	message, ok := responseJSON["message"].(map[string]interface{})
	if !ok {
		log.Println("Error: 'message' field is not a map")
		return ""
	}

	content, ok := message["content"].(string)
	if !ok {
		log.Println("Error: 'content' field is not a string")
		return ""
	}

	log.Printf("Content: %s\n", content)

	if !generateReference {
		return content
	} else {
		semanticMeaningID, err := weaviate.CreateSemanticMeaningObject(content)
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
