package web

import (
	"embed"
	"html/template"
	"log"
)

var templatesFS embed.FS

var Tmpl *template.Template

func InitTemplates() {
	var err error
	Tmpl, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Ошибка парсинга шаблонов: %v", err)
	}
}
