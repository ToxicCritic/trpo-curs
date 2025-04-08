package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/models"

	"github.com/gin-gonic/gin"
)

func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	out := clauses[0]
	for i := 1; i < len(clauses); i++ {
		out += sep + clauses[i]
	}
	return out
}

func loadAllSubjects(db *sql.DB) ([]models.SubjectDisplay, error) {
	rows, err := db.Query(`SELECT id, name FROM subjects ORDER BY name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []models.SubjectDisplay
	for rows.Next() {
		var s models.SubjectDisplay
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}
	return subjects, nil
}

func loadAllGroups(db *sql.DB) ([]models.GroupDisplay, error) {
	rows, err := db.Query(`SELECT id, name FROM groups ORDER BY name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.GroupDisplay
	for rows.Next() {
		var g models.GroupDisplay
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, nil
}

func loadAllTeachers(db *sql.DB) ([]models.TeacherDisplay, error) {
	rows, err := db.Query(`SELECT id, name FROM teachers ORDER BY name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.TeacherDisplay
	for rows.Next() {
		var t models.TeacherDisplay
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func loadAllClassrooms(db *sql.DB) ([]models.ClassroomDisplay, error) {
	rows, err := db.Query(`SELECT id, room_number FROM classrooms ORDER BY room_number;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ClassroomDisplay
	for rows.Next() {
		var c models.ClassroomDisplay
		if err := rows.Scan(&c.ID, &c.RoomNumber); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func loadAllDepartments(db *sql.DB) ([]models.DepartmentDisplay, error) {
	rows, err := db.Query(`SELECT id, name FROM departments ORDER BY name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var departments []models.DepartmentDisplay
	for rows.Next() {
		var d models.DepartmentDisplay
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		departments = append(departments, d)
	}
	return departments, nil
}

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

func RenderLoginPage(c *gin.Context) {
	alarm := c.Query("alarm")
	c.HTML(http.StatusOK, "login", gin.H{
		"Title": "Авторизация",
		"Alarm": alarm,
	})
}
