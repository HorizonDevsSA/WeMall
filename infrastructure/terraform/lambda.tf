# IAM Role for the Lambda function
resource "aws_iam_role" "media_processor_role" {
  name = "${var.project_name}-media-processor-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# IAM Policy for the Lambda function
resource "aws_iam_policy" "media_processor_policy" {
  name        = "${var.project_name}-media-processor-policy"
  description = "IAM policy for media processor Lambda to read/write S3 and log"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = [
          "${aws_s3_bucket.media_raw.arn}/*",
          "${aws_s3_bucket.media_public.arn}/*",
          "${aws_s3_bucket.media_private.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "media_processor_attach" {
  role       = aws_iam_role.media_processor_role.name
  policy_arn = aws_iam_policy.media_processor_policy.arn
}

# Archive the Lambda source code
data "archive_file" "media_processor_zip" {
  type        = "zip"
  source_dir  = "${path.module}/../lambdas/media-processor"
  output_path = "${path.module}/media-processor.zip"
}

# Lambda Function
resource "aws_lambda_function" "media_processor" {
  filename         = data.archive_file.media_processor_zip.output_path
  function_name    = "${var.project_name}-media-processor"
  role             = aws_iam_role.media_processor_role.arn
  handler          = "index.handler"
  runtime          = "nodejs20.x"
  source_code_hash = data.archive_file.media_processor_zip.output_base64sha256
  timeout          = 30
  memory_size      = 1024

  environment {
    variables = {
      DEST_PUBLIC_BUCKET  = aws_s3_bucket.media_public.bucket
      DEST_PRIVATE_BUCKET = aws_s3_bucket.media_private.bucket
      API_CALLBACK_URL    = "https://api.${var.domain_name}/api/v1/media/callback"
    }
  }
}

# Lambda Permission to allow S3 to invoke it
resource "aws_lambda_permission" "allow_s3" {
  statement_id  = "AllowS3Invoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.media_processor.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.media_raw.arn
}

# S3 Bucket Notification to trigger the Lambda
resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = aws_s3_bucket.media_raw.id

  lambda_function {
    lambda_function_arn = aws_lambda_function.media_processor.arn
    events              = ["s3:ObjectCreated:*"]
  }

  depends_on = [aws_lambda_permission.allow_s3]
}
