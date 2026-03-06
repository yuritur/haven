---
name: go-senior
description: >
  Senior-level Go development skill for writing modern, idiomatic, production-ready Go code.
  Use this skill whenever writing, generating, reviewing, or advising on Go code — especially
  CLI tools. Triggers on: writing Go functions/structs/packages, designing CLI commands,
  asking for Go best practices, refactoring Go code, choosing between patterns, structuring
  a Go project, handling errors in Go, adding logging, config, or testing in Go. Even if the
  user just says "write me a Go function" or "how should I structure this in Go", invoke this skill.
---

# Senior Go Development

Write Go code the way experienced Go engineers do: simple, explicit, composable, and testable.
The goal is code a new team member can read in 5 minutes — not code that shows off cleverness.

## Core Philosophy

Go rewards simplicity. Before adding any abstraction, ask: "Does this exist because I need it today,
or because I imagine I might need it tomorrow?" Premature abstractions in Go create friction without value.

- **Explicit over implicit** — make data flow and dependencies visible
- **Flat over nested** — prefer early returns, minimize indentation
- **Concrete before abstract** — start with concrete types, extract interfaces when you have 2+ implementations or need to test
- **No globals** — pass dependencies explicitly; globals make code untestable and behavior unpredictable
- **Zero-value usability** — design structs so the zero value is useful or at least safe

## Project Structure

Follow standard Go layout for CLI tools:

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go          # thin: parse flags, wire deps, call run()
├── internal/
│   ├── cli/                 # command definitions (cobra commands, etc.)
│   ├── config/              # config loading and validation
│   └── <domain>/            # business logic, no CLI/infra concerns
├── go.mod
└── go.sum
```

`main.go` should be thin — its job is to wire dependencies and call a `run()` function that returns an error:

```go
func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

Keep business logic in `internal/` — it can't be imported by external packages, which is intentional.
Use `pkg/` only for code you genuinely intend to expose as a library.

## CLI Framework Selection

Choose based on complexity, not habit:

| Scenario | Framework |
|---|---|
| Simple tool, 1-3 flags, no subcommands | `flag` stdlib |
| Multiple subcommands, auto-generated help | `cobra` |
| Functional style preferred, no code generation | `urfave/cli` |
| Needs shell completion, man pages | `cobra` |

**Cobra pattern** — each command in its own file, dependencies injected via closure or struct:

```go
// internal/cli/deploy.go
func newDeployCmd(svc DeployService) *cobra.Command {
    var region string

    cmd := &cobra.Command{
        Use:   "deploy <model>",
        Short: "Deploy a model to AWS",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return svc.Deploy(cmd.Context(), args[0], region)
        },
    }

    cmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")
    return cmd
}
```

Notice `RunE` (returns error) over `Run` — always use `RunE`. Errors surface cleanly to `main`.

**Avoid cobra globals** — don't use `cobra.Command` at package level with `init()` registration.
Wire everything in `main.go` or a `NewRootCmd(deps...)` constructor.

## Dependency Injection

Pass dependencies explicitly. No `var db *sql.DB` at package level.

**Interface at the point of use** — define interfaces in the package that uses them, not where they're implemented:

```go
// internal/cli/deploy.go — consumer defines the interface
type DeployService interface {
    Deploy(ctx context.Context, model, region string) error
}
```

**Constructor injection** — wire once in `main.go`:

```go
func run() error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }

    awsClient := aws.NewClient(cfg.Region)
    deploySvc := deploy.NewService(awsClient)

    root := cli.NewRootCmd(deploySvc)
    return root.Execute()
}
```

Keep `NewFoo` constructors simple — they should not do I/O. Move I/O to an explicit `Init()` or lazy-init pattern if needed.

## Error Handling

Errors are values. Handle them where you have context to act on them.

**Wrap with context, don't repeat the word "error":**

```go
// Bad
if err != nil {
    return fmt.Errorf("error creating stack: %w", err)
}

// Good
if err != nil {
    return fmt.Errorf("create stack %s: %w", stackName, err)
}
```

Error messages form a chain: `deploy model: create stack haven-abc: describe stack: operation timed out`.
Each level adds one piece of context — the operation name and relevant identifier.

**Sentinel errors for known conditions callers check:**

```go
var ErrNotFound = errors.New("not found")

// caller
if errors.Is(err, state.ErrNotFound) {
    // handle
}
```

**Custom error types when callers need structured data:**

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}
```

**Don't ignore errors.** If you genuinely can't do anything (`defer f.Close()`), at least log at debug level.

## Context

Pass `context.Context` as the first argument to any function that does I/O or might be cancelled.
Never store context in a struct — pass it per-call.

```go
// Good
func (s *Service) Deploy(ctx context.Context, model string) error

// Bad
type Service struct {
    ctx context.Context
}
```

Respect cancellation — check `ctx.Err()` or pass ctx to downstream calls. Don't start work after ctx is done.

## Logging

Use `log/slog` (Go 1.21+). Structured logging, no global logger.

```go
// internal/cli/root.go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: logLevel, // from --verbose flag
}))

// pass to services
svc := deploy.NewService(awsClient, logger)
```

Log at the right level:
- `Debug` — developer diagnostics (API request details, intermediate state)
- `Info` — meaningful events (deployment started, stack created)
- `Warn` — recoverable issues worth noting
- `Error` — only when you're also returning an error or it's truly unrecoverable

Don't log and return an error — either log or return, not both (the caller will log or handle it).

## Configuration

Layered config: defaults < config file < environment variables < CLI flags. Flags win.

```go
type Config struct {
    Region  string
    Verbose bool
}

func Load() (Config, error) {
    cfg := Config{
        Region: "us-east-1", // default
    }

    if r := os.Getenv("HAVEN_REGION"); r != "" {
        cfg.Region = r
    }

    return cfg, nil
}
```

For complex config, use `viper` — but only when you need file-based config + env + flags. Don't pull
in viper for a tool with 3 flags.

Validate config early, at startup, before doing any work:

```go
func (c Config) Validate() error {
    if c.Region == "" {
        return fmt.Errorf("region is required")
    }
    return nil
}
```

## Testing

Table-driven tests, subtests, no test frameworks (standard `testing` package):

```go
func TestDeploy(t *testing.T) {
    tests := []struct {
        name    string
        model   string
        wantErr bool
    }{
        {"valid model", "llama3.2:1b", false},
        {"empty model", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := deploy.NewService(fakeAWS{})
            err := svc.Deploy(context.Background(), tt.model)
            if (err != nil) != tt.wantErr {
                t.Errorf("Deploy() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Design for testability** — if a function is hard to test, it's probably doing too much or has hidden dependencies.
Use interfaces to swap out I/O in tests. Avoid `os.Exit` in library code (only in `main`).

For CLI testing, test the service layer directly rather than exec-ing the binary.

## Common Pitfalls to Avoid

**Don't panic in library code** — return errors. Panics are for truly unrecoverable programmer errors
(nil pointer on a value that must never be nil).

**Don't use `init()`** — it runs implicitly, makes initialization order unpredictable, and is untestable.
Wire everything explicitly.

**Don't over-interface** — a single-implementation interface is usually a sign of premature abstraction.
Add the interface when you need the second implementation or need to mock for tests.

**Don't shadow `err`** — using `:=` in a new scope for `err` silently creates a new variable:

```go
// Bug: outer err is never set
if something {
    result, err := doThing()  // new err, shadows outer
    _ = result
}
return err  // always nil
```

**Don't use goroutines without thinking about their lifetime** — always have a clear answer to
"what stops this goroutine and when?"

**Avoid `interface{}` / `any` unless genuinely necessary** — use generics (Go 1.18+) for type-safe
collection utilities, or concrete types with clear contracts.

## CLI UX

Good CLI tools are predictable and composable:

- **Exit codes matter**: 0 = success, 1 = usage error, 2+ = operational error. Use `os.Exit` only in `main`.
- **Stderr for diagnostics, stdout for data**: logs and progress go to stderr; output that might be piped goes to stdout.
- **Meaningful help text**: every flag needs a description. `Use` and `Short` fields in cobra are not optional.
- **--verbose / --debug flags**: don't hardcode log levels.
- **Machine-readable output**: consider `--output json` for tools used in scripts.
- **Validate early**: check args and flags before doing any I/O. Fail fast with a clear message.

```go
cmd := &cobra.Command{
    Use:   "deploy <model>",
    Short: "Deploy a model to your AWS account",
    Long: `Provisions an EC2 instance, installs Ollama, pulls the model,
and returns an OpenAI-compatible endpoint.`,
    Example: `  haven deploy llama3.2:1b
  haven deploy phi3:mini --region eu-west-1`,
    Args: cobra.ExactArgs(1),
    RunE: ...,
}
```
