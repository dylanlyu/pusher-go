# pusher-go

An **unofficial** Go SDK for [Pusher](https://pusher.com), covering two products:

- **[Channels](#channels)** — server-side publishing of WebSocket events via the Pusher HTTP API
- **[Beams](#beams)** — server-side triggering of push notifications via the Pusher Beams API

> **Disclaimer:** This project is not affiliated with, officially maintained by, or endorsed by Pusher. For the official SDKs, see [pusher-http-go](https://github.com/pusher/pusher-http-go) and [push-notifications-go](https://github.com/pusher/push-notifications-go).

## Installation

Install only the product(s) you need — `channels` and `beams` are separate Go modules:

```bash
# Pusher Channels
go get github.com/dylanlyu/pusher-go/channels

# Pusher Beams
go get github.com/dylanlyu/pusher-go/beams
```

Requires Go 1.22 or later.

---

## Channels

### Configuration

```go
import "github.com/dylanlyu/pusher-go/channels"

client := channels.NewClient(channels.Config{
    AppID:   "APP_ID",
    Key:     "APP_KEY",
    Secret:  "APP_SECRET",
    Cluster: "APP_CLUSTER",
})
```

**HTTPS:**

```go
client := channels.NewClient(channels.Config{
    AppID:   "APP_ID",
    Key:     "APP_KEY",
    Secret:  "APP_SECRET",
    Cluster: "mt1",
    Secure:  true,
})
```

**Custom HTTP client (e.g. for timeouts):**

```go
import "net/http"
import "time"

httpClient := &http.Client{Timeout: 3 * time.Second}
client := channels.NewClient(channels.Config{...}, channels.WithHTTPClient(httpClient))
```

**From environment variable** (`PUSHER_URL=http://<key>:<secret>@api-<cluster>.pusher.com/apps/<app_id>`):

```go
client, err := channels.NewClientFromEnv("PUSHER_URL")
```

### Triggering Events

**Single channel:**

```go
data := map[string]string{"message": "hello"}
err := client.Trigger("my-channel", "my-event", data)
```

**Exclude a socket (prevent echo):**

```go
socketID := "1234.12"
err := client.TriggerWithParams("my-channel", "my-event", data, channels.TriggerParams{
    SocketID: &socketID,
})
```

**Multiple channels:**

```go
err := client.TriggerMulti([]string{"ch-one", "ch-two"}, "my-event", data)
```

**Batch (up to 10 events in a single request):**

```go
batch := []channels.Event{
    {Channel: "ch-one", Name: "event-a", Data: "hello"},
    {Channel: "ch-two", Name: "event-b", Data: "world"},
}
err := client.TriggerBatch(batch)
```

**Send to a specific authenticated user:**

```go
err := client.SendToUser("user-123", "my-event", data)
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

**Authenticating users (Channels User Authentication):**

```go
userData := map[string]interface{}{"id": "user-123", "name": "Alice"}
response, err := client.AuthenticateUser(body, userData)
```

### Application State

```go
// List all channels (optionally filter by prefix)
prefix := "presence-"
info := "user_count"
chs, err := client.Channels(channels.ChannelsParams{
    FilterByPrefix: &prefix,
    Info:           &info,
})

// Single channel state
ch, err := client.Channel("presence-chatroom", channels.ChannelParams{Info: &info})

// Users in a presence channel
users, err := client.GetChannelUsers("presence-chatroom")
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
client := channels.NewClient(channels.Config{
    AppID:                     "APP_ID",
    Key:                       "APP_KEY",
    Secret:                    "APP_SECRET",
    Cluster:                   "mt1",
    EncryptionMasterKeyBase64: "<base64_master_key>",
})
```

Only channels prefixed with `private-encrypted-` are encrypted.

---

## Beams

### Configuration

```go
import "github.com/dylanlyu/pusher-go/beams"

client, err := beams.NewClient("INSTANCE_ID", "SECRET_KEY")
if err != nil {
    log.Fatal(err)
}
```

### Publish to Interests

```go
publishRequest := map[string]interface{}{
    "apns": map[string]interface{}{
        "aps": map[string]interface{}{
            "alert": map[string]interface{}{
                "title": "Hello",
                "body":  "Hello, world",
            },
        },
    },
    "fcm": map[string]interface{}{
        "notification": map[string]interface{}{
            "title": "Hello",
            "body":  "Hello, world",
        },
    },
}

publishID, err := client.PublishToInterests([]string{"hello", "world"}, publishRequest)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Publish ID:", publishID)
```

Constraints: up to 100 interests per request; interest names may contain `A-Za-z0-9_\-=@,.;` and must be ≤ 164 characters.

### Publish to Users

```go
publishID, err := client.PublishToUsers([]string{"user-001", "user-002"}, publishRequest)
```

Up to 1,000 user IDs per request.

### Generate Beams Auth Token

Use this in your Beams authentication endpoint to issue signed JWTs to verified users:

```go
http.HandleFunc("/pusher/beams-auth", func(w http.ResponseWriter, r *http.Request) {
    // Verify the user via your own auth system.
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
err := client.DeleteUser("user-001")
```

---

## License

[MIT](LICENSE)
