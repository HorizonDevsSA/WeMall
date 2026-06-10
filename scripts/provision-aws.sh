#!/bin/bash

set -e

echo "==========================================="
echo "   WeMall AWS Automated Provisioning       "
echo "==========================================="

REGION=$(aws configure get region)
if [ -z "$REGION" ]; then
    REGION="us-east-1"
fi
echo "Using region: $REGION"

INSTANCE_TYPE="t3.micro"
# Fetch latest Ubuntu 22.04 LTS AMI dynamically for the current region
AMI_ID=$(aws ssm get-parameters --names /aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp2/ami-id --query 'Parameters[0].Value' --output text)
echo "Resolved Ubuntu AMI: $AMI_ID"

KEY_NAME="wemall-prod-key"
SG_NAME="wemall-prod-sg"

# 1. Check AWS CLI setup
if ! command -v aws &> /dev/null; then
    echo "Error: AWS CLI is not installed."
    exit 1
fi

echo "Creating Key Pair ($KEY_NAME)..."
aws ec2 delete-key-pair --key-name $KEY_NAME 2>/dev/null || true
aws ec2 create-key-pair --key-name $KEY_NAME --query 'KeyMaterial' --output text > $KEY_NAME.pem
chmod 400 $KEY_NAME.pem

echo "Creating Security Group ($SG_NAME)..."
# Get VPC ID (default)
VPC_ID=$(aws ec2 describe-vpcs --filters "Name=isDefault,Values=true" --query 'Vpcs[0].VpcId' --output text)

SG_ID=$(aws ec2 create-security-group \
    --group-name $SG_NAME \
    --description "Security group for WeMall Production" \
    --vpc-id $VPC_ID \
    --query 'GroupId' \
    --output text 2>/dev/null) || SG_ID=$(aws ec2 describe-security-groups --group-names $SG_NAME --query 'SecurityGroups[0].GroupId' --output text)

echo "Authorizing Ingress Rules for $SG_NAME..."
aws ec2 authorize-security-group-ingress --group-id $SG_ID --protocol tcp --port 22 --cidr 0.0.0.0/0 2>/dev/null || true
aws ec2 authorize-security-group-ingress --group-id $SG_ID --protocol tcp --port 80 --cidr 0.0.0.0/0 2>/dev/null || true
aws ec2 authorize-security-group-ingress --group-id $SG_ID --protocol tcp --port 443 --cidr 0.0.0.0/0 2>/dev/null || true

echo "Launching EC2 Instance ($INSTANCE_TYPE)..."
INSTANCE_ID=$(aws ec2 run-instances \
    --image-id $AMI_ID \
    --instance-type $INSTANCE_TYPE \
    --key-name $KEY_NAME \
    --security-group-ids $SG_ID \
    --block-device-mappings '[{"DeviceName":"/dev/sda1","Ebs":{"VolumeSize":30,"VolumeType":"gp3"}}]' \
    --user-data file://scripts/aws-userdata.sh \
    --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=WeMall-Prod}]' \
    --query 'Instances[0].InstanceId' \
    --output text)

echo "Waiting for instance $INSTANCE_ID to enter 'running' state..."
aws ec2 wait instance-running --instance-ids $INSTANCE_ID

echo "Allocating and Associating Elastic IP..."
ALLOC_ID=$(aws ec2 allocate-address --domain vpc --query 'AllocationId' --output text)
EIP=$(aws ec2 describe-addresses --allocation-ids $ALLOC_ID --query 'Addresses[0].PublicIp' --output text)

aws ec2 associate-address --instance-id $INSTANCE_ID --allocation-id $ALLOC_ID > /dev/null

echo "==========================================="
echo "   Provisioning Complete!                  "
echo "==========================================="
echo "Instance ID : $INSTANCE_ID"
echo "Public IP   : $EIP"
echo "SSH Command : ssh -i $KEY_NAME.pem ubuntu@$EIP"
echo ""
echo "Note: It will take about 5-10 minutes for the userdata script to finish installing Docker, pulling the code, and starting the microservices."
echo "You can check the progress by SSHing into the machine and running:"
echo "   tail -f /var/log/user-data.log"
echo ""
echo "Once finished, your API gateway will be available at: http://$EIP.nip.io"
echo "==========================================="
