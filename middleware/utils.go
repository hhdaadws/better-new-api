package middleware

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const MaskedErrorMessage = "Internal Server Error"

func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...string) {
	codeStr := ""
	if len(code) > 0 {
		codeStr = code[0]
	}
	userId := c.GetInt("id")
	// 先记录真实错误日志
	logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, message))
	// 如果开启了错误伪装，替换返回给用户的消息
	displayMessage := message
	if operation_setting.ShouldMaskErrorMessage() {
		displayMessage = MaskedErrorMessage
	}
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": common.MessageWithRequestId(displayMessage, c.GetString(common.RequestIdKey)),
			"type":    "new_api_error",
			"code":    codeStr,
		},
	})
	c.Abort()
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	// 先记录真实错误日志
	logger.LogError(c.Request.Context(), description)
	// 如果开启了错误伪装，替换返回给用户的消息
	displayDescription := description
	if operation_setting.ShouldMaskErrorMessage() {
		displayDescription = MaskedErrorMessage
	}
	c.JSON(statusCode, gin.H{
		"description": displayDescription,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
}
