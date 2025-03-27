package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateScheduleHandler(c *gin.Context, db *sql.DB) {
	var body struct {
		SubjectID   int       `json:"subject_id"`
		TeacherID   int       `json:"teacher_id"`
		ClassroomID int       `json:"classroom_id"`
		StartTime   time.Time `json:"start_time"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	endTime := body.StartTime.Add(90 * time.Minute)

	var collisionCount int
	collisionQuery := `
        SELECT COUNT(*) FROM schedule 
        WHERE (teacher_id = $1 OR classroom_id = $2)
          AND start_time < $3
          AND end_time > $4
    `
	err := db.QueryRow(collisionQuery, body.TeacherID, body.ClassroomID, endTime, body.StartTime).Scan(&collisionCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки коллизий: " + err.Error()})
		return
	}
	if collisionCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Коллизия обнаружена: занятие пересекается с уже существующим."})
		return
	}

	var scheduleID int
	insertQuery := `
        INSERT INTO schedule (subject_id, teacher_id, classroom_id, start_time, end_time)
        VALUES ($1, $2, $3, $4, $5) RETURNING id
    `
	err = db.QueryRow(insertQuery, body.SubjectID, body.TeacherID, body.ClassroomID,
		body.StartTime, endTime).Scan(&scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании расписания: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Schedule created",
		"schedule_id": scheduleID,
	})
}

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
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	endTime := body.StartTime.Add(90 * time.Minute)

	var collisionCount int
	collisionQuery := `
        SELECT COUNT(*) FROM schedule 
        WHERE id <> $1
          AND (teacher_id = $2 OR classroom_id = $3)
          AND start_time < $4
          AND end_time > $5
    `
	err = db.QueryRow(collisionQuery, scheduleID, body.TeacherID, body.ClassroomID, endTime, body.StartTime).Scan(&collisionCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки коллизий: " + err.Error()})
		return
	}
	if collisionCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Коллизия обнаружена: занятие пересекается с уже существующим."})
		return
	}

	updateQuery := `
        UPDATE schedule
        SET subject_id=$1, teacher_id=$2, classroom_id=$3, start_time=$4, end_time=$5
        WHERE id=$6
    `
	_, err = db.Exec(updateQuery, body.SubjectID, body.TeacherID, body.ClassroomID,
		body.StartTime, endTime, scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления расписания: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated"})
}
