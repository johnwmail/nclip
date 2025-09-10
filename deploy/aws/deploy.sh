#!/bin/bash
set -euo pipefail

# Deploy nclip to AWS Lambda using SAM
# Usage: ./deploy.sh [environment] [region] [domain]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Configuration
ENVIRONMENT="${1:-staging}"
AWS_REGION="${2:-us-east-1}"
DOMAIN="${3:-}"
STACK_NAME="nclip-${ENVIRONMENT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

# Validate dependencies
check_dependencies() {
    log "Checking dependencies..."
    
    if ! command -v aws &> /dev/null; then
        error "AWS CLI is not installed. Please install it first."
        exit 1
    fi
    
    if ! command -v sam &> /dev/null; then
        error "SAM CLI is not installed. Please install it first."
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        error "Go is not installed. Please install it first."
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        error "AWS credentials not configured. Please run 'aws configure'."
        exit 1
    fi
    
    success "All dependencies available"
}

# Build the Lambda binary
build_lambda() {
    log "Building Lambda binary..."
    
    cd "$PROJECT_ROOT"
    
    # Create dist directory
    mkdir -p dist
    
    # Build Lambda binary for AL2023
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
        -ldflags="-s -w" \
        -o dist/bootstrap \
        ./cmd/lambda
    
    # Create deployment package
    cd dist
    zip -q lambda.zip bootstrap
    rm bootstrap
    
    success "Lambda binary built and packaged"
}

# Deploy using SAM
deploy_stack() {
    log "Deploying stack: $STACK_NAME to region: $AWS_REGION"
    
    cd "$SCRIPT_DIR"
    
    # SAM deploy parameters
    local sam_params=(
        "--stack-name" "$STACK_NAME"
        "--region" "$AWS_REGION"
        "--capabilities" "CAPABILITY_IAM"
        "--no-fail-on-empty-changeset"
        "--parameter-overrides" "Environment=$ENVIRONMENT"
    )
    
    # Add domain if provided
    if [[ -n "$DOMAIN" ]]; then
        sam_params+=("Domain=$DOMAIN")
    fi
    
    # Deploy
    if sam deploy "${sam_params[@]}"; then
        success "Stack deployed successfully"
    else
        error "Stack deployment failed"
        exit 1
    fi
}

# Get stack outputs
get_outputs() {
    log "Getting stack outputs..."
    
    local api_url
    api_url=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" \
        --output text)
    
    local function_name
    function_name=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        --query "Stacks[0].Outputs[?OutputKey=='FunctionName'].OutputValue" \
        --output text)
    
    local table_name
    table_name=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        --query "Stacks[0].Outputs[?OutputKey=='TableName'].OutputValue" \
        --output text)
    
    echo
    success "Deployment completed!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${GREEN}Stack Name:${NC}     $STACK_NAME"
    echo -e "${GREEN}Region:${NC}         $AWS_REGION"
    echo -e "${GREEN}Environment:${NC}    $ENVIRONMENT"
    echo -e "${GREEN}API URL:${NC}        $api_url"
    echo -e "${GREEN}Function:${NC}       $function_name"
    echo -e "${GREEN}DynamoDB:${NC}       $table_name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    echo -e "${BLUE}Test the deployment:${NC}"
    echo "  curl -X POST $api_url -d 'Hello from AWS Lambda!'"
    echo "  curl $api_url/\$(echo 'Hello' | curl -s -X POST $api_url -d @-)"
    echo
    echo -e "${BLUE}Monitor logs:${NC}"
    echo "  sam logs --stack-name $STACK_NAME --region $AWS_REGION --tail"
    echo
}

# Test the deployment
test_deployment() {
    log "Testing deployment..."
    
    local api_url
    api_url=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --region "$AWS_REGION" \
        --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" \
        --output text)
    
    # Test health endpoint
    if curl -f -s "$api_url/health" > /dev/null; then
        success "Health check passed"
    else
        warn "Health check failed"
    fi
    
    # Test paste creation and retrieval
    local test_data="Test from deployment script $(date)"
    local paste_id
    paste_id=$(curl -s -X POST "$api_url" -d "$test_data" | jq -r '.id // empty')
    
    if [[ -n "$paste_id" ]]; then
        local retrieved_data
        retrieved_data=$(curl -s "$api_url/$paste_id")
        
        if [[ "$retrieved_data" == "$test_data" ]]; then
            success "Paste creation and retrieval test passed"
        else
            warn "Paste retrieval test failed"
        fi
    else
        warn "Paste creation test failed"
    fi
}

# Cleanup on exit
cleanup() {
    cd "$PROJECT_ROOT"
}
trap cleanup EXIT

# Main execution
main() {
    echo "nclip AWS Lambda Deployment Script"
    echo "=================================="
    echo
    
    check_dependencies
    build_lambda
    deploy_stack
    get_outputs
    
    # Run tests if not in CI
    if [[ "${CI:-false}" != "true" ]]; then
        test_deployment
    fi
}

# Show help
show_help() {
    cat << EOF
nclip AWS Lambda Deployment Script

Usage: $0 [environment] [region] [domain]

Arguments:
  environment    Environment name (staging|production) [default: staging]
  region         AWS region [default: us-east-1]
  domain         Custom domain name (optional)

Examples:
  $0                                    # Deploy to staging in us-east-1
  $0 production us-west-2               # Deploy to production in us-west-2
  $0 staging us-east-1 api.example.com  # Deploy with custom domain

Environment Variables:
  AWS_PROFILE    AWS profile to use
  AWS_REGION     AWS region (overridden by argument)
  CI             Set to 'true' to skip interactive tests

Dependencies:
  - AWS CLI v2
  - SAM CLI
  - Go 1.19+
  - jq (for testing)

EOF
}

# Handle arguments
case "${1:-}" in
    -h|--help|help)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
