// Package resolver implements all GraphQL resolvers for the WeMall API gateway.
// Each resolver method translates a GraphQL operation into one or more gRPC calls
// to the downstream microservices.
package resolver

import (
	"github.com/wemall/api-gateway/internal/clients"
)

// Resolver is the root dependency container injected into every generated resolver.
type Resolver struct {
	Clients *clients.Clients
}
