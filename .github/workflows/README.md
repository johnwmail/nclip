# Infrastructure Deployment Workflows

This directory contains GitHub Actions workflows for deploying nclip infrastructure on AWS.

## Available Workflows

### 1. `deploy-api-gateway.yml` - API Gateway (HTTP) Deployment

Creates an AWS API Gateway (HTTP API) that routes requests to your Lambda function.

**Features:**
- HTTP API Gateway v2 (more cost-effective than REST API)
- Lambda proxy integration
- CORS configuration
- Auto-deployment to specified stage
- Health monitoring and testing

**Usage:**
```bash
# Manual trigger with parameters
gh workflow run deploy-api-gateway.yml \
  -f lambda_function_name="your-lambda-function-name" \
  -f api_gateway_name="nclip-http-api" \
  -f stage_name="prod"

# Or push to trigger branch
git push origin your-branch:deploy/api-gateway
```

**Required Secrets:**
- `AWS_REGION` - AWS region for deployment
- `AWS_AUDIENCE` - OIDC audience for AWS authentication
- `AWS_ROLE_TO_ASSUME` - IAM role ARN for deployment

**Required Variables:**
- `LAMBDA_FUNCTION_NAME` - Default Lambda function name (if not provided in input)

### 2. `deploy-cloudfront.yml` - CloudFront Distribution

Creates a CloudFront distribution for global content delivery with HTTP to HTTPS redirect.

**Features:**
- HTTP to HTTPS redirect (enforced)
- Global edge caching
- Custom security headers
- Origin request/response policies
- Compression enabled
- IPv6 support

**Usage:**
```bash
# Manual trigger with parameters
gh workflow run deploy-cloudfront.yml \
  -f api_gateway_domain="abc123.execute-api.us-east-1.amazonaws.com" \
  -f price_class="PriceClass_100"

# Or push to trigger branch
git push origin your-branch:deploy/cloudfront
```

**Required Inputs:**
- `api_gateway_domain` - The domain name of your deployed API Gateway

### 3. `deploy-infrastructure.yml` - Complete Infrastructure

Deploys both API Gateway and CloudFront in sequence for a complete setup.

**Features:**
- Deploys API Gateway first
- Automatically configures CloudFront to use the API Gateway as origin
- Comprehensive testing of both components
- Deployment summary with all URLs
- Option to skip CloudFront deployment

**Usage:**
```bash
# Deploy complete infrastructure
gh workflow run deploy-infrastructure.yml \
  -f lambda_function_name="your-lambda-function-name" \
  -f deploy_cloudfront=true

# Deploy only API Gateway
gh workflow run deploy-infrastructure.yml \
  -f lambda_function_name="your-lambda-function-name" \
  -f deploy_cloudfront=false
```

## Prerequisites

### 1. AWS Setup

1. **Lambda Function**: Deploy your nclip Lambda function first using the existing `deploy-lambda.yml` workflow
2. **IAM Role**: Create an IAM role with the following permissions:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "cloudformation:*",
           "apigateway:*",
           "lambda:*",
           "cloudfront:*",
           "iam:PassRole"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

3. **OIDC Provider**: Configure GitHub OIDC provider in AWS IAM

### 2. GitHub Setup

Configure the following repository secrets:
- `AWS_REGION` - Your preferred AWS region (e.g., `us-east-1`)
- `AWS_AUDIENCE` - OIDC audience (usually `sts.amazonaws.com`)
- `AWS_ROLE_TO_ASSUME` - ARN of the IAM role for deployment

Configure the following repository variables:
- `LAMBDA_FUNCTION_NAME` - Name of your deployed Lambda function

## Deployment Flow

1. **Deploy Lambda** (existing workflow):
   ```bash
   gh workflow run deploy-lambda.yml
   ```

2. **Deploy Infrastructure**:
   ```bash
   gh workflow run deploy-infrastructure.yml \
     -f lambda_function_name="your-lambda-function"
   ```

3. **Access your service**:
   - API Gateway: `https://abc123.execute-api.region.amazonaws.com/prod`
   - CloudFront: `https://d1234567890.cloudfront.net`

## HTTP to HTTPS Redirect

The CloudFront distribution is configured with:
- `ViewerProtocolPolicy: redirect-to-https` - Automatically redirects HTTP to HTTPS
- `OriginProtocolPolicy: https-only` - Only communicates with origin via HTTPS
- Security headers including HSTS for enhanced security
- `CloudFront-Forwarded-Proto: https` header for application detection

## Cost Optimization

- **API Gateway**: HTTP API is ~70% cheaper than REST API
- **CloudFront**: Uses `PriceClass_100` by default (US, Canada, Europe)
- **Lambda**: Existing deployment remains unchanged

## Monitoring

All workflows include:
- Health endpoint testing
- Basic functionality validation
- Error handling and rollback capabilities
- Deployment status reporting

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure your IAM role has sufficient permissions
2. **Lambda Not Found**: Verify the Lambda function name is correct
3. **CloudFront Propagation**: Allow 10-15 minutes for global propagation

### Debugging

- Check CloudFormation stacks in AWS console
- Review workflow logs in GitHub Actions
- Test endpoints manually with curl

### Cleanup

To remove deployed infrastructure:
```bash
# Delete CloudFormation stacks
aws cloudformation delete-stack --stack-name nclip-cloudfront
aws cloudformation delete-stack --stack-name nclip-api-gateway-prod
```

## Security Considerations

- All traffic is forced to HTTPS
- CORS is configured for web browser compatibility
- Lambda permissions are scoped to specific API Gateway
- CloudFront includes security headers (HSTS, X-Frame-Options, etc.)
- No sensitive data is logged or exposed