package handlers

import (
	"fmt"
	"time"
)

func generateResponseID(requestID string) string {
	return fmt.Sprintf("resp_%s_%d", requestID, time.Now().UnixNano())
}
