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

    // The SDK automatically routes requests to the correct regional API
    // based on your API key prefix (nhk_us_... → US, nhk_eu_... → EU,
    // nhk_ap_... → Asia Pacific). No configuration needed.
    //
    // To override for testing/local dev:
    //   c, err := client.New("nhk_us_...", client.WithBaseURL("http://localhost:3001"))
    //
    // For unit tests, mock the SDK client at the dependency injection boundary.
    // For integration tests, override the base URL to point at a local server.

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
    mgmt, err := management.New("nhm_your_management_token")
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

### Deliveries

Read access to a workspace's webhook deliveries — paginated list scoped to an
endpoint, single-delivery metadata with an optional payload envelope, and
the list of HTTP attempts behind a delivery.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    nahook "github.com/getnahook/nahook-go"
    "github.com/getnahook/nahook-go/management"
)

func main() {
    mgmt, err := management.New("nhm_your_management_token")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    workspaceID := "ws_abc123"
    endpointID := "ep_abc123"

    // ── List deliveries (paginated, newest-first) ──
    //
    // NextCursor is an opaque server-encrypted token — pass it back on the
    // next call verbatim. It is nil when there are no more pages.
    limit := 50
    page, err := mgmt.Deliveries.List(ctx, workspaceID, endpointID, &nahook.ListDeliveriesOptions{
        Limit: &limit,
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, d := range page.Data {
        fmt.Printf("Delivery %s status=%s attempts=%d\n", d.ID, d.Status, d.TotalAttempts)
    }
    if page.NextCursor != nil {
        nextPage, _ := mgmt.Deliveries.List(ctx, workspaceID, endpointID, &nahook.ListDeliveriesOptions{
            Limit:  &limit,
            Cursor: *page.NextCursor,
        })
        fmt.Printf("Next page has %d more deliveries\n", len(nextPage.Data))
    }

    // ── Filter by status ──
    failed, err := mgmt.Deliveries.List(ctx, workspaceID, endpointID, &nahook.ListDeliveriesOptions{
        Status: "failed",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%d failed deliveries\n", len(failed.Data))

    // ── Get a single delivery's metadata ──
    delivery, err := mgmt.Deliveries.Get(ctx, workspaceID, "del_abc123", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Delivery %s endpoint=%s hasPayload=%v\n",
        delivery.ID, delivery.EndpointID, delivery.HasPayload)

    // ── Get with payload envelope ──
    //
    // The envelope is a tagged union: inspect Status before reading Data.
    // The four non-"available" statuses are returned with HTTP 200 — they
    // are not errors, they describe why the payload could not be returned.
    withPayload, err := mgmt.Deliveries.Get(ctx, workspaceID, "del_abc123", &nahook.GetDeliveryOptions{
        IncludePayload: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    if withPayload.Payload != nil {
        switch withPayload.Payload.Status {
        case "available":
            var body map[string]interface{}
            if err := json.Unmarshal(withPayload.Payload.Data, &body); err == nil {
                fmt.Printf("Payload (%s): %v\n", withPayload.Payload.ContentType, body)
            }
        case "forbidden":
            fmt.Println("Payload storage not included in this workspace's plan")
        case "processing":
            fmt.Println("Delivery still in flight, try again shortly")
        case "not_found":
            fmt.Println("No stored payload for this delivery")
        case "error":
            fmt.Println("Transient infrastructure failure reading the payload")
        }
    }

    // ── List attempts (chronological, oldest first) ──
    attempts, err := mgmt.Deliveries.GetAttempts(ctx, workspaceID, "del_abc123")
    if err != nil {
        log.Fatal(err)
    }
    for _, a := range attempts {
        statusCode := "n/a"
        if a.ResponseStatusCode != nil {
            statusCode = fmt.Sprintf("%d", *a.ResponseStatusCode)
        }
        fmt.Printf("Attempt #%d status=%s response=%s\n",
            a.AttemptNumber, a.Status, statusCode)
    }
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
| `WithHTTPClient(c)` | tuned default | Caller-owned `*http.Client` (see below) |

### Management Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL(url)` | `https://api.nahook.com` | API base URL |
| `WithTimeout(d)` | 30s | HTTP request timeout |
| `WithHTTPClient(c)` | tuned default | Caller-owned `*http.Client` (see below) |

The management client does not support retries.

### Advanced HTTP configuration

The SDK ships with a `*http.Client` backed by a tuned `*http.Transport`:

| Setting | Value |
|---|---|
| `MaxIdleConnsPerHost` | 50 |
| `MaxIdleConns` | 100 |
| `IdleConnTimeout` | 90s |
| `ForceAttemptHTTP2` | true |
| `Dialer.KeepAlive` | 30s |

Go's `http.DefaultTransport` allows only 2 idle conns per host — fine for a script, expensive for a service that fans out webhooks in bursts (every send tears down and re-opens a connection). The tuned defaults are sized for moderate concurrent throughput.

For full control, supply your own `*http.Client`:

```go
import (
    "net/http"
    "time"

    "github.com/getnahook/nahook-go/client"
)

custom := &http.Client{
    Timeout: 15 * time.Second,
    Transport: &http.Transport{
        // Don't forget Proxy: http.ProxyFromEnvironment if you want
        // HTTP_PROXY / HTTPS_PROXY env support — handcrafted Transports
        // drop the default's proxy wiring.
        Proxy: http.ProxyFromEnvironment,
        // Your own pool sizing, mTLS, custom RoundTripper, etc.
        MaxIdleConnsPerHost: 200,
        IdleConnTimeout:     60 * time.Second,
        ForceAttemptHTTP2:   true,
    },
}

c, err := client.New("nhk_us_...",
    client.WithHTTPClient(custom),
)
```

When `WithHTTPClient` is supplied:

- The SDK uses your client verbatim and does **not** mutate it (good for shared clients).
- Your `http.Client.Timeout` governs request timeouts and is what `TimeoutError.TimeoutMs` reports. If you set `Timeout: 0` (Go's "no timeout"), `TimeoutError.TimeoutMs` will be `0` in any error report — set a concrete timeout if you care about the reported value.
- `WithTimeout` is silently ignored — your client's `Timeout` wins.
- You own its lifecycle. `Client.Close()` is a no-op in this case — call `CloseIdleConnections()` on your own transport when you want to drain.

### Graceful shutdown — `Close()`

```go
c, err := client.New("nhk_us_...")
if err != nil {
    log.Fatal(err)
}
defer c.Close()
```

`Close()` drains the SDK-owned `*http.Transport`'s idle connection pool. Idempotent and cheap — safe in `defer` blocks, graceful shutdown handlers, or before recycling a long-lived client. It's a no-op when a custom `*http.Client` was supplied via `WithHTTPClient`, since that transport's lifecycle belongs to the caller.

`management.Management` exposes the same `Close()` method.

Skipping `Close()` is fine for short-lived scripts — the OS reaps sockets on process exit — but matters for test harnesses and long-running services where you want connections drained at known points.

Wrap your `http.RoundTripper` to plug in OpenTelemetry, Datadog, custom retry logic, or auth refresh — the SDK only ever calls `Do(req)` on the supplied client, so any standard middleware composes cleanly. The same `WithHTTPClient` option is available on the management client.

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
