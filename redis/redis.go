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

	members := []interface{}{
		"Explain me this:",
		"How does the following code work?",
		"Explain me step by step how this code works:",
	}

	rdb.SAdd(ctx, "default", members...)

	members = []interface{}{
		"What security considerations should be taken into account when using this code?",
		"Are there any security problems in this code:",
	}

	rdb.SAdd(ctx, "security", members...)

	members = []interface{}{
		"Explain me what this piece of code does like angry Linux Torvalds on Linux kernel code reviews:",
		"Explain this code as if you were a wizard casting a spell.",
		"Pretend you're a detective solving a mystery related to this code.",
		"Explain this code as if you were a teacher explaining a concept to a student.",
		"Describe this code using only emojis and internet slang.",
	}

	rdb.SAdd(ctx, "funny", members...)

}

func AddInstruct(c *gin.Context) {

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

	c.JSON(http.StatusOK, "added Item to list: "+listName)
}

func DeleteInstruct(c *gin.Context) {

	rdb := loadClient()

	var requestData struct {
		Item string `json:"item"`
		Set  string `json:"set,omitempty"`
	}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	listName := requestData.Set
	if listName == "" {
		listName = "default"
	}

	ctx := context.Background()
	if err := rdb.SRem(ctx, listName, requestData.Item).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// GetSet retrieves the whole set from Redis
func GetSet(setName string) ([]string, error) {
	rdb := loadClient()

	if setName == "" {
		setName = "default"
	}

	ctx := context.Background()
	vals, err := rdb.SMembers(ctx, setName).Result()
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// GetSetMember retrieves a single member from Redis
func GetSetMember(setName string) (string, error) {
	rdb := loadClient()

	if setName == "" {
		setName = "default"
	}

	ctx := context.Background()
	val, err := rdb.SRandMember(ctx, setName).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}
