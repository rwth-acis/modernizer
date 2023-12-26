package main

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

func main() {
	router := gin.Default()
	router.GET("/", getInfo)

	router.Run(":8080")
}

func getInfo(c *gin.Context) {

	resp, err := http.Get("http://ollama-service.ba-kovacevic:11434")

	if err != nil {
		//
	}

	// Set the content type to text/plain
	c.Header("Content-Type", "text/plain")

	// Copy the HTML data directly to the response
	_, err = io.Copy(c.Writer, resp.Body)

	defer resp.Body.Close()

}
