# AWS Lambda Serverless Deployment Guide

This guide shows how to deploy nclip as a serverless application on AWS Lambda.

## 🏗️ Architecture Overview

```
Internet → API Gateway → Lambda Function → DynamoDB
                                      ↓
                                   CloudWatch Logs
```

## 📋 Prerequisites

1. AWS CLI configured with appropriate permissions
2. Go 1.21+ installed
3. AWS SAM CLI (optional, for easier deployment)

## 🗄️ Storage Setup

### Option 1: DynamoDB (Recommended)

#### Create DynamoDB Table

```bash
aws dynamodb create-table \
    --table-name nclip-pastes \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --region us-east-1
```

#### Enable TTL for Auto-Expiration

```bash
aws dynamodb update-time-to-live \
    --table-name nclip-pastes \
    --time-to-live-specification Enabled=true,AttributeName=expires_at \
    --region us-east-1
```

### Option 2: S3 with Lifecycle (Alternative)

```bash
# Create S3 bucket
aws s3 mb s3://your-nclip-bucket

# Configure lifecycle rule for 1-day expiration
aws s3api put-bucket-lifecycle-configuration \
    --bucket your-nclip-bucket \
    --lifecycle-configuration file://s3-lifecycle.json
```

**s3-lifecycle.json:**
```json
{
    "Rules": [
        {
            "ID": "PasteExpiration",
            "Status": "Enabled",
            "Filter": {"Prefix": "pastes/"},
            "Expiration": {"Days": 1}
        }
    ]
}
```

## 🔧 Lambda Configuration

### Environment Variables

```bash
# For DynamoDB storage
NCLIP_STORAGE_TYPE=dynamodb
NCLIP_DYNAMODB_TABLE=nclip-pastes
NCLIP_EXPIRE_DAYS=1
NCLIP_DOMAIN=your-api-gateway-domain.amazonaws.com

# For S3 storage (alternative)
NCLIP_STORAGE_TYPE=s3
NCLIP_S3_BUCKET=your-nclip-bucket
NCLIP_EXPIRE_DAYS=1
```

### Lambda Function Settings

- **Runtime**: Custom runtime (Go 1.x)
- **Memory**: 128-512 MB (depending on paste size limits)
- **Timeout**: 30 seconds
- **Architecture**: x86_64 or arm64

## 📦 Deployment Methods

### Method 1: AWS SAM Template

**template.yaml:**
```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31

Globals:
  Function:
    Timeout: 30
    MemorySize: 256
    Runtime: provided.al2
    Architectures:
      - x86_64

Resources:
  GoNclipFunction:
    Type: AWS::Serverless::Function
    Properties:
      CodeUri: ./
      Handler: bootstrap
      Environment:
        Variables:
          NCLIP_STORAGE_TYPE: dynamodb
          NCLIP_DYNAMODB_TABLE: !Ref NclipTable
          NCLIP_EXPIRE_DAYS: 1
      Events:
        CatchAll:
          Type: Api
          Properties:
            Path: /{proxy+}
            Method: ANY
        Root:
          Type: Api
          Properties:
            Path: /
            Method: ANY
      Policies:
        - DynamoDBCrudPolicy:
            TableName: !Ref NclipTable

  NclipTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: nclip-pastes
      AttributeDefinitions:
        - AttributeName: id
          AttributeType: S
      KeySchema:
        - AttributeName: id
          KeyType: HASH
      BillingMode: PAY_PER_REQUEST
      TimeToLiveSpecification:
        AttributeName: expires_at
        Enabled: true

Outputs:
  GoNclipAPI:
    Description: "API Gateway endpoint URL"
    Value: !Sub "https://${ServerlessRestApi}.execute-api.${AWS::Region}.amazonaws.com/Prod/"
```

**Deploy with SAM:**
```bash
# Build
sam build

# Deploy
sam deploy --guided
```

### Method 2: Manual Deployment

```bash
# Build for Lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap ./cmd/lambda

# Create deployment package
zip lambda-deployment.zip bootstrap

# Create/update Lambda function
aws lambda create-function \
    --function-name nclip \
    --runtime provided.al2 \
    --role arn:aws:iam::ACCOUNT:role/lambda-execution-role \
    --handler bootstrap \
    --zip-file fileb://lambda-deployment.zip \
    --environment Variables='{
        "NCLIP_STORAGE_TYPE":"dynamodb",
        "NCLIP_DYNAMODB_TABLE":"nclip-pastes",
        "NCLIP_EXPIRE_DAYS":"1"
    }'
```

## 🔄 Lambda Adapter Code

You'll need to create a Lambda entry point. Here's a template:

**cmd/lambda/main.go:**
```go
package main

import (
    "context"
    "log"
    
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
    
    "github.com/nclip/nclip/internal/config"
    "github.com/nclip/nclip/internal/server"
    "github.com/nclip/nclip/internal/storage"
)

var httpLambda *httpadapter.HandlerAdapter

func init() {
    // Load configuration
    cfg, err := config.LoadFromEnvironment()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize storage (DynamoDB or S3)
    var store storage.Storage
    switch cfg.StorageType {
    case "dynamodb":
        store = storage.NewDynamoDBStorage(cfg.DynamoDBTable, logger)
    default:
        store = storage.NewFilesystemStorage(cfg.OutputDir, logger)
    }
    
    // Create HTTP server
    httpServer := server.NewHTTPServer(cfg, store, nil, logger)
    
    // Create Lambda adapter
    httpLambda = httpadapter.New(httpServer.Handler())
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    return httpLambda.ProxyWithContext(ctx, req)
}

func main() {
    lambda.Start(Handler)
}
```

## 💰 Cost Estimation

### DynamoDB Costs (Pay-per-request)
- **Storage**: ~$0.25/GB/month
- **Read requests**: $0.25 per million
- **Write requests**: $1.25 per million

### Lambda Costs
- **Requests**: $0.20 per 1M requests
- **Duration**: $0.0000166667 per GB-second

### Example Monthly Cost (1000 pastes/day)
- Lambda: ~$0.50
- DynamoDB: ~$2.00  
- **Total: ~$2.50/month**

## 🚀 Production Checklist

- [ ] Set up CloudWatch monitoring
- [ ] Configure API Gateway throttling
- [ ] Set up custom domain
- [ ] Enable CloudWatch Logs
- [ ] Configure CORS if needed
- [ ] Set up backup for DynamoDB (if critical)
- [ ] Monitor Lambda cold starts
- [ ] Set up alarms for errors

## 🔧 Required Dependencies

Add these to your `go.mod` for Lambda deployment:

```bash
go get github.com/aws/aws-lambda-go
go get github.com/awslabs/aws-lambda-go-api-proxy
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
```

This setup gives you:
✅ Automatic scaling
✅ Pay-per-use pricing  
✅ Automatic paste expiration
✅ High availability
✅ Minimal maintenance
