package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"scheduleApp/internal/db"
	"scheduleApp/internal/handlers"
	"scheduleApp/internal/middleware"
	"scheduleApp/internal/web"
)

func main() {
	dbConn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer dbConn.Close()

	if err := db.CreateTables(dbConn); err != nil {
		log.Fatalf("Ошибка при создании таблиц: %v", err)
	}

	web.InitTemplates()

	r := gin.Default()

	r.SetHTMLTemplate(web.Tmpl)

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Title":   "Главная страница",
			"Message": "Добро пожаловать в систему расписания!",
		})
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.POST("/login", func(c *gin.Context) {
		handlers.LoginFormHandler(c, dbConn)
	})

	r.GET("/register", func(c *gin.Context) {
		c.String(http.StatusOK, "Страница регистрации (добавьте register.html при необходимости)")
	})

	r.POST("/register", func(c *gin.Context) {
		handlers.RegisterFormHandler(c, dbConn)
	})

	// Нужен логин
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware)
	auth.GET("/schedules", func(c *gin.Context) {
		handlers.GetSchedulesHandler(c, dbConn)
	})
	auth.POST("/requests", func(c *gin.Context) {
		handlers.CreateRequestHandler(c, dbConn)
	})

	// Нужен админ логин
	admin := auth.Group("/admin")
	admin.Use(middleware.RoleMiddleware("admin"))
	// CRUD для расписания
	admin.POST("/schedules", func(c *gin.Context) {
		handlers.CreateScheduleHandler(c, dbConn)
	})
	admin.PUT("/schedules/:id", func(c *gin.Context) {
		handlers.UpdateScheduleHandler(c, dbConn)
	})
	admin.DELETE("/schedules/:id", func(c *gin.Context) {
		handlers.DeleteScheduleHandler(c, dbConn)
	})
	// Обработка запросов на изменение
	admin.GET("/requests", func(c *gin.Context) {
		handlers.GetAllRequestsHandler(c, dbConn)
	})
	admin.PUT("/requests/:id", func(c *gin.Context) {
		handlers.ProcessRequestHandler(c, dbConn)
	})

	log.Println("Сервер запущен на порту :8080")
	r.Run(":8080")
}
