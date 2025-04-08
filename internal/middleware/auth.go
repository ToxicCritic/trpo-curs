package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"scheduleApp/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var SECRET_KEY = []byte("MY_SUPER_SECRET_KEY")

type JWTClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(c *gin.Context) {
	var tokenString string

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		}
	}
	if tokenString == "" {
		if cookieToken, err := c.Cookie("token"); err == nil {
			tokenString = cookieToken
		}
	}
	if tokenString == "" {
		log.Println("DEBUG: Токен не найден ни в заголовке, ни в куки.")
		c.Redirect(http.StatusSeeOther, "/login?alarm=Токен+не+найден")
		c.Abort()
		return
	}

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return SECRET_KEY, nil
	})
	if err != nil || !token.Valid {
		log.Printf("DEBUG: Ошибка парсинга токена: %v", err)
		if errors.Is(err, jwt.ErrTokenExpired) {
			c.Redirect(http.StatusSeeOther, "/login?alarm=Ваш+токен+истек,+войдите+снова")
			c.Abort()
			return
		}
		c.Redirect(http.StatusSeeOther, "/login?alarm=Неверный+или+недействительный+токен")
		c.Abort()
		return
	}
	// Логируем полученные claims
	log.Printf("DEBUG: Parsed JWT claims: UserID=%d, Role=%q", claims.UserID, claims.Role)

	// Если роль отсутствует, можно задать дефолтное значение или вернуть ошибку.
	if claims.Role == "" {
		log.Println("DEBUG: Поле Role отсутствует в JWT claims")
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found"})
		return
	}

	c.Set("user_id", claims.UserID)
	c.Set("role", claims.Role)
	c.Next()
}

func GenerateJWT(user models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(SECRET_KEY)
}
