package main

import (
	"bytes"
	"io"
	"io/ioutil"
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

	router.Use(LogRequestBodyMiddleware)

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
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
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
