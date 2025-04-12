package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/middleware"
	"scheduleApp/internal/models"
	"strconv"

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
	case "teacher":
		c.HTML(http.StatusOK, "index_teacher", gin.H{
			"Title":   "Главная (Преподаватель)",
			"Message": "Добро пожаловать, " + user.Username + "!",
			"Token":   token,
		})
	case "student":
		c.HTML(http.StatusOK, "index_student", gin.H{
			"Title":   "Главная (Студент)",
			"Message": "Добро пожаловать, " + user.Username + "!",
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
	groups, err := loadAllGroups(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register", gin.H{
			"Title": "Регистрация",
			"Error": "Ошибка загрузки групп: " + err.Error(),
		})
		return
	}
	departments, err := loadAllDepartments(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register", gin.H{
			"Title":     "Регистрация",
			"Error":     "Ошибка загрузки отделов: " + err.Error(),
			"AllGroups": groups,
		})
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	name := c.PostForm("name")
	role := c.PostForm("role")
	if role == "" {
		role = "student"
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register", gin.H{
			"Title":          "Регистрация",
			"Error":          "Ошибка хэширования пароля",
			"AllGroups":      groups,
			"AllDepartments": departments,
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
			"Title":          "Регистрация",
			"Error":          "Ошибка при регистрации: " + err.Error(),
			"AllGroups":      groups,
			"AllDepartments": departments,
		})
		return
	}

	if role == "student" {
		// Для студента извлекаем group_id из формы.
		groupIDStr := c.PostForm("group_id")
		var groupID int
		if groupIDStr != "" {
			groupID, err = strconv.Atoi(groupIDStr)
			if err != nil {
				c.HTML(http.StatusBadRequest, "register", gin.H{
					"Title":          "Регистрация",
					"Error":          "Неверный ID группы",
					"AllGroups":      groups,
					"AllDepartments": departments,
				})
				return
			}
		}
		_, err = db.Exec(`
          INSERT INTO students (user_id, name, group_id)
          VALUES ($1, $2, $3)
        `, userID, name, groupID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "register", gin.H{
				"Title":          "Регистрация",
				"Error":          "Ошибка при создании записи студента: " + err.Error(),
				"AllGroups":      groups,
				"AllDepartments": departments,
			})
			return
		}
	} else if role == "teacher" {
		departmentIDStr := c.PostForm("department_id")
		var departmentID int
		if departmentIDStr != "" {
			departmentID, err = strconv.Atoi(departmentIDStr)
			if err != nil {
				c.HTML(http.StatusBadRequest, "register", gin.H{
					"Title":          "Регистрация",
					"Error":          "Неверный ID отдела",
					"AllGroups":      groups,
					"AllDepartments": departments,
				})
				return
			}
		}
		_, err = db.Exec(`
          INSERT INTO teachers (user_id, name, department_id)
          VALUES ($1, $2, $3)
        `, userID, name, departmentID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "register", gin.H{
				"Title":          "Регистрация",
				"Error":          "Ошибка при создании записи преподавателя: " + err.Error(),
				"AllGroups":      groups,
				"AllDepartments": departments,
			})
			return
		}
	}

	c.HTML(http.StatusOK, "login", gin.H{
		"Title": "Авторизация",
		"Alarm": "Регистрация успешно завершена! Теперь войдите в систему.",
	})
}

func RenderRegisterPage(c *gin.Context, db *sql.DB) {
	groups, err := loadAllGroups(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register", gin.H{
			"Title": "Регистрация",
			"Error": "Ошибка загрузки групп: " + err.Error(),
		})
		return
	}
	departments, err := loadAllDepartments(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register", gin.H{
			"Title": "Регистрация",
			"Error": "Ошибка загрузки отделов: " + err.Error(),
		})
		return
	}
	c.HTML(http.StatusOK, "register", gin.H{
		"Title":          "Регистрация",
		"AllGroups":      groups,
		"AllDepartments": departments,
	})
}

func LogoutHandler(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}
