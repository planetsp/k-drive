# AWS Configuration for K-Drive

To use K-Drive with AWS S3, you need to configure your AWS credentials and region.

## Method 1: AWS Credentials File
Create `~/.aws/credentials`:
```
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
```

Create `~/.aws/config`:
```
[default]
region = us-east-1
```

## Method 2: Environment Variables
```bash
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY
export AWS_DEFAULT_REGION=us-east-1
```

## Method 3: AWS CLI
If you have AWS CLI installed:
```bash
aws configure
```

## Testing
Once configured, you can test the application:
```bash
go run cmd/kdrive/main.go
```

If AWS is not configured properly, the application will gracefully disable cloud sync and show appropriate error messages instead of crashing.
