package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rwth-acis/modernizer/ollama"
	"github.com/rwth-acis/modernizer/weaviate"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func main() {

	err := weaviate.InitSchema()
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

	router.GET("/weaviate/promptcount", func(c *gin.Context) {
		searchQuery := c.Query("query")

		// Decode the search query
		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		log.Printf("Decoded Query: %s", decodedQuery)

		count, err := weaviate.RetrievePromptCount(decodedQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(strconv.Itoa(count)))
	})

	router.POST("/generate", func(c *gin.Context) {
		// Parse the JSON request body
		var requestBody map[string]interface{}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Call ollama.GenerateResponse with the parsed request body
		response, err := ollama.GenerateResponse(requestBody)
		if err != nil {
			return
		}

		// Return the response
		c.JSON(http.StatusOK, response)
	})

	err = router.Run(":8080")
	if err != nil {
		return
	}
}
