package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"scheduleApp/internal/db"
	"scheduleApp/internal/handlers"
	"scheduleApp/internal/middleware"
	"scheduleApp/internal/web"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed resources/*
var resourceFiles embed.FS

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

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.SetHTMLTemplate(web.Tmpl)

	subStatic, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Ошибка получения поддиректории: %v", err)
	}
	r.StaticFS("/static", http.FS(subStatic))

	subResources, err := fs.Sub(resourceFiles, "resources")
	if err != nil {
		log.Fatalf("Ошибка получения поддиректории: %v", err)
	}
	r.StaticFS("/resources", http.FS(subResources))

	// Публичные маршруты
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index_guest", gin.H{
			"Title": "Главная страница (Гость)",
		})
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login", gin.H{"Title": "Авторизация"})
	})
	r.POST("/login", func(c *gin.Context) {
		handlers.LoginFormHandler(c, dbConn)
	})

	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register", gin.H{"Title": "Регистрация"})
	})
	r.POST("/register", func(c *gin.Context) {
		handlers.RegisterFormHandler(c, dbConn)
	})

	// Приватные маршруты
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware)
	{
		auth.GET("/logout", func(c *gin.Context) {
			handlers.LogoutHandler(c)
		})

		auth.GET("/schedules", func(c *gin.Context) {
			handlers.RenderSchedulesPage(c, dbConn)
		})
		auth.GET("/requests", func(c *gin.Context) {
			handlers.RenderUserRequestsPage(c, dbConn)
		})
		auth.POST("/requests", func(c *gin.Context) {
			handlers.CreateRequestHandler(c, dbConn)
			c.Redirect(http.StatusSeeOther, "/requests")
		})
	}

	// Админские маршруты
	admin := auth.Group("/admin")
	admin.Use(middleware.RoleMiddleware("admin"))
	{
		admin.GET("/schedules", func(c *gin.Context) {
			handlers.RenderAdminSchedulesPageWithFilters(c, dbConn)
		})
		admin.POST("/schedules", func(c *gin.Context) {
			handlers.CreateScheduleFormHandler(c, dbConn)
			c.Redirect(http.StatusSeeOther, "/admin/schedules")
		})
		admin.POST("/schedules/:id", func(c *gin.Context) {
			method := c.Query("_method")
			if method == "PUT" {
				handlers.UpdateScheduleFormHandler(c, dbConn)
			} else if method == "DELETE" {
				handlers.DeleteScheduleHandler(c, dbConn)
			}
			c.Redirect(http.StatusSeeOther, "/admin/schedules")
		})

		admin.GET("/requests", func(c *gin.Context) {
			handlers.RenderAdminRequestsPage(c, dbConn)
		})
		admin.POST("/requests/:id", func(c *gin.Context) {
			action := c.Query("_action")
			handlers.ProcessRequestFormHandler(c, dbConn, action)
			c.Redirect(http.StatusSeeOther, "/admin/requests")
		})

		admin.GET("/users", func(c *gin.Context) {
			handlers.RenderManageUserRolesPage(c, dbConn)
		})
		admin.POST("/users/:id", func(c *gin.Context) {
			handlers.UpdateUserRoleHandler(c, dbConn)
		})
	}

	log.Println("Сервер запущен на :8080")
	r.Run(":8080")
}
