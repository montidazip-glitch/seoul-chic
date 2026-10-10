package handlers

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
)

// TemplateRenderer defines the interface for rendering templates.
type TemplateRenderer interface {
	ExecuteTemplate(w io.Writer, name string, data interface{}) error
}

// TemplateEngine manages separate template hierarchies to avoid {{define "content"}} collisions.
type TemplateEngine struct {
	templates map[string]*template.Template
}

// NewTemplateEngine parses layout.html with each individual page template.
func NewTemplateEngine(layoutPath string, funcMap template.FuncMap, pageDirs ...string) (*TemplateEngine, error) {
	engine := &TemplateEngine{
		templates: make(map[string]*template.Template),
	}

	for _, dir := range pageDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".html" {
				pagePath := filepath.Join(dir, entry.Name())
				tmpl, err := template.New(entry.Name()).Funcs(funcMap).ParseFiles(layoutPath, pagePath)
				if err != nil {
					return nil, fmt.Errorf("failed to parse template %s: %w", pagePath, err)
				}
				engine.templates[entry.Name()] = tmpl
			}
		}
	}
	return engine, nil
}

// ExecuteTemplate renders the requested page within the base layout.
func (te *TemplateEngine) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	tmpl, ok := te.templates[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return tmpl.ExecuteTemplate(w, "layout.html", data)
}
