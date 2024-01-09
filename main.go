package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rwth-acis/modernizer/weaviate"
)

func main() {

	// create Weaviate schema
	err := weaviate.CreateSchema("Prompt", "Content of the users prompt")
	if err != nil {
		panic(err)
	}
	err = weaviate.CreateSchema("Response", "Content of the LLMs response")
	if err != nil {
		panic(err)
	}

	router := gin.Default()

	router.GET("/weaviate/schema", func(c *gin.Context) {
		schema, err := weaviate.RetrieveSchema()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(http.StatusOK, "application/json; charset=utf-8", schema)
	})

	router.Any("/ollama/*proxyPath", proxy)

	err = router.Run(":8080")
	if err != nil {
		return
	}
}

func proxy(c *gin.Context) {
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

	// Log the request and response bodies
	log.Printf("Request Body: %s\n", string(requestBody))
	log.Printf("Response Body: %s\n", responseWriter.BodyInterceptor.Bytes())
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
