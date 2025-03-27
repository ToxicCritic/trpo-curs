package handlers

import (
	"database/sql"
	"fmt"
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

func RenderAdminSchedulesPageWithFilters(c *gin.Context, db *sql.DB) {
	// Считываем фильтры из query-параметров (для админа доступны все: group, teacher, classroom)
	groupFilter := c.Query("group")
	teacherFilter := c.Query("teacher")
	classroomFilter := c.Query("classroom")

	// Формируем динамический WHERE
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	if groupFilter != "" {
		// Фильтр по группам через таблицу связи schedule_groups
		whereClauses = append(whereClauses, fmt.Sprintf("sg.group_id = $%d", argIndex))
		args = append(args, groupFilter)
		argIndex++
	}
	if teacherFilter != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.teacher_id = $%d", argIndex))
		args = append(args, teacherFilter)
		argIndex++
	}
	if classroomFilter != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.classroom_id = $%d", argIndex))
		args = append(args, classroomFilter)
		argIndex++
	}

	// Запрос с JOIN для получения связанных значений
	query := `
		SELECT s.id, sub.name as subject_name, t.name as teacher_name, c.room_number,
		       s.start_time, s.end_time, s.created_at
		FROM schedule s
		JOIN subjects sub ON s.subject_id = sub.id
		JOIN teachers t ON s.teacher_id = t.id
		JOIN classrooms c ON s.classroom_id = c.id
		LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
	`
	if len(whereClauses) > 0 {
		query += " WHERE " + joinClauses(whereClauses, " AND ")
	}
	query += " ORDER BY s.start_time;"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": "Ошибка запроса: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	// Группируем расписание по дню недели
	groupedSchedules := make(map[string][]models.ScheduleDisplay)
	for rows.Next() {
		var sch models.ScheduleDisplay
		err := rows.Scan(&sch.ID, &sch.SubjectName, &sch.TeacherName, &sch.RoomNumber,
			&sch.StartTime, &sch.EndTime, &sch.CreatedAt)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
				"Title": "Управление расписанием (Admin)",
				"Error": "Ошибка сканирования строки: " + err.Error(),
			})
			return
		}
		day := sch.StartTime.Weekday().String()
		groupedSchedules[day] = append(groupedSchedules[day], sch)
	}

	// Передаем в шаблон: сгруппированное расписание и активные фильтры
	c.HTML(http.StatusOK, "schedules_admin", gin.H{
		"Title":           "Управление расписанием (Admin)",
		"Schedules":       groupedSchedules,
		"GroupFilter":     groupFilter,
		"TeacherFilter":   teacherFilter,
		"ClassroomFilter": classroomFilter,
	})
}

func joinClauses(clauses []string, sep string) string {
	result := ""
	for i, clause := range clauses {
		if i > 0 {
			result += sep
		}
		result += clause
	}
	return result
}

func CreateScheduleFormHandler(c *gin.Context, db *sql.DB) {
	subjectID, err1 := strconv.Atoi(c.PostForm("subject_id"))
	teacherID, err2 := strconv.Atoi(c.PostForm("teacher_id"))
	classroomID, err3 := strconv.Atoi(c.PostForm("classroom_id"))
	startTimeStr := c.PostForm("start_time")

	if err1 != nil || err2 != nil || err3 != nil || startTimeStr == "" {
		c.HTML(http.StatusBadRequest, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": "Неверные данные формы",
		})
		return
	}

	layout := "2006-01-02T15:04"
	startTime, err := time.Parse(layout, startTimeStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": "Неверный формат времени начала: " + err.Error(),
		})
		return
	}
	endTime := startTime.Add(90 * time.Minute)

	// Проверка коллизий:
	var collisionCount int
	collisionQuery := `
        SELECT COUNT(*) FROM schedule
        WHERE (teacher_id = $1 OR classroom_id = $2)
          AND start_time < $3
          AND end_time > $4
    `
	err = db.QueryRow(collisionQuery, teacherID, classroomID, endTime, startTime).Scan(&collisionCount)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": "Ошибка проверки коллизий: " + err.Error(),
		})
		return
	}
	if collisionCount > 0 {
		c.HTML(http.StatusConflict, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": "Коллизия обнаружена: занятие пересекается с уже существующим.",
		})
		return
	}

	_, err = db.Exec(`
        INSERT INTO schedule (subject_id, teacher_id, classroom_id, start_time, end_time)
        VALUES ($1, $2, $3, $4, $5)
    `, subjectID, teacherID, classroomID, startTime, endTime)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Error": "Ошибка создания записи: " + err.Error(),
		})
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/schedules")
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
