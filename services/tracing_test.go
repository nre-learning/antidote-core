package services

import (
	"strings"
	"testing"
)

func TestSafePayload(t *testing.T) {
	largePayload := strings.Repeat("loremipsumdolor", 5000)
	result := SafePayload(largePayload)
	assert(t, (result[len(result)-10:] == "ipsumdolor"), "")

	smallPayload := strings.Repeat("loremipsumdolor", 10)
	result = SafePayload(smallPayload)
	assert(t, (result == smallPayload), "")
}
