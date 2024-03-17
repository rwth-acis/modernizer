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

	router := gin.New()

	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/weaviate/promptcount", "/weaviate"},
	}))
	router.Use(gin.Recovery())

	router.GET("/weaviate/promptcount", func(c *gin.Context) {
		searchQuery := c.Query("query")
		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		count, err := weaviate.RetrievePromptCount(decodedQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(strconv.Itoa(count)))
	})

	router.GET("/weaviate/retrieveresponse", func(c *gin.Context) {

		//TODO: allow user annotation

		searchQuery := c.Query("query")
		upvoteStr := c.Query("best")
		best := upvoteStr == "true"

		// Decode the search query
		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		log.Printf("Decoded Query: %s", decodedQuery)

		var response interface{}

		if best {
			response, err = weaviate.RetrieveBestResponse(decodedQuery)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			response, err = weaviate.RetrieveRandomResponse(decodedQuery)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/weaviate/retrieveresponselist", func(c *gin.Context) {
		searchQuery := c.Query("query")

		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		log.Printf("Decoded Query: %s", decodedQuery)

		responseList, err := weaviate.ResponseList(decodedQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, responseList)
	})

	router.GET("/weaviate/responsebyid", func(c *gin.Context) {
		searchQuery := c.Query("id")

		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		response, err := weaviate.RetrieveResponseByID(decodedQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/weaviate/propertiesbyid", func(c *gin.Context) {
		id := c.Query("id")

		response, err := weaviate.RetrieveProperties(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/get-similar-code", func(c *gin.Context) {
		searchQuery := c.Query("query")

		decodedQuery, err := url.QueryUnescape(searchQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search query"})
			return
		}

		response, err := SemanticSimilarity(decodedQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	router.POST("/generate", func(c *gin.Context) {
		var requestBody map[string]interface{}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		response, err := ollama.GenerateResponse(requestBody)
		if err != nil {
			return
		}

		c.JSON(http.StatusOK, response)
	})
	router.POST("/vote", func(c *gin.Context) {

		upvoteStr := c.Query("upvote")
		upvote := upvoteStr == "true"

		var requestBody map[string]interface{}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("%v\n", requestBody)

		if upvote {
			err := weaviate.UpdateRankPrompt(requestBody, true)
			if err != nil {
				log.Printf("%v", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		} else {
			err := weaviate.UpdateRankPrompt(requestBody, false)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%v", err)
				return
			}
		}

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
	router.GET("/get-all-sets", redis.GetAllSets)

	err = router.Run(":8080")
	if err != nil {
		return
	}
}

func SemanticSimilarity(code string) ([]string, error) {
	PromptExists, exists := weaviate.RetrieveHasSemanticMeaning(code)
	if !exists {
		SemanticMeaning := ollama.SemanticMeaning("", code, false)

		similarCode, err := weaviate.GetSimilarSemanticMeaning(SemanticMeaning)
		if err != nil {
			return nil, err
		}
		return similarCode, err
	} else {
		similarCode, err := weaviate.GetSimilarSemanticMeaning(PromptExists)
		if err != nil {
			log.Printf("error: %v", err)
		}

		return similarCode, err
	}

}
