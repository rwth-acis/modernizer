package main

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rwth-acis/modernizer/ollama"
	"github.com/rwth-acis/modernizer/redis"
	"github.com/rwth-acis/modernizer/weaviate"
)

func main() {

	err := weaviate.InitSchema()
	if err != nil {
		panic(err)
	}

	redis.InitRedis()

	router := gin.Default()

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

	router.GET("/weaviate/retrieveresponse", func(c *gin.Context) {
		searchQuery := c.Query("query")

		// Decode the search query
		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		log.Printf("Decoded Query: %s", decodedQuery)

		response, err := weaviate.RetrieveResponse(decodedQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(response))
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

	router.POST("/vote", func(c *gin.Context) {
		// Parse the JSON request body
		var requestBody map[string]interface{}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Call ollama.GenerateResponse with the parsed request body
		err := weaviate.UpdateRankPrompt(requestBody)
		if err != nil {
			return
		}

		// Return the response
		c.JSON(http.StatusOK, "OK")
	})

	router.GET("/get-instruct", func(c *gin.Context) {
		setName := c.Query("set")
		getAll := c.Query("all") == "true"

		var result interface{}
		var err error
		if getAll {
			result, err = redis.GetSet(setName)
		} else {
			result, err = redis.GetSetMember(setName)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": result})
	})

	router.POST("/add-instruct", redis.AddInstruct)
	router.POST("/del-instruct", redis.DeleteInstruct)

	err = router.Run(":8080")
	if err != nil {
		return
	}
}
