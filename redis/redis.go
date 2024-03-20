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
		"How would you handle errors in this code?",
		"What testing strategies would you use to test this code?",
		"How would you handle concurrency in this code?",
		"Explain me how you would refactor this code to make it more readable:",
		"Explain me how you would refactor this code to make it more performant:",
		"Explain me how you would refactor this code to make it more maintainable:",
		"Explain me how you would refactor this code to make it more testable:",
		"Can you explain the design decisions behind this code?",
	}

	rdb.SAdd(ctx, "developer", members...)

	members = []interface{}{
		"What security considerations should be taken into account when using this code?",
		"Are there any security problems in this code:",
		"What encryption algorithms are used to secure sensitive data?",
		"Explain me how you would secure this code against SQL injection attacks:",
		"Explain me how you would secure this code against XSS attacks:",
		"How would you ensure compliance with security standards in this code?",
		"Does this code comply with GDPR?",
		"What authentication and authorization mechanisms are used in this code?",
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

	members = []interface{}{
		"What is the overall architecture of this code?",
		"What technologies and frameworks are used in this code?",
		"How will this code handle scalability?",
		"What design patterns or architectural patterns are used in this code?",
		"What are the trade-offs of using this code?",
		"What considerations have been made for future maintenance and updates?",
	}

	rdb.SAdd(ctx, "architecture", members...)

	members = []interface{}{
		"What business impact does this code have?",
		"What are the business requirements for this code?",
		"How would you prioritize tasks and allocate workload among team members?",
	}

	rdb.SAdd(ctx, "project-management", members...)

	members = []interface{}{
		"What is the purpose of this function, and does it adhere to the Single Responsibility Principle (SRP)?",
		"What dependencies does this function have, and can they be minimized or eliminated?",
		"Does this function exhibit any code smells, such as long parameter lists or excessive branching?",
		"What level of technical debt does this function carry, and how can it be reduced?",
		"Are there any performance bottlenecks or inefficiencies in this function?",
		"Does this function handle error and exception cases effectively?",
		"Is this function well-documented, and does it have sufficient unit test coverage?",
		"What design patterns or architectural principles can be applied to improve this function?",
		"Can this function be optimized for concurrency or parallelism?",
		"How can this function be modularized or decoupled to promote reusability and maintainability?",
	}

	rdb.SAdd(ctx, "modernisation", members...)

	members = []interface{}{
		"Explain me this:",
		"How does the following code work?",
		"Explain me step by step how this code works:",
		"Be concise and explain this code:",
		"What does this code do?",
		"What is the semantic meaning of this code?",
	}

	rdb.SAdd(ctx, "explanation", members...)

	members = []interface{}{
		"What are the inputs required for this function, and what are their expected formats and constraints?",
		"Are there any boundary conditions or edge cases that need to be tested for this function?",
		"Are there any dependencies or external factors that may impact the behavior of this function during testing?",
		"Can you describe any assumptions or preconditions that must be met for this function to behave as expected?",
		"Are there any error handling mechanisms implemented within this function, and how do they handle unexpected inputs or exceptions?",
		"Are there any side effects or unintended consequences of calling this function that need to be tested?",
	}

	rdb.SAdd(ctx, "test-engineering", members...)

	members = []interface{}{
		"What part of this file needs to be modernized first?",
		"What part of this file contains the most complexity and needs to dealt with?",
		"How are dependencies managed within this file or module?",
		"Can you identify any potential performance bottlenecks within this file?",
		"Can you discuss any efforts or plans to refactor or modernize this file to improve maintainability?",
		"How is the modularity and cohesion of this file assessed in terms of maintainability?",
		"Can you discuss any efforts or plans to refactor or modernize this file to improve testability?",
		"Are there any specific coding standards or guidelines followed within this file to improve maintainability?",
		"Are the inline comments enough to understand the complex parts of this file?",
		"Can you provide insights into any technical debt backlog items related to this file and their prioritization?",
	}

	rdb.SAdd(ctx, "file-based", members...)

	members = []interface{}{
		"Can you identify redundant or duplicate code blocks within the codebase?",
		"Can you analyze the codebase to determine the developer's preferred coding style or patterns?",
		"Can you detect any intentional obfuscation or encryption techniques used within the codebase for security purposes?",
		"Does the codebase show recurring themes or motifs that reflect the underlying philosophy or ideology of the developers?",
		"Based on the codebase what target audience is the code written for?",
		"When taking Conway's law into account, what team structure can you derive from the codebase?",
		"When taking Conway's law into account, what team structure can you derive from the codebase and which of those are anti-patterns or worsen the architecture of the software?",
		"Can you identify any recurring anti-patterns or code smells within the codebase?",
		"Give me the code performance of each function or class in the O-Notation.",
	}

	rdb.SAdd(ctx, "miscellaneous", members...)
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

	listName := requestData.List
	if listName == "" {
		listName = "default"
	}

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

func GetAllSets(c *gin.Context) {
	rdb := loadClient()

	ctx := c.Request.Context()
	keysCmd := rdb.Keys(ctx, "*") // Get all keys matching the pattern "*"

	keys, err := keysCmd.Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sets []string
	for _, key := range keys {
		typeCmd := rdb.Type(ctx, key) // Get the type of the key

		keyType, err := typeCmd.Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if keyType == "set" {
			sets = append(sets, key)
		}
	}

	c.JSON(http.StatusOK, sets)

	return
}

func DeleteAllSets() {
	rdb := loadClient()

	keysCmd := rdb.Keys(context.Background(), "*") // Get all keys matching the pattern "*"

	keys, err := keysCmd.Result()
	if err != nil {
		return
	}

	for _, key := range keys {
		typeCmd := rdb.Type(context.Background(), key) // Get the type of the key

		keyType, err := typeCmd.Result()
		if err != nil {
			return
		}

		if keyType == "set" {
			rdb.Del(context.Background(), key)
		}
	}
}
