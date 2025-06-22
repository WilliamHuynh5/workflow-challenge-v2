package db

import (
	"context"
	"encoding/json"
	"log/slog"

	"workflow-code-test/api/services/workflow"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("Initializing database...")

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS workflows (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			definition JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		
		-- Create index on definition for better query performance
		CREATE INDEX IF NOT EXISTS idx_workflows_definition ON workflows USING GIN (definition);
		
		-- Create index on updated_at for sorting
		CREATE INDEX IF NOT EXISTS idx_workflows_updated_at ON workflows (updated_at DESC);
	`

	if _, err := pool.Exec(ctx, createTableSQL); err != nil {
		return err
	}

	slog.Info("✅ Database tables created successfully")

	if err := seedSampleWorkflow(ctx, pool); err != nil {
		return err
	}

	slog.Info("✅ Database initialisation completed")
	return nil
}

func seedSampleWorkflow(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("Seeding sample workflow...")

	workflowDef := workflow.WorkflowGraph{
		ID: "550e8400-e29b-41d4-a716-446655440000",
		Nodes: []workflow.Node{
			{
				ID:       "start",
				Type:     "start",
				Position: workflow.Position{X: -160, Y: 300},
				Data: workflow.NodeData{
					Label:       "Start",
					Description: "Begin weather check workflow",
					Metadata: map[string]interface{}{
						"hasHandles": map[string]interface{}{
							"source": true,
							"target": false,
						},
					},
				},
			},
			{
				ID:       "form",
				Type:     "form",
				Position: workflow.Position{X: 152, Y: 304},
				Data: workflow.NodeData{
					Label:       "User Input",
					Description: "Process collected data - name, email, location",
					Metadata: map[string]interface{}{
						"hasHandles": map[string]interface{}{
							"source": true,
							"target": true,
						},
						"inputFields":     []string{"name", "email", "city"},
						"outputVariables": []string{"name", "email", "city"},
					},
				},
			},
			{
				ID:       "weather-api",
				Type:     "integration",
				Position: workflow.Position{X: 460, Y: 304},
				Data: workflow.NodeData{
					Label:       "Weather API",
					Description: "Fetch current temperature for {{city}}",
					Metadata: map[string]interface{}{
						"hasHandles": map[string]interface{}{
							"source": true,
							"target": true,
						},
						"inputVariables": []string{"city"},
						"apiEndpoint":    "https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lon}&current_weather=true",
						"options": []map[string]interface{}{
							{"city": "Sydney", "lat": -33.8688, "lon": 151.2093},
							{"city": "Melbourne", "lat": -37.8136, "lon": 144.9631},
							{"city": "Brisbane", "lat": -27.4698, "lon": 153.0251},
							{"city": "Perth", "lat": -31.9505, "lon": 115.8605},
							{"city": "Adelaide", "lat": -34.9285, "lon": 138.6007},
						},
						"outputVariables": []string{"temperature"},
					},
				},
			},
			{
				ID:       "condition",
				Type:     "condition",
				Position: workflow.Position{X: 794, Y: 304},
				Data: workflow.NodeData{
					Label:       "Check Condition",
					Description: "Evaluate temperature threshold",
					Metadata: map[string]interface{}{
						"hasHandles": map[string]interface{}{
							"source": []string{"true", "false"},
							"target": true,
						},
						"conditionExpression": "temperature {{operator}} {{threshold}}",
						"outputVariables":     []string{"conditionMet"},
					},
				},
			},
			{
				ID:       "email",
				Type:     "email",
				Position: workflow.Position{X: 1096, Y: 88},
				Data: workflow.NodeData{
					Label:       "Send Alert",
					Description: "Email weather alert notification",
					Metadata: map[string]interface{}{
						"hasHandles": map[string]interface{}{
							"source": true,
							"target": true,
						},
						"inputVariables": []string{"name", "city", "temperature"},
						"emailTemplate": map[string]interface{}{
							"subject": "Weather Alert",
							"body":    "Weather alert for {{city}}! Temperature is {{temperature}}°C!",
						},
						"outputVariables": []string{"emailSent"},
					},
				},
			},
			{
				ID:       "end",
				Type:     "end",
				Position: workflow.Position{X: 1360, Y: 302},
				Data: workflow.NodeData{
					Label:       "Complete",
					Description: "Workflow execution finished",
					Metadata: map[string]interface{}{
						"hasHandles": map[string]interface{}{
							"source": false,
							"target": true,
						},
					},
				},
			},
		},
		Edges: []workflow.Edge{
			{
				ID: "e1", Source: "start", Target: "form", Type: "smoothstep", Animated: true,
				Style: map[string]interface{}{"stroke": "#10b981", "strokeWidth": 3},
				Label: "Initialize",
			},
			{
				ID: "e2", Source: "form", Target: "weather-api", Type: "smoothstep", Animated: true,
				Style: map[string]interface{}{"stroke": "#3b82f6", "strokeWidth": 3},
				Label: "Submit Data",
			},
			{
				ID: "e3", Source: "weather-api", Target: "condition", Type: "smoothstep", Animated: true,
				Style: map[string]interface{}{"stroke": "#f97316", "strokeWidth": 3},
				Label: "Temperature Data",
			},
			{
				ID: "e4", Source: "condition", Target: "email", Type: "smoothstep", Animated: true,
				SourceHandle: "true",
				Style:        map[string]interface{}{"stroke": "#10b981", "strokeWidth": 3},
				Label:        "✓ Condition Met",
				LabelStyle:   map[string]interface{}{"fill": "#10b981", "fontWeight": "bold"},
			},
			{
				ID: "e5", Source: "condition", Target: "end", Type: "smoothstep", Animated: true,
				SourceHandle: "false",
				Style:        map[string]interface{}{"stroke": "#6b7280", "strokeWidth": 3},
				Label:        "✗ No Alert Needed",
				LabelStyle:   map[string]interface{}{"fill": "#6b7280", "fontWeight": "bold"},
			},
			{
				ID: "e6", Source: "email", Target: "end", Type: "smoothstep", Animated: true,
				Style:      map[string]interface{}{"stroke": "#ef4444", "strokeWidth": 2},
				Label:      "Alert Sent",
				LabelStyle: map[string]interface{}{"fill": "#ef4444", "fontWeight": "bold"},
			},
		},
	}

	definitionJSON, err := json.Marshal(workflowDef)
	if err != nil {
		return err
	}

	query := `INSERT INTO workflows (id, name, definition) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING`
	_, err = pool.Exec(ctx, query, "550e8400-e29b-41d4-a716-446655440000", "Weather Alert Workflow", definitionJSON)
	if err != nil {
		return err
	}

	slog.Info("✅ Sample workflow seeded successfully")
	return nil
}
