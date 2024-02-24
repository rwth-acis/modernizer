package redis

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func loadClient() (rdb *redis.Client) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // Use default DB.
	})

	return rdb
}

func InitRedis() {
	ctx := context.Background()
	rdb := loadClient()

	members := []interface{}{"Explain me this:", "How does the following code work?"}
	rdb.SAdd(ctx, "default", members...)
}

func addItem(c *gin.Context) {

	rdb := loadClient()

	var requestData struct {
		Item string `json:"item"`
		List string `json:"list,omitempty"`
	}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Set the default list name to "default" if not provided.
	listName := requestData.List
	if listName == "" {
		listName = "default"
	}

	// Add the item to the specified redis set.
	ctx := context.Background()
	if err := rdb.SAdd(ctx, listName, requestData.Item).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func deleteItem(c *gin.Context) {

	rdb := loadClient()

	var requestData struct {
		Item string `json:"item"`
		List string `json:"list,omitempty"`
	}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Set the default list name to "default" if not provided.
	listName := requestData.List
	if listName == "" {
		listName = "default"
	}

	// Remove the item from the specified redis set.
	ctx := context.Background()
	if err := rdb.SRem(ctx, listName, requestData.Item).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func GetList(c *gin.Context) {

	rdb := loadClient()

	listName := c.Query("list")
	getAll := c.Query("all") == "true"

	if listName == "" {
		listName = "default"
	}

	if getAll {
		ctx := context.Background()
		vals, err := rdb.SMembers(ctx, listName).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"list": listName, "objects": vals})
		return
	}

	ctx := context.Background()
	vals, err := rdb.SRandMember(ctx, listName).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": listName, "objects": vals})
}
