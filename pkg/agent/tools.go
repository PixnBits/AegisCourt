package agent

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]string
	Execute(args map[string]interface{}) (string, error)
}

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web"
}

func (t *WebSearchTool) Parameters() map[string]string {
	return map[string]string{"query": "string"}
}

func (t *WebSearchTool) Execute(args map[string]interface{}) (string, error) {
	// Stub: mediated HTTP
	return "search results", nil
}

type Runtime struct {
	// stub
}
