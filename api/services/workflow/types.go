package workflow

import "time"

type Workflow struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Definition WorkflowGraph `json:"definition"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type WorkflowGraph struct {
	ID    string `json:"id"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type NodeData struct {
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Edge struct {
	ID           string                 `json:"id"`
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	Type         string                 `json:"type"`
	Animated     bool                   `json:"animated"`
	Style        map[string]interface{} `json:"style"`
	Label        string                 `json:"label"`
	LabelStyle   map[string]interface{} `json:"labelStyle,omitempty"`
	SourceHandle string                 `json:"sourceHandle,omitempty"`
}

type ExecutionRequest struct {
	FormData           map[string]interface{} `json:"formData"`
	Condition          map[string]interface{} `json:"condition"`
	WorkflowDefinition *WorkflowGraph         `json:"workflowDefinition,omitempty"`
}

type ExecutionResponse struct {
	ExecutedAt string          `json:"executedAt"`
	Status     string          `json:"status"`
	Steps      []ExecutionStep `json:"steps"`
}

type ExecutionStep struct {
	NodeID      string                 `json:"nodeId"`
	Type        string                 `json:"type"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

type WeatherResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
	} `json:"current_weather"`
}

type CityCoordinates struct {
	City string  `json:"city"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}
