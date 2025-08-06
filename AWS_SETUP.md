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

## S3 Bucket Setup
1. **Create a bucket** in your AWS console or using AWS CLI:
   ```bash
   aws s3 mb s3://your-unique-bucket-name --region us-east-1
   ```

2. **Update conf.json** with your bucket name:
   ```json
   {
     "bucketName": "your-unique-bucket-name"
   }
   ```

## Troubleshooting Common Issues

### PermanentRedirect Error (301)
If you see "PermanentRedirect" error, your bucket exists in a different region:

**Solution 1: Update your region**
```bash
# Find your bucket's region
aws s3api get-bucket-location --bucket your-bucket-name

# Update ~/.aws/config with the correct region
[default]
region = eu-west-1  # Use the region from above
```

**Solution 2: Use a bucket in your current region**
```bash
# Create a new bucket in your current region
aws s3 mb s3://new-bucket-name --region us-east-1

# Update conf.json with the new bucket name
```

### NoSuchBucket Error
The bucket doesn't exist. Create it or use an existing bucket name.

### Credentials Error
Double-check your AWS access keys and ensure they have S3 permissions.

## Testing
Once configured, you can test the application:
```bash
go run cmd/kdrive/main.go
```

If AWS is not configured properly, the application will gracefully disable cloud sync and show appropriate error messages instead of crashing.
