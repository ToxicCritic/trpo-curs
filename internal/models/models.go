package models

import "time"

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type Teacher struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	Name       string `json:"name"`
	Department string `json:"department"`
}

type Student struct {
	ID      int    `json:"id"`
	UserID  int    `json:"user_id"`
	Name    string `json:"name"`
	GroupID int    `json:"group_id"`
}

type Subject struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Classroom struct {
	ID         int    `json:"id"`
	RoomNumber string `json:"room_number"`
	Building   string `json:"building"`
	Capacity   int    `json:"capacity"`
}

type Schedule struct {
	ID          int       `json:"id"`
	SubjectID   int       `json:"subject_id"`
	TeacherID   int       `json:"teacher_id"`
	ClassroomID int       `json:"classroom_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
}

type Request struct {
	ID            int    `json:"id"`
	UserID        int    `json:"user_id"`
	ScheduleID    int    `json:"schedule_id"`
	DesiredChange string `json:"desired_change"`
	Status        string `json:"status"`
}

type ScheduleDisplay struct {
	ID          int       `json:"id"`
	SubjectID   int       `json:"subject_id"`
	TeacherID   int       `json:"teacher_id"`
	ClassroomID int       `json:"classroom_id"`
	GroupID     int       `json:"group_id"`
	GroupNames  string    `json:"group_names"`
	SubjectName string    `json:"subject_name"`
	TeacherName string    `json:"teacher_name"`
	RoomNumber  string    `json:"room_number"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedAt   time.Time `json:"created_at"`
}

type SubjectDisplay struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GroupDisplay struct {
	ID   int
	Name string
}

type TeacherDisplay struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ClassroomDisplay struct {
	ID         int    `json:"id"`
	RoomNumber string `json:"room_number"`
}

type RequestDisplay struct {
	ID            int    `json:"id"`
	UserID        int    `json:"user_id"`
	ScheduleID    int    `json:"schedule_id"`
	DesiredChange string `json:"desired_change"`
	Status        string `json:"status"`
}
