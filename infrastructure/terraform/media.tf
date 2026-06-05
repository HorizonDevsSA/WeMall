# WeMall AWS Media Infrastructure Blueprint

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# ── Variables ─────────────────────────────────────────────────────────────────

variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "Target deployment region"
}

variable "project_name" {
  type        = string
  default     = "wemall"
  description = "Project prefix for resources"
}

variable "domain_name" {
  type        = string
  default     = "wemall.com"
  description = "Root domain for DNS mapping"
}

variable "cloudfront_public_signing_key_pem" {
  type        = string
  description = "PEM encoded public key registered in CloudFront to verify signed URLs"
}

# ── 1. S3 Storage Buckets ──────────────────────────────────────────────────────

# Raw uploads landing bucket
resource "aws_s3_bucket" "media_raw" {
  bucket        = "${var.project_name}-media-raw"
  force_destroy = true
}

# Lifecycle configuration to delete raw files after 24 hours
resource "aws_s3_bucket_lifecycle_configuration" "media_raw_lifecycle" {
  bucket = aws_s3_bucket.media_raw.id

  rule {
    id     = "auto-delete-raw-uploads"
    status = "Enabled"

    expiration {
      days = 1
    }
  }
}

# Processed public files bucket
resource "aws_s3_bucket" "media_public" {
  bucket        = "${var.project_name}-media-public"
  force_destroy = true
}

# Processed private files bucket
resource "aws_s3_bucket" "media_private" {
  bucket        = "${var.project_name}-media-private"
  force_destroy = true
}

# Block all public S3 access directly (files must be fetched through CloudFront CDN)
resource "aws_s3_bucket_public_access_block" "block_public_raw" {
  bucket = aws_s3_bucket.media_raw.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "block_public_processed" {
  bucket = aws_s3_bucket.media_public.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "block_public_private" {
  bucket = aws_s3_bucket.media_private.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# ── 2. CloudFront Origin Access Control (OAC) ───────────────────────────────

resource "aws_cloudfront_origin_access_control" "oac" {
  name                              = "${var.project_name}-media-oac"
  description                       = "OAC access to public and private media buckets"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# ── 3. Trusted Signer Key Groups (For Signed URLs/Cookies) ────────────────────

resource "aws_cloudfront_public_key" "signing_key" {
  name_prefix = "${var.project_name}-signing-public-key-"
  comment     = "Key used by WeMall media-service control plane to sign CloudFront URLs"
  encoded_key = var.cloudfront_public_signing_key_pem

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_cloudfront_key_group" "signing_group" {
  name    = "${var.project_name}-trusted-signing-keygroup"
  comment = "Keygroup for private media files"
  items   = [aws_cloudfront_public_key.signing_key.id]
}

# ── 4. CloudFront Distributions ──────────────────────────────────────────────

# Distribution for Public Assets
resource "aws_cloudfront_distribution" "public_distribution" {
  origin {
    domain_name              = aws_s3_bucket.media_public.bucket_regional_domain_name
    origin_id                = "S3PublicMedia"
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
  }

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "Public delivery distribution for WeMall optimized images"
  default_root_object = ""

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3PublicMedia"

    forwarded_values {
      query_string = false
      headers      = ["Origin", "Access-Control-Request-Headers", "Access-Control-Request-Method"]
      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 86400
    max_ttl                = 31536000
    compress               = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

# Distribution for Private Assets (enforces Signed URLs)
resource "aws_cloudfront_distribution" "private_distribution" {
  origin {
    domain_name              = aws_s3_bucket.media_private.bucket_regional_domain_name
    origin_id                = "S3PrivateMedia"
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
  }

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "Secured delivery distribution for private user assets"
  default_root_object = ""

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3PrivateMedia"

    trusted_key_groups = [aws_cloudfront_key_group.signing_group.id]

    forwarded_values {
      query_string = true # Required to forward signature parameters
      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 0 # Do not cache private responses globally without signatures
    max_ttl                = 0
    compress               = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

# ── 5. Bucket Policies ────────────────────────────────────────────────────────

# Allow CloudFront OAC to read from public bucket
resource "aws_s3_bucket_policy" "public_bucket_policy" {
  bucket = aws_s3_bucket.media_public.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCloudFrontOAC"
        Effect    = "Allow"
        Principal = { Service = "cloudfront.amazonaws.com" }
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.media_public.arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.public_distribution.arn
          }
        }
      }
    ]
  })
}

# Allow CloudFront OAC to read from private bucket
resource "aws_s3_bucket_policy" "private_bucket_policy" {
  bucket = aws_s3_bucket.media_private.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCloudFrontOAC"
        Effect    = "Allow"
        Principal = { Service = "cloudfront.amazonaws.com" }
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.media_private.arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.private_distribution.arn
          }
        }
      }
    ]
  })
}

# ── 6. Outputs ────────────────────────────────────────────────────────────────

output "s3_raw_bucket_name" {
  value       = aws_s3_bucket.media_raw.id
  description = "Upload raw landing zone S3 bucket name"
}

output "cloudfront_public_domain" {
  value       = aws_cloudfront_distribution.public_distribution.domain_name
  description = "Public CDN domain name"
}

output "cloudfront_private_domain" {
  value       = aws_cloudfront_distribution.private_distribution.domain_name
  description = "Private CDN domain name"
}
