package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

func RenderSchedulesPage(c *gin.Context, db *sql.DB) {
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

	roleVal, exists := c.Get("role")
	if !exists {
		c.HTML(http.StatusUnauthorized, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Пользователь не авторизован",
		})
		return
	}
	role, ok := roleVal.(string)
	if !ok {
		c.HTML(http.StatusUnauthorized, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Ошибка преобразования роли пользователя",
		})
		return
	}

	// Базовый SQL-запрос, который выбирает расписание с JOIN'ами для получения связанных данных
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
	var whereClause string
	var args []interface{}

	switch role {
	case "admin":
		whereClause = ""
	case "teacher":
		whereClause = `WHERE s.teacher_id = (SELECT id FROM teachers WHERE user_id = $1)`
		args = append(args, userID)
	case "student":
		whereClause = `WHERE EXISTS (
                           SELECT 1 
                           FROM students st 
                           JOIN schedule_groups sg ON sg.group_id = st.group_id
                           WHERE st.user_id = $1 AND sg.schedule_id = s.id
                       )`
		args = append(args, userID)
	default:
		c.HTML(http.StatusUnauthorized, "schedules_user", gin.H{
			"Title": "Расписание",
			"Error": "Неверная роль пользователя",
		})
		return
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
		day := time.Date(sch.StartTime.Year(), sch.StartTime.Month(), sch.StartTime.Day(), 0, 0, 0, 0, sch.StartTime.Location())
		groupedSchedules[day] = append(groupedSchedules[day], sch)
	}

	c.HTML(http.StatusOK, "schedules_user", gin.H{
		"Title":     "Расписание",
		"Schedules": groupedSchedules,
	})
}
