#!/bin/bash
export DB_PASSWORD=$(grep DB_PASSWORD /home/ubuntu/WeMall/.env | cut -d= -f2)
for s in user-service product-service order-service media-service; do
  DB_NAME="wemall_$(echo $s | cut -d- -f1)"
  if [ "$s" = "user-service" ]; then DB_NAME="wemall_users"; fi
  if [ "$s" = "product-service" ]; then DB_NAME="wemall_products"; fi
  if [ "$s" = "order-service" ]; then DB_NAME="wemall_orders"; fi
  if [ "$s" = "media-service" ]; then DB_NAME="wemall_media"; fi
  
  echo "Migrating $s (Database: $DB_NAME)..."
  docker run --rm -v /home/ubuntu/WeMall/services/$s/db/migrations:/migrations \
    --network wemall_wemall-net \
    migrate/migrate \
    -path=/migrations/ \
    -database "postgres://wemall:${DB_PASSWORD}@postgres:5432/$DB_NAME?sslmode=disable" up
done
