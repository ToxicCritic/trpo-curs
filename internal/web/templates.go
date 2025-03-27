package web

import (
	"embed"
	"html/template"
	"log"
)

//go:embed templates/*.html
var templatesFS embed.FS

var Tmpl *template.Template

func InitTemplates() {
	var err error
	Tmpl, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Ошибка парсинга шаблонов: %v", err)
	}
	log.Println("Шаблоны успешно загружены.")
}
