# pusher-go

An **unofficial** Go SDK for [Pusher](https://pusher.com), covering two products:

- **[Channels](#channels)** — server-side publishing of WebSocket events via the Pusher HTTP API
- **[Beams](#beams)** — server-side triggering of push notifications via the Pusher Beams API

> **Disclaimer:** This project is not affiliated with, officially maintained by, or endorsed by Pusher. For the official SDKs, see [pusher-http-go](https://github.com/pusher/pusher-http-go) and [push-notifications-go](https://github.com/pusher/push-notifications-go).

## Installation

```bash
go get github.com/dylanlyu/pusher-go
```

Requires Go 1.25 or later.

---

## Channels

### Configuration

```go
import "github.com/dylanlyu/pusher-go/service/channels"

client, err := channels.New("APP_ID", "APP_KEY", "APP_SECRET",
    channels.WithCluster("mt1"),
    channels.WithSecure(true),
)
if err != nil {
    log.Fatal(err)
}
```

**Custom HTTP client (e.g. for timeouts):**

```go
httpClient := &http.Client{Timeout: 3 * time.Second}
client, err := channels.New("APP_ID", "APP_KEY", "APP_SECRET",
    channels.WithCluster("mt1"),
    channels.WithHTTPClient(httpClient),
)
```

**Available options:**

| Option | Description |
|---|---|
| `WithCluster(cluster string)` | Set Pusher cluster, e.g. `"eu"`, `"ap1"` |
| `WithHost(host string)` | Override API host (ignores cluster if set) |
| `WithSecure(secure bool)` | Force HTTPS |
| `WithHTTPClient(hc *http.Client)` | Custom HTTP client |
| `WithEncryptionMasterKeyBase64(key string)` | 32-byte E2E encryption master key (base64) |
| `WithMaxMessagePayloadKB(kb int)` | Override default 10 KB payload limit |

### Triggering Events

All methods accept a `context.Context` as the first argument.

**Single channel:**

```go
ctx := context.Background()
data := map[string]string{"message": "hello"}
err := client.Trigger(ctx, "my-channel", "my-event", data)
```

**Exclude a socket (prevent echo):**

```go
socketID := "1234.12"
result, err := client.TriggerWithParams(ctx, "my-channel", "my-event", data,
    channels.TriggerParams{SocketID: &socketID},
)
```

**Multiple channels:**

```go
err := client.TriggerMulti(ctx, []string{"ch-one", "ch-two"}, "my-event", data)
```

**Multiple channels with params:**

```go
result, err := client.TriggerMultiWithParams(ctx, []string{"ch-one", "ch-two"}, "my-event", data,
    channels.TriggerParams{SocketID: &socketID},
)
```

**Batch (up to 10 events in one request):**

```go
batch := []channels.Event{
    {Channel: "ch-one", Name: "event-a", Data: "hello"},
    {Channel: "ch-two", Name: "event-b", Data: "world"},
}
result, err := client.TriggerBatch(ctx, batch)
```

**Send to a specific authenticated user:**

```go
err := client.SendToUser(ctx, "user-123", "my-event", data)
```

### Authorizing Channels

**Private channels:**

```go
http.HandleFunc("/pusher/auth", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    response, err := client.AuthorizePrivateChannel(body)
    if err != nil {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(response)
})
```

**Presence channels:**

```go
memberData := channels.MemberData{
    UserID:   "user-123",
    UserInfo: map[string]string{"name": "Alice"},
}
response, err := client.AuthorizePresenceChannel(body, memberData)
```

**User authentication (Channels User Authentication):**

```go
userData := map[string]any{"id": "user-123", "name": "Alice"}
response, err := client.AuthenticateUser(body, userData)
```

### Application State

```go
// List channels (optionally filter by prefix)
prefix := "presence-"
info := "user_count"
chs, err := client.Channels(ctx, channels.ChannelsParams{
    FilterByPrefix: &prefix,
    Info:           &info,
})

// Single channel state
ch, err := client.Channel(ctx, "presence-chatroom", channels.ChannelParams{Info: &info})

// Users in a presence channel
users, err := client.GetChannelUsers(ctx, "presence-chatroom")
```

### Terminating User Connections

Force-disconnect all active connections for a user:

```go
err := client.TerminateUserConnections(ctx, "user-123")
```

### Webhook Validation

```go
http.HandleFunc("/pusher/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    webhook, err := client.Webhook(r.Header, body)
    if err != nil {
        http.Error(w, "Invalid webhook", http.StatusBadRequest)
        return
    }
    for _, event := range webhook.Events {
        fmt.Printf("event: %+v\n", event)
    }
})
```

### End-to-End Encryption

Generate a 32-byte master key:

```bash
openssl rand -base64 32
```

```go
client, err := channels.New("APP_ID", "APP_KEY", "APP_SECRET",
    channels.WithCluster("mt1"),
    channels.WithEncryptionMasterKeyBase64("<base64_master_key>"),
)
```

Only channels prefixed with `private-encrypted-` are encrypted. Encrypted channels cannot be triggered alongside non-encrypted channels in the same `TriggerMulti` call.

---

## Beams

### Configuration

```go
import "github.com/dylanlyu/pusher-go/service/beams"

client, err := beams.New("INSTANCE_ID", "SECRET_KEY")
if err != nil {
    log.Fatal(err)
}
```

**Available options:**

| Option | Description |
|---|---|
| `WithHTTPClient(hc *http.Client)` | Custom HTTP client |
| `WithBaseURL(url string)` | Override default Beams API endpoint |

### Publish to Interests

```go
ctx := context.Background()

publishRequest := map[string]any{
    "apns": map[string]any{
        "aps": map[string]any{
            "alert": map[string]any{
                "title": "Hello",
                "body":  "Hello, world",
            },
        },
    },
    "fcm": map[string]any{
        "notification": map[string]any{
            "title": "Hello",
            "body":  "Hello, world",
        },
    },
}

publishID, err := client.PublishToInterests(ctx, []string{"hello", "world"}, publishRequest)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Publish ID:", publishID)
```

Constraints: up to 100 interests per request; interest names may contain `A-Za-z0-9_\-=@,.;` and must be ≤ 164 characters.

### Publish to Users

```go
publishID, err := client.PublishToUsers(ctx, []string{"user-001", "user-002"}, publishRequest)
```

Up to 1,000 user IDs per request.

### Generate Beams Auth Token

Use this in your Beams authentication endpoint to issue signed JWTs to verified users:

```go
http.HandleFunc("/pusher/beams-auth", func(w http.ResponseWriter, r *http.Request) {
    // Verify the user via your own auth system first.
    userID := yourAuth.GetUserID(r)
    if userID == "" || userID != r.URL.Query().Get("user_id") {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    token, err := client.GenerateToken(userID)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(token)
})
```

### Delete a User

Removes all devices associated with a user from Beams:

```go
err := client.DeleteUser(ctx, "user-001")
```

---

## License

[MIT](LICENSE)
