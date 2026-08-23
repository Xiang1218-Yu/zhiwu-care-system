package middleware

import (
	"strings"

	"plant-diary/internal/service"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "user_id"

func RequireAuth(auth *service.AuthService, html bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else if cookie, err := c.Cookie("plant_diary_token"); err == nil {
			token = cookie
		}

		userID, err := auth.ParseToken(token)
		if err != nil {
			if html {
				c.Redirect(302, "/login")
			} else {
				c.AbortWithStatusJSON(401, gin.H{"error": "未登录或登录已过期"})
			}
			c.Abort()
			return
		}
		c.Set(UserIDKey, userID)
		c.Next()
	}
}

func UserID(c *gin.Context) string {
	value, _ := c.Get(UserIDKey)
	userID, _ := value.(string)
	return userID
}
