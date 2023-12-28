package main

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var OllamaURL = "http://ollama-service.ba-kovacevic:11434"

func main() {
	router := gin.Default()
	router.GET("/", getInfo)

	router.Any("/api/*proxyPath", proxy)

	err := router.Run(":8080")
	if err != nil {
		return
	}
}

func getInfo(c *gin.Context) {

	resp, err := http.Get(OllamaURL)

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
	remote, err := url.Parse(OllamaURL)
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
