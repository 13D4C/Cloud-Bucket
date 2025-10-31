# Script manual

# sudo ./deploy_app.sh <EFS_ID> <DB_HOST> <DB_USER> <DB_PASSWORD> <DB_NAME>



# ================= RDS import Data ===================

# -------------- OPTIONAL ---------------- extract existing data -------------- OPTIONAL ----------------

# mysqldump -h [SOURCE_HOST] -u [SOURCE_USER] -p [SOURCE_DB_NAME] > backup.sql

# -------------- OPTIONAL ---------------- extract existing data -------------- OPTIONAL ----------------

#=====================================================================================
# sudo yum install -y mariadb-client

# ---------- OR using this -------------

# sudo yum install -y mariadb105   
# ======================================================================================================

# mysql -h <YOUR_RDS_ENDPOINT> -u <USERNAME> -p <DB_NAME> < <SQL_FILE>









#!/bin/bash

# ==============================================================================
#  Reusable Application Deployment Script for Amazon Linux 2023
#
#  Usage:
#  sudo ./deploy_app.sh <EFS_ID> <DB_HOST> <DB_USER> <DB_PASS> <DB_NAME>
# ==============================================================================

# Exit immediately if a command exits with a non-zero status.
set -e
# Print commands and their arguments as they are executed for easier debugging.
set -x

# --- Step 0: Validate and Assign Input Arguments ---
if [ "$#" -ne 5 ]; then
    echo "Illegal number of parameters. Usage: $0 <EFS_ID> <DB_HOST> <DB_USER> <DB_PASS> <DB_NAME>"
    exit 1
fi

EFS_FILESYSTEM_ID=$1
DB_HOST=$2
DB_USER=$3
DB_PASSWORD=$4
DB_NAME=$5

# --- Step 1: Install Dependencies ---
# ⭐ อัปเกรด: เพิ่ม 'mariadb' (ซึ่งรวม client) เข้าไป
echo "Installing dependencies: git, amazon-efs-utils, docker, mariadb"
while fuser /var/run/yum.pid >/dev/null 2>&1; do
  echo "Waiting for other yum processes to finish..."
  sleep 1
done
yum update -y
yum install -y git amazon-efs-utils docker mariadb
echo "Dependencies installed."

# --- Step 2: Configure and Start Services ---
echo "Starting and enabling Docker service"
service docker start
systemctl enable docker
usermod -a -G docker ec2-user
echo "Services configured."

# --- Step 3: Mount EFS Filesystem ---
MOUNT_POINT="/mnt/data"
echo "Mounting EFS filesystem $EFS_FILESYSTEM_ID to $MOUNT_POINT"
mkdir -p $MOUNT_POINT
if ! mount | grep -q "on ${MOUNT_POINT} type efs"; then
  mount -t efs -o tls $EFS_FILESYSTEM_ID:/ $MOUNT_POINT
fi
if ! grep -q "$EFS_FILESYSTEM_ID" /etc/fstab; then
  echo "$EFS_FILESYSTEM_ID:/ $MOUNT_POINT efs _netdev,tls 0 0" >> /etc/fstab
fi
# (1) ให้สิทธิ์ chmod 
echo "Setting file permissions for EFS mount point..."
chmod 777 $MOUNT_POINT

# --- Step 4: Prepare Application Directory ---
APP_DIR=$(pwd)
echo "Ensuring correct ownership for application directory: $APP_DIR"
chown -R ec2-user:ec2-user "$APP_DIR"

# --- Step 5: Create .env file ---
echo "Creating .env file in $APP_DIR"
cat <<EOT > "${APP_DIR}/.env"
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_HOST=${DB_HOST}
DB_PORT=3306
DB_NAME=${DB_NAME}
JWT_SECRET_KEY="That moment of joy when you find out the snack you love is actually Halal"
EOT
echo ".env file created."

# --- Step 6: (Optional) Import Database ---
DB_SQL_FILE="database.sql"
echo "Checking for database file: $DB_SQL_FILE"

if [ -f "$DB_SQL_FILE" ]; then
    echo "Found $DB_SQL_FILE, attempting to import into database '$DB_NAME'..."
    export MYSQL_PASSWORD="$DB_PASSWORD"
    mysql -h "$DB_HOST" -u "$DB_USER" "$DB_NAME" < "$DB_SQL_FILE"
    unset MYSQL_PASSWORD
    echo "Database import complete."
else
    echo "$DB_SQL_FILE not found, skipping database import."
fi

# --- Step 7: Build and Run Docker Container ---
echo "Checking for Dockerfile..."
if [ ! -f "Dockerfile" ]; then
    echo "ERROR: Dockerfile not found in $APP_DIR"
    exit 1
fi

echo "Building Docker image..."
docker build -t clowncloud:1.0 .

echo "Running Docker container..."
docker rm -f my-website || true
docker run -d --name my-website -p 8080:8080 \
  --restart always \
  -v "${APP_DIR}/.env:/app/.env" \
  -v "${MOUNT_POINT}:/app/uploads" \
  clowncloud:1.0

echo "--- Deployment script finished successfully at $(date) ---"
echo "Application should be running. Check with 'docker ps'."
