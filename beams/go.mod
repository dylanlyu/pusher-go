module github.com/dylanlyu/pusher-go/beams

go 1.22

require (
	github.com/dylanlyu/pusher-go/config v0.0.0
	github.com/dylanlyu/pusher-go/internal v0.0.0
	github.com/golang-jwt/jwt/v5 v5.3.1
)

replace (
	github.com/dylanlyu/pusher-go/config => ../config
	github.com/dylanlyu/pusher-go/internal => ../internal
)
