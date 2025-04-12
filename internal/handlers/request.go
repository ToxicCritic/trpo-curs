package handlers

import (
	"database/sql"
	"net/http"
	"scheduleApp/internal/models"

	"github.com/gin-gonic/gin"
)

func CreateRequestHandler(c *gin.Context, db *sql.DB) {
	userIDVal, _ := c.Get("user_id")
	roleVal, _ := c.Get("role")

	role, _ := roleVal.(string)
	userID, _ := userIDVal.(int)

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

func RenderAdminRequestsPage(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, user_id, schedule_id, desired_change, status
        FROM requests
				WHERE status != 'rejected'
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
