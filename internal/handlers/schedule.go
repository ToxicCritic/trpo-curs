package handlers

import (
	"database/sql"
	"log"
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

func UpdateScheduleFormHandler(c *gin.Context, db *sql.DB) {
	scheduleID := c.Param("id")

	subjectID, _ := strconv.Atoi(c.PostForm("subject_id"))
	teacherID, _ := strconv.Atoi(c.PostForm("teacher_id"))
	classroomID, _ := strconv.Atoi(c.PostForm("classroom_id"))
	groupID, _ := strconv.Atoi(c.PostForm("group_id"))
	startTimeStr := c.PostForm("start_time")
	if startTimeStr == "" {
		c.Set("Alarm", "Поле времени начала не заполнено.")
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	layout := "2006-01-02T15:04"
	startTime, err := time.Parse(layout, startTimeStr)
	if err != nil {
		c.Set("Alarm", "Неверный формат времени начала: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	endTime := startTime.Add(90 * time.Minute)

	idInt, err := strconv.Atoi(scheduleID)
	if err != nil {
		c.Set("Alarm", "Неверный ID занятия")
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}

	collision, err := CheckScheduleCollision(db, teacherID, classroomID, groupID, startTime, endTime, idInt)
	if err != nil {
		c.Set("Alarm", "Ошибка проверки коллизий: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	if collision {
		c.Set("Alarm", "Коллизия обнаружена: у преподавателя, в аудитории или у группы уже существует пересекающееся занятие.")
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}

	_, err = db.Exec(`
        UPDATE schedule
        SET subject_id=$1, teacher_id=$2, classroom_id=$3, start_time=$4, end_time=$5
        WHERE id=$6
    `, subjectID, teacherID, classroomID, startTime, endTime, scheduleID)
	if err != nil {
		c.Set("Alarm", "Ошибка обновления расписания: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	c.Set("Alarm", "Расписание успешно обновлено.")
	RenderAdminSchedulesPageWithFilters(c, db)
}

func CreateScheduleFormHandler(c *gin.Context, db *sql.DB) {
	subjectID, err1 := strconv.Atoi(c.PostForm("subject_id"))
	teacherID, err2 := strconv.Atoi(c.PostForm("teacher_id"))
	classroomID, err3 := strconv.Atoi(c.PostForm("classroom_id"))
	groupID, err4 := strconv.Atoi(c.PostForm("group_id"))
	startTimeStr := c.PostForm("start_time")

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || startTimeStr == "" {
		c.Set("Alarm", "Неверные данные формы")
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}

	layout := "2006-01-02T15:04"
	startTime, err := time.Parse(layout, startTimeStr)
	if err != nil {
		c.Set("Alarm", "Неверный формат времени начала: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	endTime := startTime.Add(90 * time.Minute)

	collision, err := CheckScheduleCollision(db, teacherID, classroomID, groupID, startTime, endTime, 0)
	if err != nil {
		c.Set("Alarm", "Ошибка проверки коллизий: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	if collision {
		c.Set("Alarm", "Коллизия обнаружена: у преподавателя, в аудитории или у группы уже существует пересекающееся занятие.")
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}

	insertQuery := `
        INSERT INTO schedule (subject_id, teacher_id, classroom_id, start_time, end_time)
        VALUES ($1, $2, $3, $4, $5) RETURNING id
    `
	stmt, err := db.Prepare(insertQuery)
	if err != nil {
		log.Printf("ERROR: Не удалось подготовить запрос: %v", err)
		c.Set("Alarm", "Ошибка подготовки запроса: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}
	defer stmt.Close()

	var scheduleID int
	err = stmt.QueryRow(subjectID, teacherID, classroomID, startTime, endTime).Scan(&scheduleID)
	if err != nil {
		c.Set("Alarm", "Ошибка при создании записи: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}

	_, err = db.Exec(`INSERT INTO schedule_groups (schedule_id, group_id) VALUES ($1, $2)`, scheduleID, groupID)
	if err != nil {
		c.Set("Alarm", "Ошибка создания связи с группой: "+err.Error())
		RenderAdminSchedulesPageWithFilters(c, db)
		return
	}

	c.Set("Alarm", "Занятие успешно создано.")
	RenderAdminSchedulesPageWithFilters(c, db)
}

func GetScheduleJSON(c *gin.Context, db *sql.DB) {
	scheduleID := c.Param("id")

	query := `
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
	row := db.QueryRow(query, scheduleID)

	var obj struct {
		ID          int       `json:"id"`
		SubjectID   int       `json:"subject_id"`
		TeacherID   int       `json:"teacher_id"`
		ClassroomID int       `json:"classroom_id"`
		GroupID     int       `json:"group_id"`
		StartTime   time.Time `json:"start_time"`
	}
	err := row.Scan(&obj.ID, &obj.SubjectID, &obj.TeacherID, &obj.ClassroomID, &obj.StartTime, &obj.GroupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, obj)
}
