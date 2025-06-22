package workflow

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWorkflowSerialisation tests the JSON serialisation/deserialisation of workflows
func TestWorkflowSerialization(t *testing.T) {
	workflow := &Workflow{
		ID:        "test-workflow",
		Name:      "Test Workflow",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Definition: WorkflowGraph{
			ID: "test-workflow",
			Nodes: []Node{
				{
					ID:   "start",
					Type: "start",
					Position: Position{
						X: 100,
						Y: 200,
					},
					Data: NodeData{
						Label:       "Start",
						Description: "Start node",
						Metadata: map[string]interface{}{
							"hasHandles": map[string]interface{}{
								"source": true,
								"target": false,
							},
						},
					},
				},
				{
					ID:   "form",
					Type: "form",
					Position: Position{
						X: 300,
						Y: 200,
					},
					Data: NodeData{
						Label:       "Form",
						Description: "Form node",
						Metadata: map[string]interface{}{
							"inputFields": []interface{}{"name", "email"},
						},
					},
				},
			},
			Edges: []Edge{
				{
					ID:     "e1",
					Source: "start",
					Target: "form",
					Type:   "smoothstep",
					Style: map[string]interface{}{
						"stroke":      "#10b981",
						"strokeWidth": 3,
					},
				},
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(workflow)
	if err != nil {
		t.Fatalf("Failed to marshal workflow: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledWorkflow Workflow
	err = json.Unmarshal(jsonData, &unmarshaledWorkflow)
	if err != nil {
		t.Fatalf("Failed to unmarshal workflow: %v", err)
	}

	// Verify the data is preserved
	if unmarshaledWorkflow.ID != workflow.ID {
		t.Errorf("Expected ID %s, got %s", workflow.ID, unmarshaledWorkflow.ID)
	}

	if unmarshaledWorkflow.Name != workflow.Name {
		t.Errorf("Expected Name %s, got %s", workflow.Name, unmarshaledWorkflow.Name)
	}

	if len(unmarshaledWorkflow.Definition.Nodes) != len(workflow.Definition.Nodes) {
		t.Errorf("Expected %d nodes, got %d", len(workflow.Definition.Nodes), len(unmarshaledWorkflow.Definition.Nodes))
	}

	if len(unmarshaledWorkflow.Definition.Edges) != len(workflow.Definition.Edges) {
		t.Errorf("Expected %d edges, got %d", len(workflow.Definition.Edges), len(unmarshaledWorkflow.Definition.Edges))
	}
}

// TestWorkflowValidation tests basic workflow validation
func TestWorkflowValidation(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *Workflow
		expectValid bool
	}{
		{
			name: "valid workflow with start and end nodes",
			workflow: &Workflow{
				ID:   "valid-workflow",
				Name: "Valid Workflow",
				Definition: WorkflowGraph{
					ID: "valid-workflow",
					Nodes: []Node{
						{
							ID:   "start",
							Type: "start",
							Data: NodeData{Label: "Start"},
						},
						{
							ID:   "end",
							Type: "end",
							Data: NodeData{Label: "End"},
						},
					},
					Edges: []Edge{
						{
							ID:     "e1",
							Source: "start",
							Target: "end",
						},
					},
				},
			},
			expectValid: true,
		},
		{
			name: "workflow without start node",
			workflow: &Workflow{
				ID:   "invalid-workflow",
				Name: "Invalid Workflow",
				Definition: WorkflowGraph{
					ID: "invalid-workflow",
					Nodes: []Node{
						{
							ID:   "end",
							Type: "end",
							Data: NodeData{Label: "End"},
						},
					},
					Edges: []Edge{},
				},
			},
			expectValid: false,
		},
		{
			name: "workflow without end node",
			workflow: &Workflow{
				ID:   "invalid-workflow",
				Name: "Invalid Workflow",
				Definition: WorkflowGraph{
					ID: "invalid-workflow",
					Nodes: []Node{
						{
							ID:   "start",
							Type: "start",
							Data: NodeData{Label: "Start"},
						},
					},
					Edges: []Edge{},
				},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validateWorkflow(tt.workflow)
			if isValid != tt.expectValid {
				t.Errorf("Expected valid=%t, got %t", tt.expectValid, isValid)
			}
		})
	}
}

// Helper function to validate workflow structure
func validateWorkflow(workflow *Workflow) bool {
	if workflow == nil || workflow.Definition.Nodes == nil {
		return false
	}

	hasStart := false
	hasEnd := false

	for _, node := range workflow.Definition.Nodes {
		if node.Type == "start" {
			hasStart = true
		}
		if node.Type == "end" {
			hasEnd = true
		}
	}

	return hasStart && hasEnd
}
