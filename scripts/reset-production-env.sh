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
until docker exec -i wemall-postgres-1 pg_isready -U wemall >/dev/null 2>&1; do
    echo -n "."
    sleep 1
done
echo " PostgreSQL is ready!"

# Create databases
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_users;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_products;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_orders;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_sellers;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_notifications;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_reviews;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_payments;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_chat;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_dispute;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_admin;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_promotion;"
docker exec -i wemall-postgres-1 psql -U wemall -d postgres -c "CREATE DATABASE wemall_recommendation;"

echo "Restarting services that depend on the newly created databases..."
docker compose -f docker-compose.prod.yml restart user-service product-service order-service seller-service notification-service review-service payment-service chat-service dispute-service admin-service promotion-service recommendation-service

echo "Environment reset and deploy complete!"
