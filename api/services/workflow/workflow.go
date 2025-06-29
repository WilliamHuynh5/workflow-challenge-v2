package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Service) HandleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	slog.Debug("Returning workflow definition for id", "id", id)

	ctx := r.Context()
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		slog.Error("Failed to get workflow", "id", id, "error", err)
		http.Error(w, fmt.Sprintf("Workflow not found: %s", err.Error()), http.StatusNotFound)
		return
	}

	// Map the workflow definition to a response that the frontend can use
	response := map[string]interface{}{
		"id":    workflow.Definition.ID,
		"nodes": workflow.Definition.Nodes,
		"edges": workflow.Definition.Edges,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Serialise the response to JSON
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode workflow response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Service) HandleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get the workflow id from the request
	id := mux.Vars(r)["id"]
	slog.Debug("Handling workflow execution for id", "id", id)

	// Read the request body (inputs, condition, workflow definition)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Unmarshal the request body into an ExecutionRequest
	var execReq ExecutionRequest
	if err := json.Unmarshal(body, &execReq); err != nil {
		slog.Error("Failed to parse execution request", "error", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Get the workflow from the repository
	ctx := r.Context()
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		slog.Error("Failed to get workflow for execution", "id", id, "error", err)
		http.Error(w, fmt.Sprintf("Workflow not found: %s", err.Error()), http.StatusNotFound)
		return
	}

	// If a workflow definition is provided, use it instead of the stored one
	if execReq.WorkflowDefinition != nil {
		slog.Debug("Using provided workflow definition for execution", "id", id)
		workflow.Definition = *execReq.WorkflowDefinition

		if err := s.repo.SaveWorkflow(ctx, workflow); err != nil {
			slog.Error("Failed to save updated workflow definition", "id", id, "error", err)
		} else {
			slog.Debug("Successfully saved updated workflow definition", "id", id)
		}
	} else {
		slog.Debug("Using stored workflow definition for execution", "id", id)
	}

	// Normalise the inputs, include the form data and the operator and threshold
	inputs := make(map[string]interface{})

	// Add the form data to the inputs
	for k, v := range execReq.FormData {
		inputs[k] = v
	}

	// Add the operator and threshold to the inputs
	if execReq.Condition != nil {
		if operator, ok := execReq.Condition["operator"].(string); ok {
			inputs["operator"] = operator
		}
		if threshold, ok := execReq.Condition["threshold"]; ok {
			inputs["threshold"] = threshold
		}
	}

	// Execute the workflow with the inputs
	executionResult := s.executor.Execute(ctx, workflow, inputs)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Serialise the execution result to JSON, and return it to the frontend
	if err := json.NewEncoder(w).Encode(executionResult); err != nil {
		slog.Error("Failed to encode execution response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
