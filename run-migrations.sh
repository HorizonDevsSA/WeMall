#!/bin/bash
export DB_PASSWORD=$(grep DB_PASSWORD /home/ubuntu/WeMall/.env | cut -d= -f2-)
export DB_USER=$(grep DB_USER /home/ubuntu/WeMall/.env | cut -d= -f2-)
for s in user-service product-service order-service seller-service notification-service review-service payment-service chat-service dispute-service admin-service promotion-service recommendation-service delivery-service; do
  DB_NAME="wemall_$(echo $s | cut -d- -f1)"
  if [ "$s" = "user-service" ]; then DB_NAME="wemall_users"; fi
  if [ "$s" = "product-service" ]; then DB_NAME="wemall_products"; fi
  if [ "$s" = "order-service" ]; then DB_NAME="wemall_orders"; fi
  if [ "$s" = "seller-service" ]; then DB_NAME="wemall_sellers"; fi
  if [ "$s" = "notification-service" ]; then DB_NAME="wemall_notifications"; fi
  if [ "$s" = "review-service" ]; then DB_NAME="wemall_reviews"; fi
  if [ "$s" = "payment-service" ]; then DB_NAME="wemall_payments"; fi
  if [ "$s" = "promotion-service" ]; then DB_NAME="wemall_promotion"; fi
  if [ "$s" = "recommendation-service" ]; then DB_NAME="wemall_recommendation"; fi
  if [ "$s" = "delivery-service" ]; then DB_NAME="wemall_delivery"; fi
  
  SUFFIX=$(echo $DB_NAME | cut -d_ -f2)
  DB_HOST="postgres"
  echo "Migrating $s (Database: $DB_NAME on Host: $DB_HOST)..."
  docker run --rm -v /home/ubuntu/WeMall/services/$s/db/migrations:/migrations \
    --network wemall_wemall-net \
    migrate/migrate \
    -path=/migrations/ \
    -database "postgres://${DB_USER:-wemall}:${DB_PASSWORD:-wemall_secret}@${DB_HOST}:5432/$DB_NAME?sslmode=disable" up
done
