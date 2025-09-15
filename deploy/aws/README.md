# AWS Lambda Deployment

This directory contains the AWS Lambda deployment configuration for nclip using AWS SAM (Serverless Application Model).

## Architecture

The AWS Lambda deployment includes:

- **AWS Lambda Function**: Runs the nclip application
- **API Gateway**: Provides HTTP endpoints
- **DynamoDB Table**: Stores paste data with automatic TTL
- **CloudWatch Logs**: Application logging
- **CloudWatch Alarms**: Monitoring and alerting
- **IAM Roles**: Least-privilege access control

## Prerequisites

1. **AWS CLI v2**: [Installation Guide](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
2. **SAM CLI**: [Installation Guide](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
3. **Go 1.19+**: [Installation Guide](https://golang.org/doc/install)
4. **AWS Credentials**: Configured via `aws configure`

## Quick Start

### 1. Deploy to Staging

```bash
cd deploy/aws
./deploy.sh
```

This deploys to the `staging` environment in `us-east-1` region.

### 2. Deploy to Production

```bash
./deploy.sh production us-west-2
```

### 3. Deploy with Custom Domain

```bash
./deploy.sh staging us-east-1 api.example.com
```

## Manual Deployment

### Build and Deploy

```bash
# Build the application
cd ../../
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/bootstrap .
cd dist && zip lambda.zip bootstrap && rm bootstrap

# Deploy with SAM
cd ../deploy/aws
sam deploy --guided
```

### Local Testing

```bash
# Start local API
sam local start-api --port 3001

# Test locally
curl -X POST http://localhost:3001 -d "Hello World"
curl http://localhost:3001/$(echo "test" | curl -s -X POST http://localhost:3001 -d @-)
```

## Configuration

### Environment Variables

The Lambda function supports these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NCLIP_DYNAMODB_TABLE` | Auto-set | DynamoDB table name |
| `NCLIP_EXPIRE_DAYS` | `7` | Days before pastes expire |
| `NCLIP_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `NCLIP_RATE_LIMIT` | `20/min` | Rate limit per IP address |
| `NCLIP_BUFFER_SIZE_MB` | `10` | Maximum paste size in MB |
| `NCLIP_DOMAIN` | Auto-set | Domain for paste URLs |

**Note**: Lambda deployment only supports DynamoDB storage. No storage type configuration is needed.

### CloudFormation Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `Environment` | `staging` | Environment name (staging/production) |
| `Domain` | _(empty)_ | Custom domain for API Gateway |
| `ExpireDays` | `7` | Default paste expiration |
| `RateLimit` | `20/min` | Rate limiting configuration |

## Monitoring

### CloudWatch Logs

```bash
# Tail logs in real-time
sam logs --stack-name nclip-staging --tail

# Filter error logs
aws logs filter-log-events \
  --log-group-name /aws/lambda/nclip-staging \
  --filter-pattern ERROR
```

### CloudWatch Metrics

The deployment includes these alarms:

- **Error Rate**: Triggers when error count > 10 in 5 minutes
- **Duration**: Triggers when average duration > 25 seconds

### Custom Metrics

```bash
# View function metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name Invocations \
  --dimensions Name=FunctionName,Value=nclip-staging \
  --start-time 2023-01-01T00:00:00Z \
  --end-time 2023-01-02T00:00:00Z \
  --period 3600 \
  --statistics Sum
```

## Scaling and Performance

### Automatic Scaling

Lambda automatically scales based on request volume:
- **Cold Starts**: ~200-500ms for first request
- **Warm Requests**: ~5-50ms response time
- **Concurrent Executions**: Up to 1000 by default

### DynamoDB Scaling

- **Billing Mode**: Pay-per-request (auto-scaling)
- **Read/Write Capacity**: Automatic based on usage
- **TTL**: Automatic cleanup of expired pastes

### Performance Optimization

1. **Binary Size**: Optimized with `-ldflags="-s -w"`
2. **Memory**: 512MB (adjustable in template)
3. **Timeout**: 30 seconds (adjustable)
4. **Architecture**: x86_64 (ARM64 available)

## Security

### IAM Permissions

The Lambda function has minimal permissions:
- DynamoDB: CRUD operations on the pastes table only
- CloudWatch: Log creation and writing
- No network access beyond AWS services

### Network Security

- **VPC**: Not required (DynamoDB is accessed via IAM)
- **Security Groups**: Not applicable
- **WAF**: Consider adding for production workloads

### Data Security

- **Encryption at Rest**: DynamoDB automatic encryption
- **Encryption in Transit**: HTTPS only via API Gateway
- **TTL**: Automatic data cleanup

## Cost Optimization

### Estimated Costs (us-east-1)

| Component | Usage | Monthly Cost |
|-----------|-------|--------------|
| Lambda | 1M requests, 128MB | ~$0.20 |
| API Gateway | 1M requests | ~$3.50 |
| DynamoDB | 1M reads/writes | ~$1.25 |
| CloudWatch | Standard logs | ~$0.50 |
| **Total** | | **~$5.45** |

### Cost Reduction Tips

1. **Reduce Memory**: Lower to 128MB if sufficient
2. **Optimize Code**: Reduce execution time
3. **DynamoDB**: Use on-demand billing for variable loads
4. **CloudWatch**: Reduce log retention period

## Troubleshooting

### Common Issues

#### 1. Deployment Fails

```bash
# Check SAM CLI version
sam --version

# Validate template
sam validate

# Check AWS credentials
aws sts get-caller-identity
```

#### 2. Function Errors

```bash
# Check function logs
sam logs --stack-name nclip-staging --tail

# Test function locally
sam local invoke -e events/test-event.json
```

#### 3. API Gateway Issues

```bash
# Test API Gateway
curl -v https://api-id.execute-api.region.amazonaws.com/Prod/health

# Check API Gateway logs
aws logs describe-log-groups --log-group-name-prefix API-Gateway-Execution-Logs
```

### Debug Mode

Enable debug logging:

```bash
# Deploy with debug logging
sam deploy --parameter-overrides "Environment=staging LogLevel=debug"
```

## Cleanup

### Delete Stack

```bash
# Delete the stack
sam delete --stack-name nclip-staging

# Confirm DynamoDB table deletion
aws dynamodb list-tables | grep nclip
```

### Cleanup Build Artifacts

```bash
# Remove build artifacts
rm -rf dist/
rm -rf .aws-sam/
```

## Advanced Configuration

### Custom Domain Setup

1. **Get SSL Certificate**: Use AWS Certificate Manager
2. **Create Domain**: Update CloudFormation template
3. **DNS Configuration**: Point domain to API Gateway

```yaml
# Add to template.yml
Parameters:
  SSLCertificateArn:
    Type: String
    Description: SSL certificate ARN for custom domain
```

### Blue/Green Deployment

```bash
# Deploy to different stage
sam deploy --stack-name nclip-staging-v2 --parameter-overrides Environment=staging-v2

# Switch traffic using Route 53 weighted routing
```

### Multi-Region Deployment

```bash
# Deploy to multiple regions
./deploy.sh staging us-east-1
./deploy.sh staging us-west-2
./deploy.sh staging eu-west-1
```

## Support

For issues and questions:

1. Check the [troubleshooting section](#troubleshooting)
2. Review AWS CloudWatch logs
3. Open an issue in the project repository
