package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

func RenderTeacherSchedule(c *gin.Context, db *sql.DB) {
	// Извлекаем user_id из контекста
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Пользователь не авторизован",
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

	// Получаем teacherID по user_id
	var teacherID int
	err := db.QueryRow(`SELECT id FROM teachers WHERE user_id = $1`, userID).Scan(&teacherID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Учитель не найден: " + err.Error(),
		})
		return
	}

	// Считываем фильтры: по группе и аудитории
	groupFilter := c.Query("group")
	classroomFilter := c.Query("classroom")

	// Загружаем списки для фильтрации (для формы)
	allGroups, err := loadAllGroups(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Ошибка загрузки групп: " + err.Error(),
		})
		return
	}
	allClassrooms, err := loadAllClassrooms(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": "Ошибка загрузки аудиторий: " + err.Error(),
		})
		return
	}

	// Базовый запрос для получения расписания для данного преподавателя (предстоящие занятия)
	baseQuery := `
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
	`
	// Основное условие: занятия назначены данному преподавателю и будущие (start_time > NOW())
	whereClause := `WHERE s.teacher_id = $1 AND s.start_time > NOW()`
	args := []interface{}{teacherID}
	argIndex := 2

	// Если задан фильтр по группе, добавляем условие через вложенный запрос
	if groupFilter != "" {
		groupID, err := strconv.Atoi(groupFilter)
		if err != nil {
			c.HTML(http.StatusBadRequest, "teacher_schedule", gin.H{
				"Title": "Расписание учителя",
				"Error": "Неверный формат фильтра по группе",
			})
			return
		}
		whereClause += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM schedule_groups sg2 WHERE sg2.schedule_id = s.id AND sg2.group_id = $%d)", argIndex)
		args = append(args, groupID)
		argIndex++
	}

	if classroomFilter != "" {
		classroomID, err := strconv.Atoi(classroomFilter)
		if err != nil {
			c.HTML(http.StatusBadRequest, "teacher_schedule", gin.H{
				"Title": "Расписание учителя",
				"Error": "Неверный формат фильтра по аудитории",
			})
			return
		}
		whereClause += fmt.Sprintf(" AND s.classroom_id = $%d", argIndex)
		args = append(args, classroomID)
		argIndex++
	}

	groupByClause := `
		GROUP BY s.id, sub.name, s.subject_id, t.name, s.teacher_id, c.room_number, s.classroom_id, s.start_time, s.end_time, s.created_at
		ORDER BY s.start_time ASC
	`

	fullQuery := baseQuery + " " + whereClause + " " + groupByClause

	rows, err := db.Query(fullQuery, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
			"Title": "Расписание учителя",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	groupedSchedules := make(map[time.Time][]models.ScheduleDisplay)
	for rows.Next() {
		var sch models.ScheduleDisplay
		err := rows.Scan(&sch.ID, &sch.SubjectName, &sch.SubjectID, &sch.TeacherName, &sch.TeacherID,
			&sch.RoomNumber, &sch.ClassroomID, &sch.StartTime, &sch.EndTime, &sch.CreatedAt,
			&sch.GroupNames, &sch.GroupID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "teacher_schedule", gin.H{
				"Title": "Расписание учителя",
				"Error": err.Error(),
			})
			return
		}
		dayKey := time.Date(sch.StartTime.Year(), sch.StartTime.Month(), sch.StartTime.Day(), 0, 0, 0, 0, sch.StartTime.Location())
		groupedSchedules[dayKey] = append(groupedSchedules[dayKey], sch)
	}

	c.HTML(http.StatusOK, "teacher_schedule", gin.H{
		"Title":           "Расписание учителя",
		"Schedules":       groupedSchedules,
		"AllGroups":       allGroups,
		"AllClassrooms":   allClassrooms,
		"GroupFilter":     groupFilter,
		"ClassroomFilter": classroomFilter,
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
	log.Printf("DEBUG: teacherID = %d", teacherID)

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
	log.Printf("DEBUG: Выполняем запрос расписания для teacherID = %d", teacherID)
	rows, err := db.Query(query, teacherID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
			"Title": "Комментарии к занятиям",
			"Error": err.Error(),
		})
		return
	}
	defer rows.Close()

	groupedSchedules := make(map[string][]models.ScheduleDisplay)
	for rows.Next() {
		var sch models.ScheduleDisplay
		err := rows.Scan(
			&sch.ID,
			&sch.SubjectName,
			&sch.SubjectID,
			&sch.TeacherName,
			&sch.TeacherID,
			&sch.RoomNumber,
			&sch.ClassroomID,
			&sch.StartTime,
			&sch.EndTime,
			&sch.CreatedAt,
			&sch.GroupNames,
			&sch.GroupID,
		)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "teacher_comments", gin.H{
				"Title": "Комментарии к занятиям",
				"Error": err.Error(),
			})
			return
		}
		log.Printf("DEBUG: Загружено занятие ID = %d, StartTime = %s", sch.ID, sch.StartTime)

		commentRows, err := db.Query(`
		SELECT 
			id, 
			schedule_id, 
			teacher_id, 
			comment_text, 
			COALESCE(file_path, '') as file_path,
			created_at
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
			if err := commentRows.Scan(&comm.ID, &comm.ScheduleID, &comm.TeacherID, &comm.CommentText, &comm.FilePath, &comm.CreatedAt); err != nil {
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
		log.Printf("DEBUG: Для занятия ID=%d загружено %d комментариев", sch.ID, len(sch.Comments))

		dateKey := sch.StartTime.Format("02.01.2006")
		groupedSchedules[dateKey] = append(groupedSchedules[dateKey], sch)
	}

	c.HTML(http.StatusOK, "teacher_comments", gin.H{
		"Title":         "Комментарии к занятиям",
		"PastSchedules": groupedSchedules,
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

	file, err := c.FormFile("attachment")
	var filePath string
	if err == nil {
		uploadDir := "uploads/comments" // директория для хранения файлов комментариев
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания директории: " + err.Error()})
			return
		}
		hash := sha256.New()
		hash.Write([]byte(file.Filename))
		hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		hashedPart := hex.EncodeToString(hash.Sum(nil))[:8]
		extension := filepath.Ext(file.Filename)
		filePath = filepath.Join(uploadDir, hashedPart+extension)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки файла: " + err.Error()})
			return
		}
	} else if err != http.ErrMissingFile {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обработки файла: " + err.Error()})
		return
	}

	_, err = db.Exec(`
		INSERT INTO comments (schedule_id, teacher_id, comment_text, file_path, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, scheduleID, teacherID, commentText, filePath)
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

func CreateTeacherRequest(c *gin.Context, db *sql.DB) {
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
