package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"scheduleApp/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func RenderStudentSchedule(c *gin.Context, db *sql.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Пользователь не авторизован",
		})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.HTML(http.StatusUnauthorized, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Ошибка преобразования ID пользователя",
		})
		return
	}

	teacherFilter := c.Query("teacher")
	subjectFilter := c.Query("subject")

	allTeachers, err := loadAllTeachers(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Ошибка загрузки преподавателей: " + err.Error(),
		})
		return
	}
	allSubjects, err := loadAllSubjects(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Ошибка загрузки предметов: " + err.Error(),
		})
		return
	}

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

	whereClause := `
		WHERE EXISTS (
			SELECT 1
			FROM students st
			JOIN schedule_groups sg ON sg.group_id = st.group_id
			WHERE st.user_id = $1 AND sg.schedule_id = s.id
		)
	`
	args := []interface{}{userID}
	argIndex := 2

	if teacherFilter != "" {
		whereClause += fmt.Sprintf(" AND s.teacher_id = $%d", argIndex)
		teacherID, err := strconv.Atoi(teacherFilter)
		if err != nil {
			c.HTML(http.StatusBadRequest, "schedules_user", gin.H{
				"Title": "Расписание",
				"Error": "Неверный формат фильтра преподавателя",
			})
			return
		}
		args = append(args, teacherID)
		argIndex++
	}

	if subjectFilter != "" {
		whereClause += fmt.Sprintf(" AND s.subject_id = $%d", argIndex)
		subjectID, err := strconv.Atoi(subjectFilter)
		if err != nil {
			c.HTML(http.StatusBadRequest, "schedules_user", gin.H{
				"Title": "Расписание",
				"Error": "Неверный формат фильтра предмета",
			})
			return
		}
		args = append(args, subjectID)
		argIndex++
	}

	groupByClause := `
		GROUP BY s.id, sub.name, s.subject_id, t.name, s.teacher_id, c.room_number, s.classroom_id, s.start_time, s.end_time, s.created_at
		ORDER BY s.start_time ASC
	`

	fullQuery := baseQuery + " " + whereClause + " " + groupByClause

	rows, err := db.Query(fullQuery, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_user", gin.H{
			"Title": "Расписание",
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
			c.HTML(http.StatusInternalServerError, "schedules_user", gin.H{
				"Title": "Расписание",
				"Error": err.Error(),
			})
			return
		}
		dayKey := time.Date(sch.StartTime.Year(), sch.StartTime.Month(), sch.StartTime.Day(), 0, 0, 0, 0, sch.StartTime.Location())
		groupedSchedules[dayKey] = append(groupedSchedules[dayKey], sch)
	}

	c.HTML(http.StatusOK, "schedules_user", gin.H{
		"Title":         "Расписание",
		"Schedules":     groupedSchedules,
		"AllTeachers":   allTeachers,
		"AllSubjects":   allSubjects,
		"TeacherFilter": teacherFilter,
		"SubjectFilter": subjectFilter,
	})
}

func RenderStudentComments(c *gin.Context, db *sql.DB) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.HTML(http.StatusUnauthorized, "student_comments", gin.H{
			"Title": "Комментарии преподавателей",
			"Error": "Пользователь не авторизован",
		})
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		c.HTML(http.StatusUnauthorized, "student_comments", gin.H{
			"Title": "Комментарии преподавателей",
			"Error": "Ошибка преобразования user_id",
		})
		return
	}

	var groupID int
	err := db.QueryRow("SELECT group_id FROM students WHERE user_id = $1", userID).Scan(&groupID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "student_comments", gin.H{
			"Title": "Комментарии преподавателей",
			"Error": "Ошибка получения группы студента: " + err.Error(),
		})
		return
	}

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
		JOIN schedule_groups sg ON s.id = sg.schedule_id
		JOIN subjects sub ON s.subject_id = sub.id
		JOIN teachers t ON s.teacher_id = t.id
		JOIN classrooms c ON s.classroom_id = c.id
		LEFT JOIN groups g ON sg.group_id = g.id
		WHERE sg.group_id = $1 AND s.end_time < NOW()
		GROUP BY s.id, sub.name, s.subject_id, t.name, s.teacher_id, c.room_number, s.classroom_id, s.start_time, s.end_time, s.created_at
		ORDER BY s.start_time ASC;
	`
	rows, err := db.Query(query, groupID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "student_comments", gin.H{
			"Title": "Комментарии преподавателей",
			"Error": "Ошибка запроса расписания: " + err.Error(),
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
			c.HTML(http.StatusInternalServerError, "student_comments", gin.H{
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
			c.HTML(http.StatusInternalServerError, "student_comments", gin.H{
				"Title": "Комментарии к занятиям",
				"Error": "Ошибка загрузки комментариев: " + err.Error(),
			})
			return
		}

		for commentRows.Next() {
			var comm models.Comment
			if err := commentRows.Scan(&comm.ID, &comm.ScheduleID, &comm.TeacherID, &comm.CommentText, &comm.FilePath, &comm.CreatedAt); err != nil {
				commentRows.Close()
				c.HTML(http.StatusInternalServerError, "student_comments", gin.H{
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

	c.HTML(http.StatusOK, "student_comments", gin.H{
		"Title":         "Комментарии к занятиям",
		"PastSchedules": groupedSchedules,
	})
}
