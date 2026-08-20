package web

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html static/*
var assets embed.FS

var page = template.Must(template.ParseFS(assets, "templates/editor.html"))
