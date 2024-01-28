package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// RequestBody represents the structure of the JSON request body
type RequestBody struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

func proxyLog(c *gin.Context) {
	remote, err := url.Parse(os.Getenv("OLLAMA_URL"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing remote URL"})
		return
	}

	// Save the original request body
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
	}

	var reqBody RequestBody
	if err := json.Unmarshal(requestBody, &reqBody); err != nil {
		log.Printf("Error decoding JSON request body: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = c.Param("proxyPath")
	}

	// Create a custom response writer to intercept the response
	responseWriter := &responseWriterInterceptor{
		ResponseWriter:  c.Writer,
		BodyInterceptor: &bytes.Buffer{},
	}

	// Replace the response writer with the custom one
	c.Writer = responseWriter

	// Restore the original request body
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	// ServeHTTP on the proxy
	proxy.ServeHTTP(responseWriter, c.Request)

	var body = responseWriter.BodyInterceptor.Bytes()

	// ResponseBody represents the structure of the JSON response body
	type ResponseBody struct {
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
		Response  string `json:"response"`
		Done      bool   `json:"done"`
		Context   []int  `json:"context"`
	}

	// Unmarshal the response body into the ResponseBody struct
	var respBody ResponseBody
	if err := json.Unmarshal(body, &respBody); err != nil {
		log.Printf("Error decoding JSON response body: %v", err)
		return
	}

	// Log the request and response bodies
	log.Printf("Request Body: %s\n", reqBody.Prompt)
	log.Printf("Response Body: %s\n", respBody.Response)

	//err = weaviate.CreateObject(reqBody.Prompt, "Prompt")
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = weaviate.CreateObject(respBody.Response, "Response")
	//if err != nil {
	//	panic(err)
	//}
}

// responseWriterInterceptor is a custom ResponseWriter to intercept the response body
type responseWriterInterceptor struct {
	gin.ResponseWriter
	BodyInterceptor *bytes.Buffer
}

// Write method intercepts the response body
func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	w.BodyInterceptor.Write(b)
	return w.ResponseWriter.Write(b)
}

func proxy(c *gin.Context) {
	remote, err := url.Parse(os.Getenv("OLLAMA_URL"))
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = c.Param("proxyPath")
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
