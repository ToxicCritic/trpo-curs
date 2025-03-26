package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/middleware"
	"scheduleApp/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func LoginFormHandler(c *gin.Context, db *sql.DB) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	var user models.User
	err := db.QueryRow(`
        SELECT id, username, password, email, role
        FROM users
        WHERE username=$1
    `, username).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"error": "Неверный пароль",
		})
		return
	}

	token, err := middleware.GenerateJWT(user)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"error": "Ошибка генерации токена",
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Title":   "Добро пожаловать!",
		"Message": "Ваш токен: " + token,
	})
}

func RegisterFormHandler(c *gin.Context, db *sql.DB) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	role := c.PostForm("role") // admin, teacher, student

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "index.html", gin.H{
			"error": "Ошибка хэширования пароля",
		})
		return
	}

	var userID int
	err = db.QueryRow(`
        INSERT INTO users (username, password, email, role)
        VALUES ($1, $2, $3, $4) RETURNING id
    `, username, string(hashedPwd), email, role).Scan(&userID)
	if err != nil {
		c.HTML(http.StatusConflict, "index.html", gin.H{
			"error": "Ошибка при регистрации: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Message": "Регистрация успешно завершена!",
	})
}
