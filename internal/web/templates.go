package web

import (
	"embed"
	"html/template"
	"log"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

var Tmpl *template.Template

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func InitTemplates() {
	funcMap := template.FuncMap{
		"formatTime": formatTime,
	}
	var err error
	Tmpl, err = template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Ошибка парсинга шаблонов: %v", err)
	}
	log.Println("Шаблоны успешно загружены.")
}
