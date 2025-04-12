package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"scheduleApp/internal/models"

	"github.com/gin-gonic/gin"
)

func RenderAdminSchedulesPageWithFilters(c *gin.Context, db *sql.DB) {
	if gin.Mode() == gin.TestMode {
		c.String(http.StatusOK, "Mock admin_schedules page in test mode")
		return
	}

	allGroups, err := loadAllGroups(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Alarm": "Ошибка загрузки групп: " + err.Error(),
		})
		return
	}
	allTeachers, err := loadAllTeachers(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Alarm": "Ошибка загрузки преподавателей: " + err.Error(),
		})
		return
	}
	allClassrooms, err := loadAllClassrooms(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Alarm": "Ошибка загрузки аудиторий: " + err.Error(),
		})
		return
	}
	allSubjects, err := loadAllSubjects(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Alarm": "Ошибка загрузки предметов: " + err.Error(),
		})
		return
	}

	groupFilter := c.Query("group")
	teacherFilter := c.Query("teacher")
	classroomFilter := c.Query("classroom")

	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1
	if groupFilter != "" {
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

	query := `
		SELECT
			s.id,
			sub.name as subject_name,
			s.subject_id,
			t.name as teacher_name,
			s.teacher_id,
			c.room_number,
			s.classroom_id,
			s.start_time,
			s.end_time,
			s.created_at,
			COALESCE(string_agg(g.name, ', '), '') AS group_names,
			MIN(g.id) as group_id
		FROM schedule s
		JOIN subjects sub ON s.subject_id = sub.id
		JOIN teachers t ON s.teacher_id = t.id
		JOIN classrooms c ON s.classroom_id = c.id
		LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
		LEFT JOIN groups g ON sg.group_id = g.id
	`
	if len(whereClauses) > 0 {
		query += " WHERE " + joinClauses(whereClauses, " AND ")
	}
	query += `
		GROUP BY s.id, sub.name, s.subject_id, t.name, s.teacher_id, c.room_number, s.classroom_id, s.start_time, s.end_time, s.created_at
		ORDER BY s.start_time ASC;
	`

	rows, err := db.Query(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
			"Title": "Управление расписанием (Admin)",
			"Alarm": "Ошибка запроса: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	groupedSchedules := make(map[time.Time][]models.ScheduleDisplay)
	for rows.Next() {
		var sch models.ScheduleDisplay
		if err := rows.Scan(&sch.ID, &sch.SubjectName, &sch.SubjectID, &sch.TeacherName, &sch.TeacherID,
			&sch.RoomNumber, &sch.ClassroomID, &sch.StartTime, &sch.EndTime, &sch.CreatedAt, &sch.GroupNames, &sch.GroupID); err != nil {
			c.HTML(http.StatusInternalServerError, "schedules_admin", gin.H{
				"Title": "Управление расписанием (Admin)",
				"Alarm": "Ошибка сканирования строки: " + err.Error(),
			})
			return
		}
		dateOnly := time.Date(sch.StartTime.Year(), sch.StartTime.Month(), sch.StartTime.Day(), 0, 0, 0, 0, sch.StartTime.Location())
		groupedSchedules[dateOnly] = append(groupedSchedules[dateOnly], sch)
	}

	alarm, _ := c.Get("Alarm")
	c.HTML(http.StatusOK, "schedules_admin", gin.H{
		"Title":           "Управление расписанием (Admin)",
		"Schedules":       groupedSchedules,
		"AllGroups":       allGroups,
		"AllTeachers":     allTeachers,
		"AllClassrooms":   allClassrooms,
		"AllSubjects":     allSubjects,
		"GroupFilter":     groupFilter,
		"TeacherFilter":   teacherFilter,
		"ClassroomFilter": classroomFilter,
		"Alarm":           alarm,
	})
}

func DeleteScheduleHandler(c *gin.Context, db *sql.DB) {
	scheduleID := c.Param("id")
	_, err := db.Exec("DELETE FROM schedule WHERE id=$1", scheduleID)
	if err != nil {
		c.Set("Alarm", "Ошибка удаления записи: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	c.Set("Alarm", "Запись успешно удалена.")
	RenderAdminSchedulesPageWithFilters(c, db)
}

func RenderManageUserRolesPage(c *gin.Context, db *sql.DB) {
	userIdSearch := c.Query("user_id")

	var nonStudentQuery string
	var nonStudentArgs []interface{}
	if userIdSearch == "" {
		nonStudentQuery = `
			SELECT id, username, email, role 
			FROM users 
			WHERE role <> 'student'
			ORDER BY id
		`
	} else {
		nonStudentQuery = `
			SELECT id, username, email, role 
			FROM users 
			WHERE role <> 'student' AND id = $1
			ORDER BY id
		`
		nonStudentArgs = append(nonStudentArgs, userIdSearch)
	}
	nonStudentRows, err := db.Query(nonStudentQuery, nonStudentArgs...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": err.Error(),
		})
		return
	}
	defer nonStudentRows.Close()

	var admins []models.User
	var teachers []models.User
	for nonStudentRows.Next() {
		var u models.User
		if err := nonStudentRows.Scan(&u.ID, &u.Username, &u.Email, &u.Role); err != nil {
			c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
				"Title": "Управление пользователями",
				"Error": err.Error(),
			})
			return
		}
		if u.Role == "admin" {
			admins = append(admins, u)
		} else if u.Role == "teacher" {
			teachers = append(teachers, u)
		}
	}

	var studentQuery string
	var studentArgs []interface{}
	if userIdSearch == "" {
		studentQuery = `
			SELECT u.id, u.username, u.email, u.role, COALESCE(s.group_id, 0) as group_id 
			FROM users u
			LEFT JOIN students s ON u.id = s.user_id
			WHERE u.role = 'student'
			ORDER BY u.id
		`
	} else {
		studentQuery = `
			SELECT u.id, u.username, u.email, u.role, COALESCE(s.group_id, 0) as group_id 
			FROM users u
			LEFT JOIN students s ON u.id = s.user_id
			WHERE u.role = 'student' AND u.id = $1
			ORDER BY u.id
		`
		studentArgs = append(studentArgs, userIdSearch)
	}
	studentRows, err := db.Query(studentQuery, studentArgs...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": "Ошибка загрузки студентов: " + err.Error(),
		})
		return
	}
	defer studentRows.Close()

	var students []models.User
	for studentRows.Next() {
		var u models.User
		if err := studentRows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.GroupID); err != nil {
			c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
				"Title": "Управление пользователями",
				"Error": err.Error(),
			})
			return
		}
		students = append(students, u)
	}

	allGroups, err := loadAllGroups(db)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": "Ошибка загрузки групп: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "manage_users", gin.H{
		"Title":        "Управление пользователями",
		"Admins":       admins,
		"Teachers":     teachers,
		"Students":     students,
		"AllGroups":    allGroups,
		"UserIDSearch": userIdSearch,
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

func CheckScheduleCollision(db *sql.DB, teacherID, classroomID, groupID int, startTime, endTime time.Time, excludeID int) (bool, error) {
	var query string
	if excludeID > 0 {
		query = `
         SELECT COUNT(DISTINCT s.id)
         FROM schedule s
         LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
         WHERE s.id <> $1
           AND (
                s.teacher_id = $2 OR
                s.classroom_id = $3 OR
                sg.group_id = $4
           )
           AND EXTRACT(EPOCH FROM s.start_time) < $5
           AND EXTRACT(EPOCH FROM s.end_time) > $6
         `
		log.Printf("DEBUG: Collision query with excludeID: %s", query)
		var count int
		err := db.QueryRow(query, excludeID, teacherID, classroomID, groupID, endTime.Unix(), startTime.Unix()).Scan(&count)
		if err != nil {
			log.Printf("DEBUG: Error executing collision query: %v", err)
			return false, err
		}
		log.Printf("DEBUG: Collision check result: count=%d", count)
		return count > 0, nil
	} else {
		query = `
         SELECT COUNT(DISTINCT s.id)
         FROM schedule s
         LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
         WHERE (
                s.teacher_id = $1 OR
                s.classroom_id = $2 OR
                sg.group_id = $3
           )
           AND EXTRACT(EPOCH FROM s.start_time) < $4
           AND EXTRACT(EPOCH FROM s.end_time) > $5
         `
		log.Printf("DEBUG: Collision query without excludeID: %s", query)
		var count int
		err := db.QueryRow(query, teacherID, classroomID, groupID, endTime.Unix(), startTime.Unix()).Scan(&count)
		if err != nil {
			log.Printf("DEBUG: Error executing collision query: %v", err)
			return false, err
		}
		log.Printf("DEBUG: Collision check result: count=%d", count)
		return count > 0, nil
	}
}

func UpdateStudentGroupHandler(c *gin.Context, db *sql.DB) {
	userIDStr := c.Param("id")
	groupIDStr := c.PostForm("group_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": "Неверный ID пользователя",
		})
		return
	}

	var groupID int
	if groupIDStr != "" {
		groupID, err = strconv.Atoi(groupIDStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "manage_users", gin.H{
				"Title": "Управление пользователями",
				"Error": "Неверный ID группы",
			})
			return
		}
	}

	_, err = db.Exec(`UPDATE students SET group_id = $1 WHERE user_id = $2`, groupID, userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_users", gin.H{
			"Title": "Управление пользователями",
			"Error": "Ошибка обновления группы: " + err.Error(),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/users")
}
