# IAM User for WeMall Media Service (Application Server)
resource "aws_iam_user" "media_service_user" {
  name = "${var.project_name}-media-svc-user"
}

resource "aws_iam_access_key" "media_service_key" {
  user = aws_iam_user.media_service_user.name
}

resource "aws_iam_user_policy" "media_service_policy" {
  name = "${var.project_name}-media-svc-policy"
  user = aws_iam_user.media_service_user.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:HeadObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.media_raw.arn,
          "${aws_s3_bucket.media_raw.arn}/*",
          aws_s3_bucket.media_public.arn,
          "${aws_s3_bucket.media_public.arn}/*",
          aws_s3_bucket.media_private.arn,
          "${aws_s3_bucket.media_private.arn}/*"
        ]
      }
    ]
  })
}

output "media_service_access_key" {
  value = aws_iam_access_key.media_service_key.id
}

output "media_service_secret_key" {
  value     = aws_iam_access_key.media_service_key.secret
  sensitive = true
}
