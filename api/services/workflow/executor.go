package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Executor struct {
	httpClient *http.Client
}

func NewExecutor() *Executor {
	return &Executor{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (e *Executor) Execute(ctx context.Context, wf *Workflow, inputs map[string]interface{}) *ExecutionResponse {
	steps := []ExecutionStep{}
	vars := make(map[string]interface{})

	for k, v := range inputs {
		vars[k] = v
	}

	status := "completed"
	nodes := wf.Definition.Nodes
	nodeMap := make(map[string]*Node)

	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	current := findNodeByType(nodes, "start")
	if current == nil {
		return &ExecutionResponse{
			ExecutedAt: time.Now().Format(time.RFC3339),
			Status:     "failed",
			Steps: []ExecutionStep{{
				NodeID: "system",
				Type:   "system",
				Label:  "System Error",
				Status: "failed",
				Error:  "No start node found in workflow",
			}},
		}
	}

	for current != nil {
		step := ExecutionStep{
			NodeID:      current.ID,
			Type:        current.Type,
			Label:       current.Data.Label,
			Description: current.Data.Description,
			Status:      "completed",
		}

		switch current.Type {
		case "start":

		case "form":
			if err := e.processFormNode(current, vars, &step); err != nil {
				step.Status = "failed"
				step.Error = err.Error()
				status = "failed"
			}

		case "integration":
			if err := e.processIntegrationNode(ctx, current, vars, &step); err != nil {
				step.Status = "failed"
				step.Error = err.Error()
				status = "failed"
			}

		case "condition":
			if err := e.processConditionNode(vars, &step); err != nil {
				step.Status = "failed"
				step.Error = err.Error()
				status = "failed"
			}

		case "email":
			if err := e.processEmailNode(vars, &step); err != nil {
				step.Status = "failed"
				step.Error = err.Error()
				status = "failed"
			}

		case "end":

		default:
			step.Status = "failed"
			step.Error = fmt.Sprintf("Unknown node type: %s", current.Type)
			status = "failed"
		}

		steps = append(steps, step)

		if step.Status == "failed" {
			return &ExecutionResponse{
				ExecutedAt: time.Now().Format(time.RFC3339),
				Status:     status,
				Steps:      steps,
			}
		}

		nextID := findNextNodeID(wf.Definition.Edges, current.ID, vars)
		if nextID == "" {
			break
		}

		nextNode, exists := nodeMap[nextID]
		if !exists {
			break
		}
		current = nextNode
	}

	return &ExecutionResponse{
		ExecutedAt: time.Now().Format(time.RFC3339),
		Status:     status,
		Steps:      steps,
	}
}

func (e *Executor) processFormNode(node *Node, vars map[string]interface{}, step *ExecutionStep) error {
	metadata := node.Data.Metadata
	inputFields, ok := metadata["inputFields"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid inputFields in form node metadata")
	}

	output := make(map[string]interface{})
	for _, field := range inputFields {
		fieldName, ok := field.(string)
		if !ok {
			continue
		}

		if value, exists := vars[fieldName]; exists {
			output[fieldName] = value
		} else {
			return fmt.Errorf("missing required input field: %s", fieldName)
		}
	}

	step.Output = output
	return nil
}

func (e *Executor) processIntegrationNode(ctx context.Context, node *Node, vars map[string]interface{}, step *ExecutionStep) error {
	city, ok := vars["city"].(string)
	if !ok {
		return fmt.Errorf("city not found in variables")
	}

	lat, lon := e.getCityCoordinates(node, city)
	if lat == 0 && lon == 0 {
		return fmt.Errorf("coordinates not found for city: %s", city)
	}

	temperature, err := e.fetchWeather(ctx, lat, lon)
	if err != nil {
		return fmt.Errorf("failed to fetch weather data: %w", err)
	}

	vars["temperature"] = temperature

	step.Output = map[string]interface{}{
		"temperature": temperature,
		"location":    city,
	}

	return nil
}

func (e *Executor) processConditionNode(vars map[string]interface{}, step *ExecutionStep) error {
	temperature, ok := vars["temperature"].(float64)
	if !ok {
		return fmt.Errorf("temperature not found in variables")
	}

	threshold, ok := vars["threshold"].(float64)
	if !ok {
		if thresholdInt, ok := vars["threshold"].(int); ok {
			threshold = float64(thresholdInt)
		} else {
			return fmt.Errorf("threshold not found in variables")
		}
	}

	operator, ok := vars["operator"].(string)
	if !ok {
		operator = "greater_than" // default
	}

	var conditionMet bool
	switch operator {
	case "greater_than":
		conditionMet = temperature > threshold
	case "less_than":
		conditionMet = temperature < threshold
	case "equals":
		conditionMet = temperature == threshold
	case "greater_than_or_equal":
		conditionMet = temperature >= threshold
	case "less_than_or_equal":
		conditionMet = temperature <= threshold
	default:
		conditionMet = temperature > threshold
	}

	// Store result in variables
	vars["conditionMet"] = conditionMet

	step.Output = map[string]interface{}{
		"conditionMet": conditionMet,
		"threshold":    threshold,
		"operator":     operator,
		"actualValue":  temperature,
		"message":      fmt.Sprintf("Temperature %.1f°C %s %.1f°C - condition %s", temperature, operator, threshold, map[bool]string{true: "met", false: "not met"}[conditionMet]),
	}

	return nil
}

func (e *Executor) processEmailNode(vars map[string]interface{}, step *ExecutionStep) error {
	conditionMet, ok := vars["conditionMet"].(bool)
	if !ok || !conditionMet {
		step.Output = map[string]interface{}{
			"emailSent": false,
			"message":   "Condition not met, no email sent",
		}
		return nil
	}

	city, ok := vars["city"].(string)
	if !ok {
		return fmt.Errorf("city not found in variables")
	}

	temperature, ok := vars["temperature"].(float64)
	if !ok {
		return fmt.Errorf("temperature not found in variables")
	}

	email, ok := vars["email"].(string)
	if !ok {
		return fmt.Errorf("email not found in variables")
	}

	emailDraft := map[string]interface{}{
		"to":        email,
		"from":      "weather-alerts@example.com",
		"subject":   "Weather Alert",
		"body":      fmt.Sprintf("Weather alert for %s! Temperature is %.1f°C!", city, temperature),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	step.Output = map[string]interface{}{
		"emailDraft":     emailDraft,
		"deliveryStatus": "sent",
		"messageId":      fmt.Sprintf("msg_%s", time.Now().Format("20060102150405")),
		"emailSent":      true,
	}

	return nil
}

func (e *Executor) getCityCoordinates(node *Node, city string) (float64, float64) {
	metadata := node.Data.Metadata
	options, ok := metadata["options"].([]interface{})
	if !ok {
		return 0, 0
	}

	for _, option := range options {
		if cityData, ok := option.(map[string]interface{}); ok {
			if cityName, ok := cityData["city"].(string); ok && cityName == city {
				if lat, ok := cityData["lat"].(float64); ok {
					if lon, ok := cityData["lon"].(float64); ok {
						return lat, lon
					}
				}
			}
		}
	}

	return 0, 0
}

func (e *Executor) fetchWeather(ctx context.Context, lat, lon float64) (float64, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true", lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("weather API error: %s - %s", resp.Status, string(body))
	}

	var weatherResp WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		return 0, err
	}

	return weatherResp.CurrentWeather.Temperature, nil
}

func findNodeByType(nodes []Node, nodeType string) *Node {
	for i := range nodes {
		if nodes[i].Type == nodeType {
			return &nodes[i]
		}
	}
	return nil
}

func findNextNodeID(edges []Edge, currentNodeID string, vars map[string]interface{}) string {
	for _, edge := range edges {
		if edge.Source == currentNodeID {
			if edge.SourceHandle != "" {
				conditionMet, ok := vars["conditionMet"].(bool)
				if ok {
					if (edge.SourceHandle == "true" && conditionMet) ||
						(edge.SourceHandle == "false" && !conditionMet) {
						return edge.Target
					}
				}
			} else {
				return edge.Target
			}
		}
	}
	return ""
}
