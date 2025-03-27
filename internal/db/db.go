package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "schedule_user")
	password := getEnv("DB_PASSWORD", "schedule_pass")
	dbName := getEnv("DB_NAME", "schedule_db")

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName,
	)

	dbConn, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия подключения: %w", err)
	}

	if err = dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка ping к БД: %w", err)
	}

	log.Println("Успешное подключение к PostgreSQL")
	return dbConn, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func CreateTables(dbConn *sql.DB) error {
	queries := []string{
		`
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            username VARCHAR(255) NOT NULL UNIQUE,
            password VARCHAR(255) NOT NULL,
            email VARCHAR(255) UNIQUE,
            role VARCHAR(50) NOT NULL CHECK (role IN ('admin','teacher','student')),
            created_at TIMESTAMP DEFAULT NOW()
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS groups (
            id SERIAL PRIMARY KEY,
            name VARCHAR(50) NOT NULL,
            course VARCHAR(50)
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS teachers (
            id SERIAL PRIMARY KEY,
            user_id INT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
            name VARCHAR(255) NOT NULL,
            department VARCHAR(255)
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS students (
            id SERIAL PRIMARY KEY,
            user_id INT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
            name VARCHAR(255) NOT NULL,
            group_id INT REFERENCES groups(id)
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS subjects (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            description TEXT
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS classrooms (
            id SERIAL PRIMARY KEY,
            room_number VARCHAR(50) NOT NULL,
            building VARCHAR(255),
            capacity INTEGER
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS schedule (
            id SERIAL PRIMARY KEY,
            subject_id INT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
            teacher_id INT NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
            classroom_id INT NOT NULL REFERENCES classrooms(id) ON DELETE CASCADE,
            start_time TIMESTAMP NOT NULL,
            end_time TIMESTAMP NOT NULL,
            created_at TIMESTAMP DEFAULT NOW()
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS schedule_groups (
            schedule_id INT NOT NULL REFERENCES schedule(id) ON DELETE CASCADE,
            group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
            PRIMARY KEY (schedule_id, group_id)
        );
        `,

		`
        CREATE TABLE IF NOT EXISTS requests (
            id SERIAL PRIMARY KEY,
            user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            schedule_id INT REFERENCES schedule(id) ON DELETE CASCADE,
            desired_change TEXT,
            status VARCHAR(50) NOT NULL DEFAULT 'pending'
        );
        `,
	}

	for _, q := range queries {
		if _, err := dbConn.Exec(q); err != nil {
			return fmt.Errorf("ошибка при выполнении запроса:\n%v\n%w", q, err)
		}
	}
	return nil
}
