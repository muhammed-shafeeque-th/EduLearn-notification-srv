package ports

type TemplateRenderer interface {
	Render(templateName string, data map[string]string) (string, error)
}
