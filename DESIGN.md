# Workflow Automation Platform Backend - Design Document

## 1. Overview

For this assignment, I built the backend for a visual workflow automation platform. The goal was to enable users to define custom workflows—for example, triggering an alert when the temperature in a given city exceeds a specified threshold. It also includes frontend modifications to send the `WorkflowDefinition` in the request body for the `/execute` endpoint.

The existing frontend provided a drag-and-drop editor with stubbed API endpoints. My task was to replace these stubs with real backend functionality: persisting workflow definitions to a PostgreSQL database and executing workflows in memory using live weather data from an external API. However, on the frontend side 

This involved designing a flexible schema for storing node-based workflow graphs, implementing a simple execution engine that could handle multiple node types (form, weather, email), and integrating with a public weather API to fetch current conditions. The backend responds with a detailed execution summary and mock email payloads when conditions are met.

## 2. How I Structured the System

I decided to organize the code in layers, which makes it easier to understand and test. Here's how it works:

```
┌─────────────────┐
│   HTTP Layer    │  ← Handles web requests
├─────────────────┤
│  Service Layer  │  ← Coordinates everything
├─────────────────┤
│ Execution Layer │  ← Runs the workflows
├─────────────────┤
│ Repository Layer│  ← Talks to the database
└─────────────────┘
```

Each layer has one job and doesn't worry about what the other layers are doing. This makes the code much easier to work with - if I need to change how the database works, I only touch the repository layer.

## 3. Code Organisation

I followed Go conventions and organised the files like this:

```
api/
├── main.go                 # Starts the server
├── go.mod                  # Lists dependencies
├── pkg/
│   └── db/
│       ├── init.go         # Sets up the database
│       └── postgres.go     # Connects to PostgreSQL
└── services/
    └── workflow/
        ├── interfaces.go   # Defines what each part should do
        ├── service.go      # Main business logic
        ├── workflow.go     # Handles HTTP requests
        ├── executor.go     # Runs the workflows
        ├── repository.go   # Database operations
        └── types.go        # Data structures
```

I used Go's interface system to make the code more flexible. This means I can easily swap out parts (like using a different database) without changing the rest of the code. I also made sure to handle errors properly and use context for things like timeouts.

## 4. Breaking Down the Components

### HTTP Layer (workflow.go)
This is what handles incoming web requests. When someone calls the API, this layer:
- Checks that the request is valid
- Sends back the right HTTP status codes
- Makes sure the response format is correct

It's like a receptionist - it doesn't do the actual work, but it makes sure everything is properly organized before passing it along.

### Service Layer (service.go)
This is the coordinator. It takes requests from the HTTP layer and figures out what needs to happen. It talks to both the database (to get workflow definitions) and the executor (to run workflows). Think of it as the manager who makes sure everyone is working together.

### Execution Layer (executor.go)
This is where the magic happens. It actually runs the workflows by:
- Going through each step (node) in the workflow
- Calling external APIs (like getting weather data)
- Making decisions based on conditions
- Keeping track of what's happening

### Repository Layer (repository.go)
This handles all the database stuff. It saves and retrieves workflow definitions, converts between Go structs and JSON for storage, and manages database connections efficiently.

## 5. How I Store the Data

I decided to use PostgreSQL with a JSONB column for storing workflows. Here's what the database table looks like:

```sql
CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

The JSONB column stores the entire workflow definition (nodes, edges, and all their settings) as JSON. This gives me a lot of flexibility - I can add new types of nodes without changing the database structure.

For the Go code, I created structs that map directly to the JSON:

```go
type WorkflowGraph struct {
    ID    string  `json:"id"`
    Nodes []Node  `json:"nodes"`
    Edges []Edge  `json:"edges"`
}

type Node struct {
    ID   string   `json:"id"`
    Type string   `json:"type"` // start, form, weather, condition, email, end
    Data NodeData `json:"data"`
}
```

I chose JSONB because it's flexible and fast. PostgreSQL can index JSONB data, so queries are still efficient even though I'm storing complex structures.

## 6. Making It Scale

I thought about how this system would handle more users and bigger workflows:

**Horizontal Scaling**: The system is stateless, meaning each request is independent. This makes it easy to run multiple copies of the server behind a load balancer.

**Performance**: I made several optimizations:
- Workflows run entirely in memory (no database writes during execution)
- I reuse HTTP connections for external API calls
- Database connections are pooled
- I only load workflow definitions when needed

**Large Workflows**: For complex workflows, I process nodes one at a time to manage memory usage. If a condition isn't met, the system stops early instead of processing unnecessary steps.

## 7. Making It Extensible

I wanted to make it easy to add new types of nodes. Here's how the system works:

```go
func (e *Executor) processNode(node Node, inputs map[string]interface{}) error {
    switch node.Type {
    case "start":
        return e.processStartNode(node, inputs)
    case "form":
        return e.processFormNode(node, inputs)
    case "weather":
        return e.processWeatherNode(node, inputs)
    case "condition":
        return e.processConditionNode(node, inputs)
    case "email":
        return e.processEmailNode(node, inputs)
    case "end":
        return e.processEndNode(node, inputs)
    default:
        return fmt.Errorf("unknown node type: %s", node.Type)
    }
}
```

To add a new node type, I just:
1. Add a new case to this switch statement
2. Write the processing function
3. Add some tests

The weather API integration shows the pattern for external services - proper error handling, timeouts, and response parsing.

## 8. Testing Strategy

I wrote comprehensive tests because I wanted to make sure everything works correctly. The testing approach focuses on:

**Interface-based testing**: I created mock versions of the database and executor that behave predictably. This lets me test each part independently.

**Isolated components**: Each layer is tested separately, so if something breaks, I know exactly where the problem is.

**Edge cases**: I test error conditions, invalid inputs, and boundary cases to make sure the system handles them gracefully.

**Fast execution**: The tests use in-memory mocks, so they run quickly and don't depend on external services.

I organized the tests to mirror the code structure:

```
services/workflow/
├── workflow_test.go      # Tests HTTP handlers
├── executor_test.go      # Tests workflow execution
└── repository_test.go    # Tests database operations
```

The mocks are simple but effective:

```go
type MockRepository struct {
    workflows map[string]*Workflow
}

type MockExecutor struct {
    responses map[string]*ExecutionResponse
}
```

## 9. The Trade-offs I Made

Every design decision involves trade-offs. Here are the main ones I considered:

**In-Memory vs. Persistent Execution**
- **What I chose**: In-memory execution
- **Why**: It's faster and simpler to implement
- **Downside**: No history of what workflows have run
- **My reasoning**: For this assignment, speed and simplicity were more important than keeping execution history

**JSONB vs. Normalised Database Schema**
- **What I chose**: JSONB for storing workflow definitions
- **Why**: It's flexible and easy to extend
- **Downside**: Can't do complex queries on workflow structure
- **My reasoning**: Workflow definitions are mostly read once and executed, so flexibility is more valuable than complex querying

**Synchronous vs. Asynchronous Execution**
- **What I chose**: Synchronous execution
- **Why**: Simpler error handling and immediate feedback
- **Downside**: Can block if workflows take too long
- **My reasoning**: The workflows are simple enough that synchronous execution works well

## 10. What I Assumed

I made some assumptions about how the system would be used:

**Workflow Complexity**: I assumed workflows would be relatively simple (10-50 nodes). If someone needed hundreds of nodes, the architecture would need to change.

**Data Consistency**: I assumed eventual consistency was okay for workflow definitions. This let me use JSONB instead of more complex database structures.

## 11. Conclusion

I'm happy with how this system turned out. It successfully balances simplicity, performance, and extensibility. The layered architecture makes it easy to understand and modify, while the technology choices support both current needs and future growth.

The key strengths of this implementation are:
- **Clean separation of concerns** through the layered design
- **Easy extensibility** through the node type system
- **Good performance** through in-memory execution and optimized database queries
- **Comprehensive testing** that gives confidence in the code
- **Production-ready** error handling and logging

This provides a solid foundation that could be extended into a more complex workflow automation system while keeping the simplicity needed for the current use case. The design decisions prioritise long-term maintainability over short-term convenience, which should make the system easier to work with as requirements evolve.