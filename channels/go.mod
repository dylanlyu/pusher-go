module github.com/dylanlyu/pusher-go/channels

go 1.25.0

require (
	github.com/dylanlyu/pusher-go/internal v1.0.0
	golang.org/x/crypto v0.50.0
)

require golang.org/x/sys v0.43.0 // indirect

replace (
	github.com/dylanlyu/pusher-go/config => ../config
	github.com/dylanlyu/pusher-go/internal => ../internal
)
