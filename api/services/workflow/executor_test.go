package workflow

import (
	"context"
	"testing"
)

func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		workflow       *Workflow
		inputs         map[string]interface{}
		expectedStatus string
		expectedSteps  int
	}{
		{
			name: "simple workflow with start and end",
			workflow: &Workflow{
				ID:   "test-workflow",
				Name: "Test Workflow",
				Definition: WorkflowGraph{
					ID: "test-workflow",
					Nodes: []Node{
						{
							ID:   "start",
							Type: "start",
							Data: NodeData{
								Label: "Start",
							},
						},
						{
							ID:   "end",
							Type: "end",
							Data: NodeData{
								Label: "End",
							},
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
			inputs:         map[string]interface{}{},
			expectedStatus: "completed",
			expectedSteps:  2,
		},
		{
			name: "workflow with form node",
			workflow: &Workflow{
				ID:   "test-workflow",
				Name: "Test Workflow",
				Definition: WorkflowGraph{
					ID: "test-workflow",
					Nodes: []Node{
						{
							ID:   "start",
							Type: "start",
							Data: NodeData{
								Label: "Start",
							},
						},
						{
							ID:   "form",
							Type: "form",
							Data: NodeData{
								Label: "Form",
								Metadata: map[string]interface{}{
									"inputFields": []interface{}{"name", "email"},
								},
							},
						},
						{
							ID:   "end",
							Type: "end",
							Data: NodeData{
								Label: "End",
							},
						},
					},
					Edges: []Edge{
						{
							ID:     "e1",
							Source: "start",
							Target: "form",
						},
						{
							ID:     "e2",
							Source: "form",
							Target: "end",
						},
					},
				},
			},
			inputs: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expectedStatus: "completed",
			expectedSteps:  3,
		},
		{
			name: "workflow without start node",
			workflow: &Workflow{
				ID:   "test-workflow",
				Name: "Test Workflow",
				Definition: WorkflowGraph{
					ID: "test-workflow",
					Nodes: []Node{
						{
							ID:   "end",
							Type: "end",
							Data: NodeData{
								Label: "End",
							},
						},
					},
					Edges: []Edge{},
				},
			},
			inputs:         map[string]interface{}{},
			expectedStatus: "failed",
			expectedSteps:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			ctx := context.Background()

			result := executor.Execute(ctx, tt.workflow, tt.inputs)

			if result.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if len(result.Steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(result.Steps))
			}

			// Verify executedAt is set
			if result.ExecutedAt == "" {
				t.Error("Expected executedAt to be set")
			}
		})
	}
}

func TestExecutor_ProcessFormNode(t *testing.T) {
	tests := []struct {
		name        string
		node        *Node
		vars        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid form node with all required fields",
			node: &Node{
				ID:   "form",
				Type: "form",
				Data: NodeData{
					Label: "Form",
					Metadata: map[string]interface{}{
						"inputFields": []interface{}{"name", "email"},
					},
				},
			},
			vars: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expectError: false,
		},
		{
			name: "form node with missing required field",
			node: &Node{
				ID:   "form",
				Type: "form",
				Data: NodeData{
					Label: "Form",
					Metadata: map[string]interface{}{
						"inputFields": []interface{}{"name", "email"},
					},
				},
			},
			vars: map[string]interface{}{
				"name": "John Doe",
				// email is missing
			},
			expectError: true,
		},
		{
			name: "form node with invalid inputFields metadata",
			node: &Node{
				ID:   "form",
				Type: "form",
				Data: NodeData{
					Label: "Form",
					Metadata: map[string]interface{}{
						"inputFields": "invalid", // Should be []interface{}
					},
				},
			},
			vars: map[string]interface{}{
				"name": "John Doe",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			step := &ExecutionStep{}

			err := executor.processFormNode(tt.node, tt.vars, step)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if step.Output == nil {
					t.Error("Expected output to be set")
				}
			}
		})
	}
}

func TestExecutor_ProcessConditionNode(t *testing.T) {
	tests := []struct {
		name        string
		vars        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid condition with greater_than operator",
			vars: map[string]interface{}{
				"temperature": 30.0,
				"threshold":   25.0,
				"operator":    "greater_than",
			},
			expectError: false,
		},
		{
			name: "valid condition with less_than operator",
			vars: map[string]interface{}{
				"temperature": 20.0,
				"threshold":   25.0,
				"operator":    "less_than",
			},
			expectError: false,
		},
		{
			name: "missing temperature",
			vars: map[string]interface{}{
				"threshold": 25.0,
				"operator":  "greater_than",
			},
			expectError: true,
		},
		{
			name: "missing threshold",
			vars: map[string]interface{}{
				"temperature": 30.0,
				"operator":    "greater_than",
			},
			expectError: true,
		},
		{
			name: "int threshold",
			vars: map[string]interface{}{
				"temperature": 30.0,
				"threshold":   25, // int instead of float64
				"operator":    "greater_than",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			step := &ExecutionStep{}

			err := executor.processConditionNode(tt.vars, step)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if step.Output == nil {
					t.Error("Expected output to be set")
				}
			}
		})
	}
}

func TestExecutor_ProcessEmailNode(t *testing.T) {
	tests := []struct {
		name        string
		vars        map[string]interface{}
		expectError bool
	}{
		{
			name: "valid email node with all required variables",
			vars: map[string]interface{}{
				"email":        "john@example.com",
				"city":         "Sydney",
				"temperature":  30.0,
				"conditionMet": true,
			},
			expectError: false,
		},
		{
			name: "missing email variable",
			vars: map[string]interface{}{
				"city":         "Sydney",
				"temperature":  30.0,
				"conditionMet": true,
			},
			expectError: true,
		},
		{
			name: "missing city variable",
			vars: map[string]interface{}{
				"email":        "john@example.com",
				"temperature":  30.0,
				"conditionMet": true,
			},
			expectError: true,
		},
		{
			name: "missing temperature variable",
			vars: map[string]interface{}{
				"email":        "john@example.com",
				"city":         "Sydney",
				"conditionMet": true,
			},
			expectError: true,
		},
		{
			name: "condition not met - should not error",
			vars: map[string]interface{}{
				"email":        "john@example.com",
				"city":         "Sydney",
				"temperature":  30.0,
				"conditionMet": false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			step := &ExecutionStep{}

			err := executor.processEmailNode(tt.vars, step)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if step.Output == nil {
					t.Error("Expected output to be set")
				}
			}
		})
	}
}

func TestExecutor_GetCityCoordinates(t *testing.T) {
	tests := []struct {
		name     string
		city     string
		expected struct {
			lat float64
			lon float64
		}
	}{
		{
			name: "Sydney coordinates",
			city: "Sydney",
			expected: struct {
				lat float64
				lon float64
			}{
				lat: -33.8688,
				lon: 151.2093,
			},
		},
		{
			name: "Melbourne coordinates",
			city: "Melbourne",
			expected: struct {
				lat float64
				lon float64
			}{
				lat: -37.8136,
				lon: 144.9631,
			},
		},
		{
			name: "Unknown city",
			city: "UnknownCity",
			expected: struct {
				lat float64
				lon float64
			}{
				lat: 0,
				lon: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor()
			node := &Node{
				Data: NodeData{
					Metadata: map[string]interface{}{
						"options": []interface{}{
							map[string]interface{}{"city": "Sydney", "lat": -33.8688, "lon": 151.2093},
							map[string]interface{}{"city": "Melbourne", "lat": -37.8136, "lon": 144.9631},
						},
					},
				},
			}

			lat, lon := executor.getCityCoordinates(node, tt.city)

			if lat != tt.expected.lat {
				t.Errorf("Expected lat %f, got %f", tt.expected.lat, lat)
			}
			if lon != tt.expected.lon {
				t.Errorf("Expected lon %f, got %f", tt.expected.lon, lon)
			}
		})
	}
}

func TestFindNodeByType(t *testing.T) {
	nodes := []Node{
		{
			ID:   "start",
			Type: "start",
			Data: NodeData{Label: "Start"},
		},
		{
			ID:   "form",
			Type: "form",
			Data: NodeData{Label: "Form"},
		},
		{
			ID:   "end",
			Type: "end",
			Data: NodeData{Label: "End"},
		},
	}

	tests := []struct {
		name     string
		nodeType string
		expected *Node
	}{
		{
			name:     "find start node",
			nodeType: "start",
			expected: &nodes[0],
		},
		{
			name:     "find form node",
			nodeType: "form",
			expected: &nodes[1],
		},
		{
			name:     "find end node",
			nodeType: "end",
			expected: &nodes[2],
		},
		{
			name:     "node type not found",
			nodeType: "nonexistent",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findNodeByType(nodes, tt.nodeType)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Error("Expected node, got nil")
				} else if result.ID != tt.expected.ID {
					t.Errorf("Expected node ID %s, got %s", tt.expected.ID, result.ID)
				}
			}
		})
	}
}

func TestFindNextNodeID(t *testing.T) {
	edges := []Edge{
		{
			ID:     "e1",
			Source: "start",
			Target: "form",
		},
		{
			ID:     "e2",
			Source: "form",
			Target: "condition",
		},
		{
			ID:           "e3",
			Source:       "condition",
			Target:       "email",
			SourceHandle: "true",
		},
		{
			ID:           "e4",
			Source:       "condition",
			Target:       "end",
			SourceHandle: "false",
		},
	}

	tests := []struct {
		name         string
		currentNode  string
		vars         map[string]interface{}
		expectedNext string
	}{
		{
			name:         "simple edge from start to form",
			currentNode:  "start",
			vars:         map[string]interface{}{},
			expectedNext: "form",
		},
		{
			name:        "condition met - go to email",
			currentNode: "condition",
			vars: map[string]interface{}{
				"conditionMet": true,
			},
			expectedNext: "email",
		},
		{
			name:        "condition not met - go to end",
			currentNode: "condition",
			vars: map[string]interface{}{
				"conditionMet": false,
			},
			expectedNext: "end",
		},
		{
			name:         "no next node",
			currentNode:  "end",
			vars:         map[string]interface{}{},
			expectedNext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findNextNodeID(edges, tt.currentNode, tt.vars)

			if result != tt.expectedNext {
				t.Errorf("Expected next node %s, got %s", tt.expectedNext, result)
			}
		})
	}
}
