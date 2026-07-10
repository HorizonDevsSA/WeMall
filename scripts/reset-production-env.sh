#!/bin/bash
set -e

echo "Resetting production environment..."
cd /home/ubuntu/WeMall

# Copy template to .env
cp .env.production.example .env

# Fetch public IP
PUBLIC_IP=$(curl -s http://checkip.amazonaws.com)
echo "Resolved Public IP: $PUBLIC_IP"
sed -i "s/DOMAIN=api.yourdomain.com/DOMAIN=$PUBLIC_IP.nip.io/g" .env

# Generate secure random secrets
JWT_SECRET=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 16)
REDIS_PASSWORD=$(openssl rand -base64 16)
MEILI_MASTER_KEY=$(openssl rand -base64 16)

# Replace placeholders in .env
# Using different delimiter | in sed to avoid issues if secrets contain /
sed -i "s|generate_a_long_secure_random_string_here|$JWT_SECRET|g" .env
sed -i "s|generate_a_very_secure_password_here|$DB_PASSWORD|g" .env
sed -i "s|generate_redis_password_here|$REDIS_PASSWORD|g" .env
sed -i "s|generate_meili_master_key_here|$MEILI_MASTER_KEY|g" .env

echo "Stopping containers and wiping volumes..."
docker compose -f docker-compose.prod.yml down -v

echo "Starting containers with fresh volumes..."
docker compose -f docker-compose.prod.yml up -d

echo "Recreating consolidated databases..."
# Wait for postgres to be ready
echo "Waiting for PostgreSQL to start..."
until docker exec -i wemall-postgres pg_isready -U wemall >/dev/null 2>&1; do
    echo -n "."
    sleep 1
done
echo " PostgreSQL is ready!"

# Wait for consolidated databases to be initialized by the entrypoint script
echo "Waiting for consolidated databases to be created..."
until docker exec -i wemall-postgres psql -U wemall -d postgres -tAc "SELECT count(*) FROM pg_database WHERE datname IN ('wemall_users','wemall_products','wemall_orders','wemall_sellers','wemall_notifications','wemall_reviews','wemall_payments','wemall_chat','wemall_dispute','wemall_admin','wemall_promotion','wemall_recommendation','wemall_delivery','wemall_ecocash')" | grep -q "14"; do
    echo -n "."
    sleep 1
done
echo " All databases created!"

echo "Restarting services that depend on the newly created databases..."
docker compose -f docker-compose.prod.yml restart user-service product-service order-service seller-service notification-service review-service payment-service chat-service dispute-service admin-service promotion-service recommendation-service delivery-service ecocash-service

echo "Environment reset and deploy complete!"
