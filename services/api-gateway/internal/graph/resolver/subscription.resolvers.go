package resolver

// Subscriptions are defined in Phase 4 (Payment + Real-time).
// This file is a placeholder that satisfies the generated SubscriptionResolver
// interface so the gateway compiles cleanly in Phase 1.
//
// Planned subscriptions:
//   - orderStatusChanged(orderId: ID!) — Redis pub/sub → WebSocket
//   - inventoryUpdated(variantId: ID!) — live stock on product pages
//   - flashSaleUpdated(promotionId: ID!) — promotion countdowns
//   - newOrder — seller dashboard feed

// subscriptionResolver implements the generated SubscriptionResolver interface.
type subscriptionResolver struct{ *Resolver }
