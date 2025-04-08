package web

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

var Tmpl *template.Template

var weekdayMap = map[time.Weekday]string{
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
	time.Sunday:    "Воскресенье",
}

func dayFullDate(t time.Time) string {
	dayName, ok := weekdayMap[t.Weekday()]
	if !ok {
		dayName = "Неизвестный день"
	}
	dateStr := t.Format("02.01.2006")
	return fmt.Sprintf("%s (%s)", dateStr, dayName)
}

func timeHHMM(t time.Time) string {
	return t.Format("15:04")
}

func InitTemplates() {
	var err error
	funcMap := template.FuncMap{
		"dayFullDate": dayFullDate,
		"timeHHMM":    timeHHMM,
	}
	Tmpl, err = template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Ошибка парсинга шаблонов: %v", err)
	}
	log.Println("Шаблоны успешно загружены.")
}
