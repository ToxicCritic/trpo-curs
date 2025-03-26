package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"scheduleApp/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Студент/преподаватель отправляет запрос на изменение расписания
func CreateRequestHandler(c *gin.Context, db *sql.DB) {
	userIDVal, _ := c.Get("user_id")
	roleVal, _ := c.Get("role")

	role, _ := roleVal.(string)
	userID, _ := userIDVal.(int)

	// teacher/student
	if role != "teacher" && role != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only teacher or student can create requests"})
		return
	}

	var body struct {
		ScheduleID    int    `json:"schedule_id"`
		DesiredChange string `json:"desired_change"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var requestID int
	query := `
        INSERT INTO requests (user_id, schedule_id, desired_change, status)
        VALUES ($1, $2, $3, 'pending')
        RETURNING id
    `
	err := db.QueryRow(query, userID, body.ScheduleID, body.DesiredChange).Scan(&requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Request created",
		"request_id": requestID,
	})
}

// GetAllRequestsHandler – администратор просматривает все запросы
func GetAllRequestsHandler(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, user_id, schedule_id, desired_change, status
        FROM requests
        ORDER BY id
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		var r models.Request
		if err := rows.Scan(&r.ID, &r.UserID, &r.ScheduleID, &r.DesiredChange, &r.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		requests = append(requests, r)
	}

	c.JSON(http.StatusOK, requests)
}

// Администратор рассматривает запрос
func ProcessRequestHandler(c *gin.Context, db *sql.DB) {
	requestID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body struct {
		Action string `json:"action"` // "approve" / "reject"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	status := "pending"
	if body.Action == "approve" {
		status = "approved"
		// TODO: Внесение изменений в расписание
	} else if body.Action == "reject" {
		status = "rejected"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action must be 'approve' or 'reject'"})
		return
	}

	_, err = db.Exec(`
        UPDATE requests
        SET status=$1
        WHERE id=$2
    `, status, requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	msg := fmt.Sprintf("Request %d processed. Status: %s", requestID, status)
	c.JSON(http.StatusOK, gin.H{"message": msg})
}
