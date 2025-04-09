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

	// Подключаем статику
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
	r.Static("/uploads", "./uploads")
	// Публичные маршруты
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index_guest", gin.H{
			"Title": "Главная страница (Гость)",
		})
	})
	r.GET("/login", func(c *gin.Context) {
		handlers.RenderLoginPage(c)
	})
	r.POST("/login", func(c *gin.Context) {
		handlers.LoginFormHandler(c, dbConn)
	})
	r.GET("/register", func(c *gin.Context) {
		handlers.RenderRegisterPage(c, dbConn)
	})
	r.POST("/register", func(c *gin.Context) {
		handlers.RegisterFormHandler(c, dbConn)
	})

	user := r.Group("/")
	user.Use(middleware.AuthMiddleware)
	{
		user.GET("/logout", func(c *gin.Context) {
			handlers.LogoutHandler(c)
		})
	}

	student := r.Group("/student")
	student.Use(middleware.AuthMiddleware, middleware.RoleMiddleware("student"))
	{
		student.GET("/comments", func(c *gin.Context) {
			handlers.RenderStudentComments(c, dbConn)
		})
		student.GET("/schedules", func(c *gin.Context) {
			handlers.RenderStudentSchedule(c, dbConn)
		})
		student.GET("/requests", func(c *gin.Context) {
			handlers.RenderUserRequestsPage(c, dbConn)
		})
		student.POST("/requests", func(c *gin.Context) {
			handlers.CreateRequestHandler(c, dbConn)
			c.Redirect(http.StatusSeeOther, "/user/requests")
		})
	}

	// Группа для админа
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware, middleware.RoleMiddleware("admin"))
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
		admin.GET("/schedules/:id/json", func(c *gin.Context) {
			handlers.GetScheduleJSON(c, dbConn)
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
		admin.POST("/users/:id/group", func(c *gin.Context) {
			handlers.UpdateStudentGroupHandler(c, dbConn)
		})
	}

	// Группа для учителя
	teacher := r.Group("/teacher")
	teacher.Use(middleware.AuthMiddleware, middleware.RoleMiddleware("teacher"))
	{
		teacher.GET("/logout", func(c *gin.Context) {
			handlers.LogoutHandler(c)
		})
		teacher.GET("/schedule", func(c *gin.Context) {
			handlers.RenderTeacherSchedule(c, dbConn)
		})
		teacher.GET("/comments", func(c *gin.Context) {
			handlers.RenderTeacherComments(c, dbConn)
		})
		teacher.POST("/comments/:id", func(c *gin.Context) {
			handlers.CreateTeacherComment(c, dbConn)
		})
		teacher.GET("/requests", func(c *gin.Context) {
			handlers.RenderTeacherRequests(c, dbConn)
		})
		teacher.POST("/requests", func(c *gin.Context) {
			handlers.CreateTeacherRequest(c, dbConn)
			c.Redirect(http.StatusSeeOther, "/teacher/requests")
		})
		teacher.GET("/", func(c *gin.Context) {
			handlers.RenderIndexTeacher(c, dbConn)
		})
	}

	log.Println("Сервер запущен на :8080")
	r.Run(":8080")
}
