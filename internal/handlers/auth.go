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
		c.HTML(http.StatusUnauthorized, "login", gin.H{
			"Title": "Авторизация",
			"Error": "Пользователь не найден",
		})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		c.HTML(http.StatusUnauthorized, "login", gin.H{
			"Title": "Авторизация",
			"Error": "Неверный пароль",
		})
		return
	}

	token, err := middleware.GenerateJWT(user)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login", gin.H{
			"Title": "Авторизация",
			"Error": "Ошибка генерации токена",
		})
		return
	}

	c.SetCookie("token", token, 3600, "/", "", false, true)

	switch user.Role {
	case "admin":
		c.HTML(http.StatusOK, "index_admin", gin.H{
			"Title":   "Главная (Админ)",
			"Message": "Добро пожаловать, администратор!",
			"Token":   token,
		})
	case "teacher", "student":
		c.HTML(http.StatusOK, "index_user", gin.H{
			"Title":   "Главная (Пользователь)",
			"Message": "Добро пожаловать!",
			"Token":   token,
		})
	default:
		c.HTML(http.StatusOK, "index_guest", gin.H{
			"Title":   "Главная (Гость)",
			"Message": "Добро пожаловать!",
		})
	}
}

func RegisterFormHandler(c *gin.Context, db *sql.DB) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	role := "student"

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register", gin.H{
			"Title": "Регистрация",
			"Error": "Ошибка хэширования пароля",
		})
		return
	}

	var userID int
	err = db.QueryRow(`
        INSERT INTO users (username, password, email, role)
        VALUES ($1, $2, $3, $4) RETURNING id
    `, username, string(hashedPwd), email, role).Scan(&userID)
	if err != nil {
		c.HTML(http.StatusConflict, "register", gin.H{
			"Title": "Регистрация",
			"Error": "Ошибка при регистрации: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "login", gin.H{
		"Title":   "Авторизация",
		"Message": "Регистрация успешно завершена! Теперь войдите в систему.",
	})
}

func LogoutHandler(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}
