# Nahook Go SDK

Official Go SDK for [Nahook](https://nahook.com) — the webhook delivery platform.

## Installation

```bash
go get github.com/getnahook/nahook-go
```

Requires Go 1.21+.

## Quick Start

### Ingestion Client

Use the `client` package to send webhooks to your endpoints.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    nahook "github.com/getnahook/nahook-go"
    "github.com/getnahook/nahook-go/client"
)

func main() {
    c, err := client.New("nhk_us_your_api_key",
        client.WithTimeout(10*time.Second),
        client.WithRetries(3),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Send to a specific endpoint
    result, err := c.Send(ctx, "ep_abc123", nahook.SendOptions{
        Payload: map[string]interface{}{
            "event":   "order.created",
            "orderId": "ord_12345",
            "amount":  99.99,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Delivery: %s (status: %s)\n", result.DeliveryID, result.Status)

    // Fan-out by event type
    triggerResult, err := c.Trigger(ctx, "order.paid", nahook.TriggerOptions{
        Payload: map[string]interface{}{
            "orderId": "ord_12345",
            "amount":  99.99,
        },
        Metadata: map[string]string{
            "region": "us-east-1",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Triggered %d deliveries\n", len(triggerResult.DeliveryIDs))
}
```

### Batch Operations

```go
// Send to multiple endpoints at once (max 20)
batchResult, err := c.SendBatch(ctx, []nahook.SendBatchItem{
    {
        EndpointID: "ep_abc",
        Payload:    map[string]interface{}{"event": "user.created"},
    },
    {
        EndpointID: "ep_def",
        Payload:    map[string]interface{}{"event": "user.created"},
    },
})
if err != nil {
    log.Fatal(err)
}
for _, item := range batchResult.Items {
    if item.Error != nil {
        fmt.Printf("Item %d failed: %s\n", item.Index, item.Error.Message)
    } else {
        fmt.Printf("Item %d: delivery %s\n", item.Index, item.DeliveryID)
    }
}

// Fan-out multiple event types at once (max 20)
triggerBatchResult, err := c.TriggerBatch(ctx, []nahook.TriggerBatchItem{
    {
        EventType: "order.created",
        Payload:   map[string]interface{}{"orderId": "ord_1"},
    },
    {
        EventType: "invoice.paid",
        Payload:   map[string]interface{}{"invoiceId": "inv_1"},
    },
})
```

### Management Client

Use the `management` package to administer workspaces, endpoints, event types, applications, subscriptions, and portal sessions.

```go
package main

import (
    "context"
    "fmt"
    "log"

    nahook "github.com/getnahook/nahook-go"
    "github.com/getnahook/nahook-go/management"
)

func main() {
    mgmt, err := management.New("nhm_your_management_token",
        management.WithBaseURL("https://api.nahook.com"),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    workspaceID := "ws_abc123"

    // ── Endpoints ──

    // List all endpoints
    endpoints, err := mgmt.Endpoints.List(ctx, workspaceID)
    if err != nil {
        log.Fatal(err)
    }
    for _, ep := range endpoints.Data {
        fmt.Printf("Endpoint: %s -> %s (active: %v)\n", ep.ID, ep.URL, ep.IsActive)
    }

    // Create an endpoint
    newEp, err := mgmt.Endpoints.Create(ctx, workspaceID, nahook.CreateEndpointOptions{
        URL:         "https://example.com/webhooks",
        Description: "Production webhook receiver",
        Metadata:    map[string]string{"env": "production"},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created endpoint: %s\n", newEp.ID)

    // Get an endpoint
    ep, err := mgmt.Endpoints.Get(ctx, workspaceID, newEp.ID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Endpoint: %s\n", ep.URL)

    // Update an endpoint
    active := false
    updated, err := mgmt.Endpoints.Update(ctx, workspaceID, newEp.ID, nahook.UpdateEndpointOptions{
        IsActive: &active,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated endpoint active: %v\n", updated.IsActive)

    // Delete an endpoint
    err = mgmt.Endpoints.Delete(ctx, workspaceID, newEp.ID)
    if err != nil {
        log.Fatal(err)
    }

    // ── Event Types ──

    // List event types
    eventTypes, err := mgmt.EventTypes.List(ctx, workspaceID)
    if err != nil {
        log.Fatal(err)
    }
    for _, et := range eventTypes.Data {
        fmt.Printf("Event type: %s (%s)\n", et.Name, et.ID)
    }

    // Create an event type
    newET, err := mgmt.EventTypes.Create(ctx, workspaceID, nahook.CreateEventTypeOptions{
        Name:        "order.shipped",
        Description: "Fired when an order ships",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created event type: %s\n", newET.ID)

    // Get, update, delete follow the same pattern...

    // ── Applications ──

    // List with pagination
    limit := 10
    apps, err := mgmt.Applications.List(ctx, workspaceID, &nahook.ListOptions{
        Limit: &limit,
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, app := range apps.Data {
        fmt.Printf("App: %s (%s)\n", app.Name, app.ID)
    }

    // Create an application
    newApp, err := mgmt.Applications.Create(ctx, workspaceID, nahook.CreateApplicationOptions{
        Name:       "Acme Corp",
        ExternalID: "customer_123",
        Metadata:   map[string]string{"plan": "enterprise"},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created app: %s\n", newApp.ID)

    // List endpoints for an application
    appEndpoints, err := mgmt.Applications.ListEndpoints(ctx, workspaceID, newApp.ID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("App has %d endpoints\n", len(appEndpoints.Data))

    // Create endpoint under an application
    appEp, err := mgmt.Applications.CreateEndpoint(ctx, workspaceID, newApp.ID, nahook.CreateEndpointOptions{
        URL: "https://acme.com/webhooks",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created app endpoint: %s\n", appEp.ID)

    // ── Subscriptions ──

    // List subscriptions for an endpoint
    subs, err := mgmt.Subscriptions.List(ctx, workspaceID, appEp.ID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Endpoint has %d subscriptions\n", len(subs.Data))

    // Create a subscription
    sub, err := mgmt.Subscriptions.Create(ctx, workspaceID, appEp.ID, nahook.CreateSubscriptionOptions{
        EventTypeID: newET.ID,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created subscription: %s\n", sub.ID)

    // Delete a subscription
    err = mgmt.Subscriptions.Delete(ctx, workspaceID, appEp.ID, newET.ID)
    if err != nil {
        log.Fatal(err)
    }

    // ── Portal Sessions ──

    // Create a portal session for an application
    session, err := mgmt.PortalSessions.Create(ctx, workspaceID, newApp.ID, &nahook.CreatePortalSessionOptions{
        Metadata: map[string]string{"userId": "user_456"},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Portal URL: %s (expires: %s)\n", session.URL, session.ExpiresAt)

    // Create without options
    session2, err := mgmt.PortalSessions.Create(ctx, workspaceID, newApp.ID, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Portal URL: %s\n", session2.URL)

    // ── Environments ──

    // List environments
    envs, err := mgmt.Environments.List(ctx, workspaceID)
    if err != nil {
        log.Fatal(err)
    }
    for _, env := range envs.Data {
        fmt.Printf("Env: %s (%s, default: %v)\n", env.Name, env.Slug, env.IsDefault)
    }

    // Create an environment
    newEnv, err := mgmt.Environments.Create(ctx, workspaceID, nahook.CreateEnvironmentOptions{
        Name: "Staging",
        Slug: "staging",
    })

    // Update
    updatedName := "Pre-production"
    mgmt.Environments.Update(ctx, workspaceID, newEnv.ID, nahook.UpdateEnvironmentOptions{
        Name: &updatedName,
    })

    // Delete
    mgmt.Environments.Delete(ctx, workspaceID, newEnv.ID)

    // ── Event Type Visibility ──

    // List visibility for an environment
    vis, err := mgmt.Environments.ListEventTypeVisibility(ctx, workspaceID, newEnv.ID)

    // Set event type as published in an environment
    entry, err := mgmt.Environments.SetEventTypeVisibility(ctx, workspaceID, newEnv.ID, newET.ID, nahook.SetVisibilityOptions{
        Published: true,
    })
}
```

### Error Handling

```go
result, err := c.Send(ctx, "ep_123", nahook.SendOptions{
    Payload: map[string]interface{}{"test": true},
})
if err != nil {
    switch e := err.(type) {
    case *nahook.APIError:
        fmt.Printf("API error %d: %s (code: %s)\n", e.Status, e.Message, e.Code)

        if e.IsAuthError() {
            // 401 or 403 with token_disabled
            log.Fatal("Check your API key")
        }
        if e.IsNotFound() {
            // 404
            log.Fatal("Endpoint does not exist")
        }
        if e.IsRateLimited() {
            // 429 — RetryAfter may be set
            if e.RetryAfter != nil {
                fmt.Printf("Retry after %d seconds\n", *e.RetryAfter)
            }
        }
        if e.IsValidationError() {
            // 400
            fmt.Printf("Validation: %s\n", e.Message)
        }
        if e.IsRetryable() {
            // 5xx or 429 — consider retrying
        }

    case *nahook.NetworkError:
        fmt.Printf("Network error: %v\n", e.Cause)

    case *nahook.TimeoutError:
        fmt.Printf("Timed out after %dms\n", e.TimeoutMs)
    }
}
```

### Custom Idempotency Key

```go
// Provide your own idempotency key to deduplicate sends
result, err := c.Send(ctx, "ep_123", nahook.SendOptions{
    Payload:        map[string]interface{}{"orderId": "ord_123"},
    IdempotencyKey: "my-unique-key-abc",
})
// If not provided, a UUID v4 is generated automatically
```

## Configuration

### Client Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL(url)` | `https://api.nahook.com` | API base URL |
| `WithTimeout(d)` | 30s | HTTP request timeout |
| `WithRetries(n)` | 0 | Max retries for retryable errors (5xx, 429, network) |

### Management Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL(url)` | `https://api.nahook.com` | API base URL |
| `WithTimeout(d)` | 30s | HTTP request timeout |

The management client does not support retries.

### Retry Behavior

When retries are enabled (client only):
- Retryable errors: 5xx, 429, network errors, timeouts
- Non-retryable errors: 400, 401, 403, 404, 409, 413
- Backoff: exponential with full jitter, base 500ms, max 10s
- Formula: `min(10s, 500ms * 2^attempt) * rand()`
- Respects `Retry-After` header when present

## Auth

- **Client (ingestion)**: API keys starting with `nhk_`
- **Management**: Management tokens starting with `nhm_`

Both will return an error at construction time if the prefix is invalid.

## License

MIT
