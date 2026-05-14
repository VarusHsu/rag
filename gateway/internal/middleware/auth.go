package middleware

import (
	"net/http"
	"strings"

	"gateway/internal/security"

	"github.com/gin-gonic/gin"
)

const (
	ContextToken  = "auth_token"
	ContextClaims = "auth_claims"
)

func RequireAuth(jwtManager *security.JWTManager, tokenRevoker security.TokenRevoker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			abortUnauthorized(c, "missing or invalid authorization header")
			return
		}

		if tokenRevoker != nil && tokenRevoker.IsRevoked(token) {
			abortUnauthorized(c, "token has been revoked")
			return
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			abortUnauthorized(c, "invalid token")
			return
		}

		c.Set(ContextToken, token)
		c.Set(ContextClaims, claims)
		c.Next()
	}
}

func GetToken(c *gin.Context) string {
	value, ok := c.Get(ContextToken)
	if !ok {
		return ""
	}
	token, ok := value.(string)
	if !ok {
		return ""
	}
	return token
}

func GetClaims(c *gin.Context) *security.Claims {
	value, ok := c.Get(ContextClaims)
	if !ok {
		return nil
	}
	claims, ok := value.(*security.Claims)
	if !ok {
		return nil
	}
	return claims
}

func extractBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error":      message,
		"request_id": GetRequestID(c),
	})
}
