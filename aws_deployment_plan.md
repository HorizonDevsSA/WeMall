# AWS c7i-flex.large Deployment Plan for WeMall

This document outlines a flawless, step-by-step plan to deploy the WeMall application onto a single AWS `c7i-flex.large` instance, with all databases and services hosted internally on the same machine.

## 1. Architecture & Resource Considerations

> [!WARNING]
> A `c7i-flex.large` instance provides 2 vCPUs and 8 GiB of RAM. 
> The current `docker-compose.yml` runs **8 separate PostgreSQL containers**, plus Redis, Meilisearch, NATS, and 10 microservices. Running 8 separate Postgres containers will introduce significant RAM and CPU overhead, potentially starving the machine. 
> **Recommendation:** We will consolidate the 8 PostgreSQL databases into a **single PostgreSQL container** that contains multiple logical databases (`wemall_users`, `wemall_products`, etc.).

### Services to Deploy:
- **API Gateway & Microservices** (User, Product, Order, Seller, Notification, Media, Review, Payment, Documentation)
- **State/Databases**: 1x PostgreSQL (consolidated), 1x Redis, 1x NATS JetStream, 1x Meilisearch
- **Reverse Proxy**: Caddy or Nginx (for SSL/TLS and routing port 80/443 to the API Gateway)

---

## 2. Infrastructure Provisioning (AWS)

1. **Launch EC2 Instance:**
   - **Type:** `c7i-flex.large` (2 vCPUs, 8 GiB RAM).
   - **OS:** Ubuntu 22.04 LTS or 24.04 LTS.
   - **Storage:** Allocate at least 40 GB gp3 EBS volume (Databases + Docker images require significant space).
2. **Configure Security Group:**
   - Allow **SSH (Port 22)** from your IP address only.
   - Allow **HTTP (Port 80)** from Anywhere (0.0.0.0/0).
   - Allow **HTTPS (Port 443)** from Anywhere (0.0.0.0/0).
   - *Do not expose ports like 5432 (Postgres), 6379 (Redis), or 4222 (NATS) to the public internet.*
3. **Allocate Elastic IP:**
   - Assign an Elastic IP to the instance so the IP doesn't change on reboot.
   - Map your domain name (e.g., `api.wemall.com`) to this Elastic IP in Route 53 or your DNS provider.

---

## 3. Server Preparation & Hardening

1. **SSH into the instance:**
   ```bash
   ssh -i "your-key.pem" ubuntu@<elastic-ip>
   ```
2. **System Updates & Dependencies:**
   ```bash
   sudo apt update && sudo apt upgrade -y
   sudo apt install -y curl git ufw
   ```
3. **Set up UFW Firewall (Double Protection):**
   ```bash
   sudo ufw allow OpenSSH
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   sudo ufw enable
   ```
4. **Install Docker & Docker Compose:**
   - Follow the official Docker documentation to install Docker Engine and `docker-compose-plugin`.
   - Add the `ubuntu` user to the `docker` group to run docker commands without `sudo`.

---

## 4. Application Configuration

1. **Clone the Repository:**
   - Generate an SSH key on the server (`ssh-keygen`), add it to GitHub/GitLab as a deploy key, and clone the repo:
   ```bash
   git clone git@github.com:your-org/WeMall.git /home/ubuntu/WeMall
   cd /home/ubuntu/WeMall
   ```
2. **Create Production Environment Variables:**
   - Copy `.env.example` to `.env.production`.
   - Generate strong, secure passwords for Postgres, Meilisearch Master Key, Stripe Keys, etc.
3. **Create `docker-compose.prod.yml`:**
   - **Database Consolidation:** Create a custom initialization script (`init-multiple-dbs.sh`) and map it to `/docker-entrypoint-initdb.d/` in the Postgres container to create all 8 databases inside one container.
   - Update all `DB_URL` environment variables in the microservices to point to this single `postgres` container.
   - **Log Rotation:** Add Docker logging configuration to prevent logs from filling up the EBS volume:
     ```yaml
     logging:
       driver: "json-file"
       options:
         max-size: "10m"
         max-file: "3"
     ```

---

## 5. Reverse Proxy & SSL (Caddy)

To make the deployment flawless and secure, we will use **Caddy** as a reverse proxy. It automatically handles Let's Encrypt SSL certificates.

1. **Add Caddy to `docker-compose.prod.yml`:**
   ```yaml
   caddy:
     image: caddy:2-alpine
     ports:
       - "80:80"
       - "443:443"
     volumes:
       - ./Caddyfile:/etc/caddy/Caddyfile
       - caddy_data:/data
       - caddy_config:/config
   ```
2. **Create a `Caddyfile`:**
   ```caddyfile
   api.yourdomain.com {
       reverse_proxy api-gateway:8080
   }
   
   media.yourdomain.com {
       reverse_proxy media-service:8087
   }
   ```

---

## 6. Deployment & Execution

1. **Start the Infrastructure (Databases, Redis, NATS first):**
   ```bash
   docker compose -f docker-compose.prod.yml up -d postgres redis nats meilisearch
   ```
2. **Wait for Databases to Initialize.**
3. **Start the Microservices:**
   ```bash
   docker compose -f docker-compose.prod.yml up -d --build
   ```
4. **Verify Health:**
   ```bash
   docker compose -f docker-compose.prod.yml ps
   docker compose -f docker-compose.prod.yml logs -f api-gateway
   ```

---

## 7. Flawless Operations & Maintenance (Missing Steps Filled)

To ensure long-term stability on a single instance:

1. **Automated Database Backups (Cron):**
   - Create a daily cron job that runs `pg_dump` against the Postgres container and uploads the archive to an AWS S3 bucket using the AWS CLI.
2. **Swap File Configuration:**
   - Since `c7i-flex.large` has 8GB RAM and you are running a heavy microservice stack, configuring a **4GB Swap file** is critical to prevent Out-Of-Memory (OOM) kills during traffic spikes.
   ```bash
   sudo fallocate -l 4G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile
   echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
   ```
3. **CI/CD Integration:**
   - Set up GitHub Actions to SSH into the `c7i-flex.large` machine, pull the latest changes, and restart the specific updated containers seamlessly.
4. **Monitoring:**
   - Install the **CloudWatch Agent** on the EC2 instance to monitor memory usage and disk space, setting up alarms if memory usage exceeds 85% or disk space exceeds 80%.
