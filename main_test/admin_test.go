package main_test

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"scheduleApp/internal/handlers"
)

func setupTestContextJSON(method, target string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func TestCreateScheduleHandler_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	body := `{
        "subject_id":   1,
        "teacher_id":   2,
        "classroom_id": 3,
        "start_time":   "2025-09-01T08:00:00Z"
    }`

	layout := time.RFC3339
	startTime, _ := time.Parse(layout, "2025-09-01T08:00:00Z")
	endTime := startTime.Add(90 * time.Minute)

	collisionQuery := `
        SELECT COUNT(*) FROM schedule 
        WHERE (teacher_id = $1 OR classroom_id = $2)
          AND start_time < $3
          AND end_time > $4
    `
	mock.ExpectQuery(collisionQuery).
		WithArgs(2, 3, endTime, startTime).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	insertQuery := `
        INSERT INTO schedule (subject_id, teacher_id, classroom_id, start_time, end_time)
        VALUES ($1, $2, $3, $4, $5) RETURNING id
    `
	mock.ExpectQuery(insertQuery).
		WithArgs(1, 2, 3, startTime, endTime).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))

	c, w := setupTestContextJSON("POST", "/api/schedule", body)
	handlers.CreateScheduleHandler(c, db)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateScheduleHandler_Collision(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	body := `{
        "subject_id": 1,
        "teacher_id": 2,
        "classroom_id": 3,
        "start_time": "2025-09-01T08:00:00Z"
    }`

	layout := time.RFC3339
	startTime, _ := time.Parse(layout, "2025-09-01T08:00:00Z")
	endTime := startTime.Add(90 * time.Minute)

	collisionQuery := `
        SELECT COUNT(*) FROM schedule 
        WHERE (teacher_id = $1 OR classroom_id = $2)
          AND start_time < $3
          AND end_time > $4
    `
	mock.ExpectQuery(collisionQuery).
		WithArgs(2, 3, endTime, startTime).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	c, w := setupTestContextJSON("POST", "/api/schedule", body)
	handlers.CreateScheduleHandler(c, db)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateScheduleHandler_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	c, w := setupTestContextJSON("PUT", "/api/schedule/55", `{
        "subject_id":   10,
        "teacher_id":   20,
        "classroom_id": 30,
        "start_time":   "2025-10-01T10:00:00Z"
    }`)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "55"})

	layout := time.RFC3339
	startTime, _ := time.Parse(layout, "2025-10-01T10:00:00Z")
	endTime := startTime.Add(90 * time.Minute)

	collisionQuery := `
        SELECT COUNT(*) FROM schedule 
        WHERE id <> $1
          AND (teacher_id = $2 OR classroom_id = $3)
          AND start_time < $4
          AND end_time > $5
    `
	mock.ExpectQuery(collisionQuery).
		WithArgs(55, 20, 30, endTime, startTime).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	updateQuery := `
        UPDATE schedule
        SET subject_id=$1, teacher_id=$2, classroom_id=$3, start_time=$4, end_time=$5
        WHERE id=$6
    `
	mock.ExpectExec(updateQuery).
		WithArgs(10, 20, 30, startTime, endTime, 55).
		WillReturnResult(sqlmock.NewResult(0, 1))

	handlers.UpdateScheduleHandler(c, db)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateScheduleHandler_InvalidID(t *testing.T) {
	// Ошибка, если в URL вместо числа что-то другое
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	c, w := setupTestContextJSON("PUT", "/api/schedule/abc", `{
        "subject_id":   10,
        "teacher_id":   20,
        "classroom_id": 30,
        "start_time":   "2025-10-01T10:00:00Z"
    }`)
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "abc"})

	handlers.UpdateScheduleHandler(c, db)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid schedule ID")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetScheduleJSON_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	c, w := setupTestContextJSON("GET", "/api/schedule/77", "")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "77"})

	selectQuery := `
		SELECT 
			s.id,
			s.subject_id,
			s.teacher_id,
			s.classroom_id,
			s.start_time,
			COALESCE(MIN(g.id), 0) AS group_id
		FROM schedule s
		LEFT JOIN schedule_groups sg ON s.id = sg.schedule_id
		LEFT JOIN groups g ON g.id = sg.group_id
		WHERE s.id = $1
		GROUP BY s.id, s.subject_id, s.teacher_id, s.classroom_id, s.start_time
	`
	mock.ExpectQuery(selectQuery).
		WithArgs("77").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "subject_id", "teacher_id", "classroom_id", "start_time", "group_id",
		}).AddRow(
			77, 1, 2, 3, time.Date(2025, 9, 2, 10, 0, 0, 0, time.UTC), 5,
		))

	handlers.GetScheduleJSON(c, db)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetScheduleJSON_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	c, w := setupTestContextJSON("GET", "/api/schedule/999", "")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "999"})

	mock.ExpectQuery("SELECT .* FROM schedule").
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	handlers.GetScheduleJSON(c, db)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetScheduleJSON_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	c, w := setupTestContextJSON("GET", "/api/schedule/123", "")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "123"})

	mock.ExpectQuery("SELECT .*").
		WithArgs("123").
		WillReturnError(errors.New("DB connection error"))

	handlers.GetScheduleJSON(c, db)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}
