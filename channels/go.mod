module github.com/dylanlyu/pusher-go/channels

go 1.25.0

require (
	github.com/dylanlyu/pusher-go/config v0.0.0
	github.com/dylanlyu/pusher-go/internal v0.0.0
	golang.org/x/crypto v0.50.0
)

replace (
	github.com/dylanlyu/pusher-go/config => ../config
	github.com/dylanlyu/pusher-go/internal => ../internal
)
