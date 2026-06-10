#!/bin/bash

# Log all output to a file for debugging
exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

echo "Starting WeMall AWS Cloud-Init Setup..."

# 1. Update and install dependencies
apt-get update
apt-get install -y ca-certificates curl gnupg git ufw jq

# 2. Setup 4GB Swap Space (Critical for t3.large running microservices)
echo "Setting up swap space..."
fallocate -l 4G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab

# 3. Configure Firewall (UFW)
echo "Configuring UFW..."
ufw --force enable
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp

# 4. Install Docker
echo "Installing Docker..."
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 5. Add default ubuntu user to docker group
usermod -aG docker ubuntu

# 6. Clone the Repository
echo "Cloning WeMall repository..."
cd /home/ubuntu
# Using HTTPS clone since we are running via automated script without SSH keys setup for GitHub yet.
# Note: For private repos, you'd inject a PAT or use SSH keys. Assuming public for now or public fork.
git clone https://github.com/HorizonDevsSA/WeMall.git
cd WeMall

# 7. Setup Environment Variables
echo "Setting up environment variables..."
cp .env.production.example .env

# Replace domain name in .env with the instance's public IP temporarily so it works out of the box
PUBLIC_IP=$(curl -s http://checkip.amazonaws.com)
sed -i "s/DOMAIN=api.yourdomain.com/DOMAIN=$PUBLIC_IP.nip.io/g" .env

# Generate secure random secrets
JWT_SECRET=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 16)
REDIS_PASSWORD=$(openssl rand -base64 16)
MEILI_MASTER_KEY=$(openssl rand -base64 16)

sed -i "s/generate_a_long_secure_random_string_here/$JWT_SECRET/g" .env
sed -i "s/generate_a_very_secure_password_here/$DB_PASSWORD/g" .env
sed -i "s/generate_redis_password_here/$REDIS_PASSWORD/g" .env
sed -i "s/generate_meili_master_key_here/$MEILI_MASTER_KEY/g" .env

# Replace Caddyfile domain
sed -i "s/api.yourdomain.com/$PUBLIC_IP.nip.io/g" Caddyfile
sed -i "s/media.yourdomain.com/media.$PUBLIC_IP.nip.io/g" Caddyfile

chown -R ubuntu:ubuntu /home/ubuntu/WeMall

# 8. Start the Application sequentially to avoid OOM killer
echo "Building microservices sequentially..."
su - ubuntu -c "cd /home/ubuntu/WeMall && for svc in api-gateway user-service product-service order-service seller-service notification-service media-service documentation-service review-service payment-service chat-service dispute-service admin-service promotion-service recommendation-service; do docker compose -f docker-compose.prod.yml build \$svc; done"

echo "Starting Docker Compose..."
su - ubuntu -c "cd /home/ubuntu/WeMall && docker compose -f docker-compose.prod.yml up -d"

echo "WeMall AWS Setup Complete!"
