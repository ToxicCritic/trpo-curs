package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"scheduleApp/internal/models"

	"github.com/gin-gonic/gin"
)

func RenderIndexTeacher(c *gin.Context, db *sql.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "index_teacher", gin.H{
			"Title": "Главная (Преподаватель)",
			"Error": "Пользователь не авторизован",
		})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.HTML(http.StatusUnauthorized, "index_teacher", gin.H{
			"Title": "Главная (Преподаватель)",
			"Error": "Ошибка преобразования user_id",
		})
		return
	}

	var teacherName string
	var departmentID int
	err := db.QueryRow(`SELECT name, department_id FROM teachers WHERE user_id = $1`, userID).Scan(&teacherName, &departmentID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "index_teacher", gin.H{
			"Title": "Главная (Преподаватель)",
			"Error": "Ошибка получения данных преподавателя: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "index_teacher", gin.H{
		"Title":       "Главная (Преподаватель)",
		"TeacherName": teacherName,
		"Message":     "Добро пожаловать, " + teacherName + "!",
	})
}

// RenderTeacherSchedule выводит расписание занятий для текущего учителя.
func RenderTeacherSchedule(c *gin.Context, db *sql.DB) {
	// Получаем user_id из контекста, установленного миддлварой.
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Ошибка авторизации",
		})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.HTML(http.StatusUnauthorized, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Ошибка преобразования user_id",
		})
		return
	}

	// Получаем teacher_id из таблицы teachers, связывающей user_id с преподавателем.
	var teacherID int
	err := db.QueryRow(`SELECT id FROM teachers WHERE user_id = $1`, userID).Scan(&teacherID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Учитель не найден: " + err.Error(),
		})
		return
	}

	// Формируем запрос для получения расписания для данного учителя.
	query := `
		SELECT 
			s.id,
			sub.name AS subject_name,
			s.subject_id,
			t.name AS teacher_name,
			s.teacher_id,
			c.room_number,
			s.classroom_id,
			s.start_time,
			s.end_time,
			s.created_at,
			COALESCE(string_agg(g.name, ', '), '') AS group_names,
			COALESCE(MIN(g.id), 0) AS group_id
		FROM schedule s
		JOIN subjects sub ON s.subject_id = sub.id
		JOIN teachers t ON s.teacher_id = t.id
		JOIN classrooms c ON s.classroom_id = c.id
		LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
		LEFT JOIN groups g ON sg.group_id = g.id
		WHERE s.teacher_id = $1
		AND s.start_time > NOW()
		GROUP BY s.id, sub.name, s.subject_id, t.name, s.teacher_id, c.room_number, s.classroom_id, s.start_time, s.end_time, s.created_at
		ORDER BY s.start_time ASC;
	`
	rows, err := db.Query(query, teacherID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	// Группируем занятия по дате (без учета времени)
	groupedSchedules := make(map[time.Time][]models.ScheduleDisplay)
	for rows.Next() {
		var sch models.ScheduleDisplay
		err := rows.Scan(&sch.ID, &sch.SubjectName, &sch.SubjectID, &sch.TeacherName, &sch.TeacherID,
			&sch.RoomNumber, &sch.ClassroomID, &sch.StartTime, &sch.EndTime, &sch.CreatedAt, &sch.GroupNames, &sch.GroupID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
				"Title": "Расписание учителя",
				"Error": err.Error(),
			})
			return
		}
		day := time.Date(sch.StartTime.Year(), sch.StartTime.Month(), sch.StartTime.Day(), 0, 0, 0, 0, sch.StartTime.Location())
		groupedSchedules[day] = append(groupedSchedules[day], sch)
	}

	c.HTML(http.StatusOK, "teacher_schedule", gin.H{
		"Title":     "Расписание учителя",
		"Schedules": groupedSchedules,
	})
}

func RenderTeacherComments(c *gin.Context, db *sql.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "teacher_comments", gin.H{
			"Title": "Комментарии к занятиям",
			"Error": "Пользователь не авторизован",
		})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.HTML(http.StatusUnauthorized, "teacher_comments", gin.H{
			"Title": "Комментарии к занятиям",
			"Error": "Ошибка преобразования user_id",
		})
		return
	}
	var teacherID int
	err := db.QueryRow(`SELECT id FROM teachers WHERE user_id = $1`, userID).Scan(&teacherID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
			"Title": "Комментарии к занятиям",
			"Error": "Учитель не найден: " + err.Error(),
		})
		return
	}

	// Выбираем прошедшие занятия для данного учителя (где end_time < NOW())
	query := `
		SELECT 
			s.id,
			sub.name AS subject_name,
			s.subject_id,
			t.name AS teacher_name,
			s.teacher_id,
			c.room_number,
			s.classroom_id,
			s.start_time,
			s.end_time,
			s.created_at,
			COALESCE(string_agg(g.name, ', '), '') AS group_names,
			COALESCE(MIN(g.id), 0) AS group_id
		FROM schedule s
		JOIN subjects sub ON s.subject_id = sub.id
		JOIN teachers t ON s.teacher_id = t.id
		JOIN classrooms c ON s.classroom_id = c.id
		LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
		LEFT JOIN groups g ON sg.group_id = g.id
		WHERE s.teacher_id = $1 AND s.end_time < NOW()
		GROUP BY s.id, sub.name, s.subject_id, t.name, s.teacher_id, c.room_number, s.classroom_id, s.start_time, s.end_time, s.created_at
		ORDER BY s.start_time ASC;
	`
	rows, err := db.Query(query, teacherID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
			"Title": "Комментарии к занятиям",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var pastSchedules []models.ScheduleDisplay
	for rows.Next() {
		var sch models.ScheduleDisplay
		err := rows.Scan(&sch.ID, &sch.SubjectName, &sch.SubjectID, &sch.TeacherName, &sch.TeacherID,
			&sch.RoomNumber, &sch.ClassroomID, &sch.StartTime, &sch.EndTime, &sch.CreatedAt,
			&sch.GroupNames, &sch.GroupID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
				"Title": "Комментарии к занятиям",
				"Error": err.Error(),
			})
			return
		}

		commentRows, err := db.Query(`
			SELECT id, schedule_id, teacher_id, comment_text, created_at
			FROM comments
			WHERE schedule_id = $1
			ORDER BY created_at ASC
		`, sch.ID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
				"Title": "Комментарии к занятиям",
				"Error": "Ошибка загрузки комментариев: " + err.Error(),
			})
			return
		}

		for commentRows.Next() {
			var comm models.Comment
			if err := commentRows.Scan(&comm.ID, &comm.ScheduleID, &comm.TeacherID, &comm.CommentText, &comm.CreatedAt); err != nil {
				commentRows.Close()
				c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
					"Title": "Комментарии к занятиям",
					"Error": "Ошибка сканирования комментария: " + err.Error(),
				})
				return
			}
			sch.Comments = append(sch.Comments, comm)
		}
		commentRows.Close()

		pastSchedules = append(pastSchedules, sch)
	}

	c.HTML(http.StatusOK, "teacher_comments", gin.H{
		"Title":         "Комментарии к занятиям",
		"PastSchedules": pastSchedules,
	})
}

func CreateTeacherComment(c *gin.Context, db *sql.DB) {
	scheduleIDStr := c.Param("id")
	scheduleID, err := strconv.Atoi(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID занятия"})
		return
	}

	commentText := c.PostForm("comment")
	if commentText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Комментарий не может быть пустым"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка преобразования user_id"})
		return
	}
	var teacherID int
	err = db.QueryRow(`SELECT id FROM teachers WHERE user_id = $1`, userID).Scan(&teacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Учитель не найден: " + err.Error()})
		return
	}

	_, err = db.Exec(`
		INSERT INTO comments (schedule_id, teacher_id, comment_text, created_at)
		VALUES ($1, $2, $3, NOW())
	`, scheduleID, teacherID, commentText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении комментария: " + err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, "/teacher/comments")
}

func RenderTeacherRequests(c *gin.Context, db *sql.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "teacher_requests", gin.H{
			"Title": "Запросы на изменения",
			"Error": "Пользователь не авторизован",
		})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.HTML(http.StatusUnauthorized, "teacher_requests", gin.H{
			"Title": "Запросы на изменения",
			"Error": "Ошибка преобразования user_id",
		})
		return
	}

	rows, err := db.Query(`
		SELECT id, user_id, schedule_id, desired_change, status
		FROM requests
		WHERE user_id = $1
		ORDER BY id DESC
	`, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_requests", gin.H{
			"Title": "Запросы на изменения",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		var r models.Request
		err := rows.Scan(&r.ID, &r.UserID, &r.ScheduleID, &r.DesiredChange, &r.Status)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "teacher_requests", gin.H{
				"Title": "Запросы на изменения",
				"Error": err.Error(),
			})
			return
		}
		requests = append(requests, r)
	}

	c.HTML(http.StatusOK, "teacher_requests", gin.H{
		"Title":    "Запросы на изменения расписания",
		"Requests": requests,
	})
}

// CreateTeacherRequest создаёт новый запрос на изменение расписания от учителя.
func CreateTeacherRequest(c *gin.Context, db *sql.DB) {
	// Извлекаем параметры из формы
	scheduleIDStr := c.PostForm("schedule_id")
	scheduleID, err := strconv.Atoi(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID занятия"})
		return
	}
	desiredChange := c.PostForm("desired_change")
	if desiredChange == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Запрос не может быть пустым"})
		return
	}

	// Получаем user_id из контекста
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не авторизован"})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ошибка преобразования user_id"})
		return
	}

	// Вставляем новый запрос (в таблице requests предполагается поле user_id)
	_, err = db.Exec(`
		INSERT INTO requests (user_id, schedule_id, desired_change, status)
		VALUES ($1, $2, $3, 'pending')
	`, userID, scheduleID, desiredChange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании запроса: " + err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, "/teacher/requests")
}
