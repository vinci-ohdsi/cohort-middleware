package utils

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ParseNumericArg(c *gin.Context, paramName string) (int, error) {
	// parse and validate:
	numericArgValue := c.Param(paramName)
	log.Printf("Querying %s: ", paramName)
	if numericId, err := strconv.Atoi(numericArgValue); err != nil {
		log.Printf("bad request - %s should be a number", paramName)
		return -1, fmt.Errorf("bad request - %s should be a number", paramName)
	} else {
		return numericId, nil
	}
}

// parse and validate mandatory request argument.
func ParseStringArg(c *gin.Context, paramName string) (string, error) {
	// parse and validate:
	stringArgValue := c.Param(paramName)
	log.Printf("Querying %s: ", paramName)
	if stringArgValue == "" {
		log.Printf("bad request - %s should be set", paramName)
		return "", fmt.Errorf("bad request - %s should set", paramName)
	} else {
		return stringArgValue, nil
	}
}
