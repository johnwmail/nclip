# AWS Lambda Deployment Guide for nclip

This guide provides comprehensive instructions for deploying nclip as a serverless application on AWS Lambda with S3 storage.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [IAM Permissions](#iam-permissions)
- [S3 Bucket Setup](#s3-bucket-setup)
- [Deployment](#deployment)
- [Configuration](#configuration)
- [CloudWatch Logging](#cloudwatch-logging)
- [Monitoring & Debugging](#monitoring--debugging)
- [Troubleshooting](#troubleshooting)
- [Cost Optimization](#cost-optimization)

## Overview

nclip automatically detects when running on AWS Lambda (via `AWS_LAMBDA_FUNCTION_NAME` environment variable) and switches to S3 for storage. This provides a serverless deployment with automatic scaling and pay-per-use pricing.

**Architecture:**
- **Runtime:** AWS Lambda (provided.al2023)
- **Storage:** Amazon S3
- **API Gateway:** HTTP API or Function URL
- **Logging:** CloudWatch Logs

## Prerequisites

### AWS Account & CLI
- AWS account with appropriate permissions
- AWS CLI installed and configured
- GitHub repository access (for automated deployment)

### Required AWS Services
- **S3 Bucket:** For paste content and metadata storage
- **IAM Role:** For Lambda function execution
- **Lambda Function:** The nclip application
- **API Gateway (optional):** For custom domain and advanced routing

## IAM Permissions

### Lambda Execution Role

Create an IAM role with the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "arn:aws:logs:*:*:*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:HeadObject"
            ],
            "Resource": [
                "arn:aws:s3:::YOUR_BUCKET_NAME",
                "arn:aws:s3:::YOUR_BUCKET_NAME/*"
            ]
        }
    ]
}
```

**Policy Breakdown:**
- `logs:*` - CloudWatch logging permissions
- `s3:GetObject` - Read paste content and metadata
- `s3:PutObject` - Create new pastes
- `s3:DeleteObject` - Clean up expired pastes
- `s3:HeadObject` - Check if pastes exist (for slug generation)

### GitHub Actions Permissions

For automated deployment, ensure your GitHub repository has these secrets/variables:

**Required Secrets:**
- `LAMBDA_EXECUTION_ROLE` - ARN of the IAM role above

**Required Variables:**
- `LAMBDA_FUNCTION_NAME` - Your Lambda function name
- `S3_BUCKET` - S3 bucket name
- `S3_PREFIX` - S3 key prefix (default: "nclip")
- `NCLIP_BUFFER_SIZE` - Max upload size (default: "5242880")
- `GIN_MODE` - Set to "release" for production

## S3 Bucket Setup

### Create Bucket

```bash
# Create S3 bucket
aws s3api create-bucket \
    --bucket your-nclip-bucket \
    --region us-east-1 \
    --create-bucket-configuration LocationConstraint=us-east-1

# Enable versioning (recommended)
aws s3api put-bucket-versioning \
    --bucket your-nclip-bucket \
    --versioning-configuration Status=Enabled
```

### Bucket Policy (Optional)

For public read access to pastes:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::your-nclip-bucket/nclip/*"
        }
    ]
}
```

**Note:** This makes pastes publicly accessible. For private pastes, implement authentication in your application.

## Deployment

### Automated Deployment (Recommended)

1. **Set Repository Variables:**
   - Go to your GitHub repository → Settings → Secrets and variables → Actions
   - Add the variables listed in [IAM Permissions](#iam-permissions)

2. **Deploy via Git Branch:**
   ```bash
   # Push to deployment branch
   git checkout -b deploy/lambda
   git push origin deploy/lambda
   ```

3. **Monitor Deployment:**
   - Check GitHub Actions tab for deployment status
   - Function ARN and version will be displayed in workflow logs

### Manual Deployment

1. **Build for Lambda:**
   ```bash
   # Set environment variables
   export GOOS=linux
   export CGO_ENABLED=0
   export GOARCH=amd64  # or arm64

   # Build
   go build -ldflags "-s -w" -o bootstrap .
   zip lambda-function.zip bootstrap
   ```

2. **Create/Update Function:**
   ```bash
   aws lambda create-function \
       --function-name your-function-name \
       --runtime provided.al2023 \
       --role arn:aws:iam::ACCOUNT:role/nclip-lambda-role \
       --handler bootstrap \
       --zip-file fileb://lambda-function.zip \
       --architectures x86_64 \
       --environment "Variables={NCLIP_S3_BUCKET=your-bucket,NCLIP_S3_PREFIX=nclip,GIN_MODE=release}"
   ```

### Function URL Setup

Create a Function URL for direct access:

```bash
aws lambda add-permission \
    --function-name your-function-name \
    --statement-id FunctionURLAllowPublicAccess \
    --action lambda:InvokeFunctionUrl \
    --principal "*" \
    --function-url-auth-type NONE

aws lambda create-function-url-config \
    --function-name your-function-name \
    --auth-type NONE
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `NCLIP_S3_BUCKET` | S3 bucket name | - | Yes |
| `NCLIP_S3_PREFIX` | S3 key prefix | `nclip` | No |
| `NCLIP_BUFFER_SIZE` | Max upload size (bytes) | `5242880` | No |
| `GIN_MODE` | Gin framework mode | `debug` | No |
| `NCLIP_URL` | Base URL for links | Auto-detected | No |
| `NCLIP_TTL` | Default paste TTL | `24h` | No |

### Lambda Configuration

**Memory:** 128 MB (minimum recommended)
**Timeout:** 30 seconds (default is usually sufficient)
**Architecture:** x86_64 or arm64

## CloudWatch Logging

### Log Group

Lambda automatically creates a CloudWatch log group:
```
/aws/lambda/your-function-name
```

### Viewing Logs with AWS CLI

#### Real-time Log Tailing
```bash
# Tail logs in real-time
aws logs tail /aws/lambda/your-function-name \
    --region us-east-1 \
    --follow \
    --format short
```

#### Get Recent Logs
```bash
# Last 5 minutes
aws logs tail /aws/lambda/your-function-name \
    --region us-east-1 \
    --since 5m \
    --format short
```

#### Filter by Time Range
```bash
# Specific time range
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --start-time $(date -d '1 hour ago' +%s) \
    --end-time $(date +%s) \
    --region us-east-1
```

#### Search for Specific Events
```bash
# Find errors
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --filter-pattern "ERROR" \
    --region us-east-1

# Find specific paste operations
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --filter-pattern "POST" \
    --region us-east-1
```

#### Export Logs
```bash
# Export to file
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --start-time $(date -d '1 day ago' +%s) \
    --region us-east-1 \
    --output text > lambda-logs.txt
```

## Monitoring & Debugging

### Key Metrics to Monitor

#### Lambda Metrics
```bash
# Function invocations
aws cloudwatch get-metric-statistics \
    --namespace AWS/Lambda \
    --metric-name Invocations \
    --dimensions Name=FunctionName,Value=your-function-name \
    --start-time $(date -d '1 day ago' +%s) \
    --end-time $(date +%s) \
    --period 3600 \
    --statistics Sum \
    --region us-east-1
```

#### Error Rate
```bash
# Function errors
aws cloudwatch get-metric-statistics \
    --namespace AWS/Lambda \
    --metric-name Errors \
    --dimensions Name=FunctionName,Value=your-function-name \
    --start-time $(date -d '1 hour ago' +%s) \
    --end-time $(date +%s) \
    --period 300 \
    --statistics Sum \
    --region us-east-1
```

### Common Debug Scenarios

#### Paste Creation Failures
```bash
# Look for slug generation errors
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --filter-pattern "failed to generate slug" \
    --region us-east-1
```

#### S3 Permission Issues
```bash
# Check for S3 access errors
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --filter-pattern "S3.*error\|HeadObject\|AccessDenied" \
    --region us-east-1
```

#### High Latency Requests
```bash
# Find slow requests
aws logs filter-log-events \
    --log-group-name /aws/lambda/your-function-name \
    --filter-pattern "1.0s\|2.0s\|3.0s" \
    --region us-east-1
```

### X-Ray Tracing

Enable X-Ray for detailed performance insights:

```bash
aws lambda update-function-configuration \
    --function-name your-function-name \
    --tracing-config Mode=Active \
    --region us-east-1
```

## Troubleshooting

### Common Issues

#### 1. "Failed to create paste: failed to generate slug"
**Cause:** S3 permission issues or bucket not accessible
**Solution:**
```bash
# Check IAM role permissions
aws iam get-role-policy --role-name nclip-lambda-role --policy-name nclip-s3-policy

# Verify bucket exists and is accessible
aws s3 ls s3://your-bucket-name/
```

#### 2. Function Times Out
**Cause:** Lambda timeout too low or S3 operations slow
**Solution:**
```bash
# Increase timeout
aws lambda update-function-configuration \
    --function-name your-function-name \
    --timeout 60 \
    --region us-east-1
```

#### 3. High Cold Start Times
**Cause:** Function not optimized for Lambda
**Solution:**
- Use provisioned concurrency
- Optimize package size
- Consider ARM64 architecture

#### 4. CORS Issues
**Cause:** Missing CORS headers in API Gateway
**Solution:** Configure CORS in API Gateway or Function URL

### Health Checks

```bash
# Test function health
curl -X GET https://your-function-url/health

# Test paste creation
curl -X POST https://your-function-url \
    -H "Content-Type: text/plain" \
    -d "test paste"
```

## Cost Optimization

### Lambda Costs
- **Free Tier:** 1M requests/month, 400,000 GB-seconds
- **Pay per use:** $0.20 per 1M requests + $0.0000166667 per GB-second

### S3 Costs
- **Storage:** $0.023 per GB/month
- **Requests:** $0.0004 per 1,000 requests
- **Data Transfer:** $0.09 per GB (first 10TB)

### Optimization Tips

1. **Enable Provisioned Concurrency** for consistent performance
2. **Use ARM64** for 20% cost reduction
3. **Implement TTL** to auto-delete expired pastes
4. **Monitor usage** and set budgets
5. **Use CloudFront** for frequently accessed pastes

### Cost Monitoring

```bash
# Lambda costs
aws ce get-cost-and-usage \
    --time-period Start=2024-01-01,End=2024-02-01 \
    --metrics "BlendedCost" \
    --group-by Type=DIMENSION,Key=SERVICE \
    --filter '{"Dimensions": {"Key": "SERVICE", "Values": ["AWS Lambda"]}}' \
    --region us-east-1

# S3 costs
aws ce get-cost-and-usage \
    --time-period Start=2024-01-01,End=2024-02-01 \
    --metrics "BlendedCost" \
    --group-by Type=DIMENSION,Key=SERVICE \
    --filter '{"Dimensions": {"Key": "SERVICE", "Values": ["Amazon Simple Storage Service"]}}' \
    --region us-east-1
```

## Security Best Practices

1. **Least Privilege:** Use minimal IAM permissions
2. **VPC Deployment:** Run in VPC for enhanced security
3. **Encryption:** Enable S3 SSE-KMS if needed
4. **Monitoring:** Enable CloudTrail and GuardDuty
5. **Rate Limiting:** Implement in API Gateway

## References

- [AWS Lambda Documentation](https://docs.aws.amazon.com/lambda/)
- [Amazon S3 Documentation](https://docs.aws.amazon.com/s3/)
- [CloudWatch Logs Documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/)
- [nclip GitHub Repository](https://github.com/johnwmail/nclip)