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
)

func main() {
	router := gin.Default()

	// router.GET("/", getInfo)

	// router.Use(LogRequestBodyMiddleware)

	router.Any("/*proxyPath", proxy)

	err := router.Run(":8080")
	if err != nil {
		return
	}
}

func LogRequestBodyMiddleware(c *gin.Context) {

	body, err := c.GetRawData()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error reading request body"})
		return
	}

	// Log the request body
	log.Printf("Request Body: %s\n", body)

	// Rewind the request body so that subsequent middleware/handlers can read it
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	c.Next()
}

func getInfo(c *gin.Context) {

	resp, err := http.Get(os.Getenv("OLLAMA_URL"))

	if err != nil {
		//
	}

	// Set the content type to text/plain
	c.Header("Content-Type", "text/plain")

	// Copy the HTML data directly to the response
	_, err = io.Copy(c.Writer, resp.Body)

	defer resp.Body.Close()

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
