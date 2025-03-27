package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Создать занятие (admin)
func CreateScheduleHandler(c *gin.Context, db *sql.DB) {
	var body struct {
		SubjectID   int       `json:"subject_id"`
		TeacherID   int       `json:"teacher_id"`
		ClassroomID int       `json:"classroom_id"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// TODO: Проверка коллизий
	// SELECT COUNT(*) FROM schedule WHERE teacher_id = $1 AND время

	var scheduleID int
	query := `
        INSERT INTO schedule (subject_id, teacher_id, classroom_id, start_time, end_time)
        VALUES ($1, $2, $3, $4, $5) RETURNING id
    `
	err := db.QueryRow(query, body.SubjectID, body.TeacherID, body.ClassroomID,
		body.StartTime, body.EndTime).Scan(&scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Schedule created",
		"schedule_id": scheduleID,
	})
}

// Отредактировать занятие (admin)
func UpdateScheduleHandler(c *gin.Context, db *sql.DB) {
	scheduleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var body struct {
		SubjectID   int       `json:"subject_id"`
		TeacherID   int       `json:"teacher_id"`
		ClassroomID int       `json:"classroom_id"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `
        UPDATE schedule
        SET subject_id=$1, teacher_id=$2, classroom_id=$3, start_time=$4, end_time=$5
        WHERE id=$6
    `
	_, err = db.Exec(query, body.SubjectID, body.TeacherID, body.ClassroomID,
		body.StartTime, body.EndTime, scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated"})
}

// Посмотреть расписание (все роли)
func GetSchedulesHandler(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT s.id, s.subject_id, s.teacher_id, s.classroom_id, s.start_time, s.end_time, s.created_at
        FROM schedule s
        ORDER BY s.start_time
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var sc models.Schedule
		err := rows.Scan(&sc.ID, &sc.SubjectID, &sc.TeacherID, &sc.ClassroomID,
			&sc.StartTime, &sc.EndTime, &sc.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		schedules = append(schedules, sc)
	}

	c.JSON(http.StatusOK, schedules)
}
