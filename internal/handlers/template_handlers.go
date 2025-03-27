package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RenderSchedulesPage рендерит страницу расписания для обычных пользователей (student/teacher)
func RenderSchedulesPage(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, subject_id, teacher_id, classroom_id, start_time, end_time, created_at
        FROM schedule
        ORDER BY start_time
    `)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		if err := rows.Scan(&s.ID, &s.SubjectID, &s.TeacherID, &s.ClassroomID,
			&s.StartTime, &s.EndTime, &s.CreatedAt); err != nil {
			c.HTML(http.StatusInternalServerError, "schedules_user", gin.H{
				"Title": "Расписание",
				"Error": err.Error(),
			})
			return
		}
		schedules = append(schedules, s)
	}

	c.HTML(http.StatusOK, "schedules_user", gin.H{
		"Title":     "Расписание",
		"Schedules": schedules,
	})
}

// RenderAdminSchedulesPage рендерит страницу управления расписанием для админа
func RenderAdminSchedulesPage(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, subject_id, teacher_id, classroom_id, start_time, end_time, created_at
        FROM schedule
        ORDER BY start_time
    `)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		if err := rows.Scan(&s.ID, &s.SubjectID, &s.TeacherID, &s.ClassroomID,
			&s.StartTime, &s.EndTime, &s.CreatedAt); err != nil {
			c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
				"Title": "Управление расписанием (Admin)",
				"Error": err.Error(),
			})
			return
		}
		schedules = append(schedules, s)
	}

	c.HTML(http.StatusOK, "schedules_admin", gin.H{
		"Title":     "Управление расписанием (Admin)",
		"Schedules": schedules,
	})
}

// CreateScheduleFormHandler создает новую запись в расписании (админ)
func CreateScheduleFormHandler(c *gin.Context, db *sql.DB) {
	subjectID, _ := strconv.Atoi(c.PostForm("subject_id"))
	teacherID, _ := strconv.Atoi(c.PostForm("teacher_id"))
	classroomID, _ := strconv.Atoi(c.PostForm("classroom_id"))
	startTimeStr := c.PostForm("start_time")
	endTimeStr := c.PostForm("end_time")

	layout := "2006-01-02T15:04"
	startTime, _ := time.Parse(layout, startTimeStr)
	endTime, _ := time.Parse(layout, endTimeStr)

	_, err := db.Exec(`
        INSERT INTO schedule (subject_id, teacher_id, classroom_id, start_time, end_time)
        VALUES ($1, $2, $3, $4, $5)
    `, subjectID, teacherID, classroomID, startTime, endTime)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": err.Error(),
		})
		return
	}
}

// UpdateScheduleFormHandler обновляет существующую запись расписания (админ)
func UpdateScheduleFormHandler(c *gin.Context, db *sql.DB) {
	scheduleID := c.Param("id")

	subjectID, _ := strconv.Atoi(c.PostForm("subject_id"))
	teacherID, _ := strconv.Atoi(c.PostForm("teacher_id"))
	classroomID, _ := strconv.Atoi(c.PostForm("classroom_id"))
	startTimeStr := c.PostForm("start_time")
	endTimeStr := c.PostForm("end_time")

	layout := "2006-01-02T15:04"
	startTime, _ := time.Parse(layout, startTimeStr)
	endTime, _ := time.Parse(layout, endTimeStr)

	_, err := db.Exec(`
        UPDATE schedule
        SET subject_id=$1, teacher_id=$2, classroom_id=$3, start_time=$4, end_time=$5
        WHERE id=$6
    `, subjectID, teacherID, classroomID, startTime, endTime, scheduleID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": err.Error(),
		})
		return
	}
}

// DeleteScheduleHandler удаляет запись расписания (админ)
func DeleteScheduleHandler(c *gin.Context, db *sql.DB) {
	scheduleID := c.Param("id")
	_, err := db.Exec("DELETE FROM schedule WHERE id=$1", scheduleID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": err.Error(),
		})
		return
	}
}

// RenderUserRequestsPage рендерит страницу с запросами для обычных пользователей (student/teacher)
func RenderUserRequestsPage(c *gin.Context, db *sql.DB) {
	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(int)

	rows, err := db.Query(`
        SELECT id, user_id, schedule_id, desired_change, status
        FROM requests
        WHERE user_id=$1
        ORDER BY id
    `, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "requests_user", gin.H{
			"Title": "Мои запросы",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		var r models.Request
		if err := rows.Scan(&r.ID, &r.UserID, &r.ScheduleID, &r.DesiredChange, &r.Status); err != nil {
			c.HTML(http.StatusInternalServerError, "requests_user", gin.H{
				"Title": "Мои запросы",
				"Error": err.Error(),
			})
			return
		}
		requests = append(requests, r)
	}

	c.HTML(http.StatusOK, "requests_user", gin.H{
		"Title":    "Мои запросы",
		"Requests": requests,
	})
}

// RenderAdminRequestsPage рендерит страницу с запросами для админа
func RenderAdminRequestsPage(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, user_id, schedule_id, desired_change, status
        FROM requests
        ORDER BY id
    `)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "requests_admin", gin.H{
			"Title": "Запросы (Admin)",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		var r models.Request
		if err := rows.Scan(&r.ID, &r.UserID, &r.ScheduleID, &r.DesiredChange, &r.Status); err != nil {
			c.HTML(http.StatusInternalServerError, "requests_admin", gin.H{
				"Title": "Запросы (Admin)",
				"Error": err.Error(),
			})
			return
		}
		requests = append(requests, r)
	}

	c.HTML(http.StatusOK, "requests_admin", gin.H{
		"Title":    "Запросы (Admin)",
		"Requests": requests,
	})
}

func RenderManageUserRolesPage(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, username, email, role
        FROM users
        ORDER BY id
    `)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role); err != nil {
			c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
				"Title": "Управление пользователями",
				"Error": err.Error(),
			})
			return
		}
		users = append(users, u)
	}

	c.HTML(http.StatusOK, "manage_users", gin.H{
		"Title": "Управление пользователями",
		"Users": users,
	})
}

func UpdateUserRoleHandler(c *gin.Context, db *sql.DB) {
	userIDStr := c.Param("id")
	newRole := c.PostForm("role")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": "Неверный ID пользователя",
		})
		return
	}

	// Обновляем роль пользователя
	_, err = db.Exec(`UPDATE users SET role=$1 WHERE id=$2`, newRole, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/users")
}

func ProcessRequestFormHandler(c *gin.Context, db *sql.DB, action string) {
	reqID := c.Param("id")
	status := ""
	if action == "approve" {
		status = "approved"
	} else if action == "reject" {
		status = "rejected"
	} else {
		c.HTML(http.StatusBadRequest, "requests_admin", gin.H{
			"Title": "Запросы (Admin)",
			"Error": "Неверное действие",
		})
		return
	}

	_, err := db.Exec(`UPDATE requests SET status=$1 WHERE id=$2`, status, reqID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "requests_admin", gin.H{
			"Title": "Запросы (Admin)",
			"Error": err.Error(),
		})
		return
	}
}
