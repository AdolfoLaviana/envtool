# EnvTool

A comprehensive environment file (.env) management utility written in Go.

## Features

EnvTool provides a complete solution for managing environment variable files through a command line interface:

### Core Features

1. **Validation (.env format checking)**
   - Validates proper KEY=VALUE format
   - Detects duplicate keys
   - Identifies missing values compared to a base file

2. **Comparison (diff between .env files)**
   - Compares variables present in one file but not the other
   - Identifies variables with the same name but different values
   - Provides automatic merging of files

3. **Encryption and Decryption**
   - Encrypts sensitive values using AES encryption
   - Uses environment variable or local key for encryption
   - Compatible with CI/CD workflows

4. **Remote Synchronization**
   - Upload/download to multiple storage options (S3, Firebase, Git)
   - Maintain synchronized .env versions across environments and teams
   - Backup files automatically during operations

### Additional Features

5. **Template-based Generation**
   - Generate .env files from schema definitions
   - Create files with default values or placeholders
   - Maintain consistency across projects

6. **Change Auditing**
   - Track history of changes to .env files
   - Identify who/what modified values
   - Secure hash verification

7. **Linting and Formatting**
   - Suggest better variable naming
   - Fix inconsistently named keys
   - Organize variables by section or alphabetically

## Installation

### Prerequisites

- Go 1.18 or later

### Building from Source

1. Clone the repository:
   ```
   git clone https://github.com/adolfoc/envtool.git
   cd envtool
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Build the application:
   ```
   go build
   ```

4. Add to your PATH (optional):
   - On Windows: Copy the executable to a location in your PATH or add the current directory to your PATH
   - On Linux/macOS: `sudo cp envtool /usr/local/bin/`

### Running the Application

- Run `./envtool [command] [flags]` or just `envtool [command] [flags]` if added to PATH

## Detailed Usage Guide

### Getting Help

To see all available commands and their descriptions:

```
envtool help
```

### Validation

The validation feature helps ensure your .env files are correctly formatted and contain all required values.

#### Basic Validation

Validate the syntax and format of your .env file:

```
envtool validate path/to/.env
```

Output example:
```
✅ .env file is valid
```

Or with errors:
```
❌ Invalid format lines:
  Line 3: DEBUG=TRUE= (Invalid format, should be KEY=VALUE)
  Line 7: APIKEY (Missing value)

⚠️ Duplicate keys:
  Key 'APP_ENV' appears at lines [1, 10]
```

#### Comparing Against a Base File

Check if your .env file contains all the required keys from a base/template file:

```
envtool validate path/to/.env --base path/to/.env.example
```

Output example:
```
⚠️ Missing keys from base file:
  DATABASE_PASSWORD
  REDIS_URL
```

#### Advanced Validation Options

Check for empty values:
```
envtool validate path/to/.env --check-empty
```

Strict validation mode (fails on any issues):
```
envtool validate path/to/.env --strict
```

### File Comparison

Compare two .env files to identify differences.

#### Basic Comparison

```
envtool diff path/to/.env.development path/to/.env.production
```

Output example:
```
🔍 Variables only in .env.development:
  DEBUG=true
  LOG_LEVEL=debug

🔍 Variables only in .env.production:
  CACHE_TTL=3600
  REDIS_POOL_SIZE=20

🔄 Variables with different values:
  APP_ENV:
    - development
    + production
  LOG_VERBOSITY:
    - verbose
    + error
```

#### Merging Files

Combine two .env files, resolving conflicts:

```
envtool diff path/to/.env.base path/to/.env.overrides --merge path/to/.env.merged
```

By default, values from the second file take precedence. To reverse this:

```
envtool diff path/to/.env.base path/to/.env.overrides --merge path/to/.env.merged --prefer-first
```

#### Format-Preserving Comparison

Compare while preserving comments and formatting:

```
envtool diff path/to/.env.development path/to/.env.production --preserve-format
```

### Encryption & Decryption

Protect sensitive information in your .env files.

#### Setting Up an Encryption Key

Set up a persistent encryption key:

```
# Windows
set ENVCIPHER_KEY=your-secure-key-here

# Linux/macOS
export ENVCIPHER_KEY=your-secure-key-here
```

Or generate a new random key:

```
envtool encrypt --generate-key
```

Output:
```
✅ Generated new encryption key: BTAl6JkhmnNxJpYGzwrpLfEO8c7Cd+FY=
Set ENVCIPHER_KEY environment variable to use this key.
```

#### Encrypting a File

```
envtool encrypt path/to/.env --out path/to/.env.enc
```

To encrypt only specific values (keeping others as plaintext):

```
envtool encrypt path/to/.env --out path/to/.env.enc --only "DATABASE_PASSWORD,API_KEY,JWT_SECRET"
```

#### Decrypting a File

```
envtool decrypt path/to/.env.enc --out path/to/.env
```

### Remote Synchronization

Share and maintain consistent .env files across environments and team members.

#### Configuring Remote Providers

**AWS S3:**
```
envtool push path/to/.env --env production --provider s3 --bucket my-env-bucket --region us-west-2
```

**Firebase:**
```
envtool push path/to/.env --env staging --provider firebase --project my-project-id --path configs
```

**Git:**
```
envtool push path/to/.env --env dev --provider git --repo https://github.com/username/config.git --branch env-files
```

#### Pushing Files

Push a local .env file to a remote location:

```
envtool push path/to/.env --env production
```

With encryption:
```
envtool push path/to/.env --env production --encrypt
```

#### Pulling Files

Download a remote .env file:

```
envtool pull --env staging --out path/to/.env.staging
```

With decryption:
```
envtool pull --env production --out path/to/.env.production --decrypt
```

### Template Generation

Generate consistent .env files from schema templates.

#### Creating a Sample Schema

Generate a template schema file:

```
envtool generate --create-schema path/to/.env.schema.json
```

#### Generating an .env File from Schema

```
envtool generate --schema path/to/.env.schema.json --out path/to/.env
```

With interactive mode to prompt for values:
```
envtool generate --schema path/to/.env.schema.json --out path/to/.env --interactive
```

#### Converting Existing .env to Schema

```
envtool generate --from path/to/.env --schema path/to/new-schema.json
```

### Change Auditing

Track and manage changes to your .env files over time.

#### Viewing Change History

```
envtool audit path/to/.env
```

Output example:
```
Change History for .env:

[2025-04-30 15:42:33] User: john.doe
  + Added: REDIS_URL=redis://localhost:6379
  ~ Changed: DEBUG from true to false
  
[2025-04-29 10:15:22] User: deploy-bot
  + Added: API_VERSION=2.1.0
  - Removed: LEGACY_SUPPORT
```

#### Detailed Change Information

View detailed information about specific changes:

```
envtool audit path/to/.env --detailed
```

### Linting and Formatting

Maintain consistent naming conventions and formatting in your .env files.

#### Basic Linting

Check for naming convention issues:

```
envtool lint path/to/.env
```

Output example:
```
⚠️ Linting issues found:
  Line 5: DatabaseUrl - Should use SNAKE_CASE (suggested: DATABASE_URL)
  Line 7: apiKey - Should use SNAKE_CASE (suggested: API_KEY)
  Line 12: MAX_CONNECTIONS - Missing value
```

#### Automatic Fixing

Fix naming convention issues automatically:

```
envtool lint path/to/.env --fix
```

#### Advanced Linting Options

Apply specific conventions:

```
envtool lint path/to/.env --convention snake_case
envtool lint path/to/.env --convention camelCase
envtool lint path/to/.env --convention PascalCase
```

Sort variables alphabetically:

```
envtool lint path/to/.env --fix --sort
```

Group variables by sections (requires section comments in the file):

```
envtool lint path/to/.env --fix --group-sections
```

## Configuration

### Encryption Key

To set an encryption key, use the `ENVCIPHER_KEY` environment variable:

```
export ENVCIPHER_KEY=your-base64-encoded-key
```

### Remote Synchronization

Configure remote storage providers using environment variables:

#### S3
```
ENVTOOL_SYNC_PROVIDER=s3
ENVTOOL_S3_BUCKET=your-bucket-name
ENVTOOL_S3_REGION=us-east-1
ENVTOOL_AWS_ACCESS_KEY=your-access-key  # Optional if using AWS CLI configured credentials
ENVTOOL_AWS_SECRET_KEY=your-secret-key  # Optional if using AWS CLI configured credentials
```

#### Firebase
```
ENVTOOL_SYNC_PROVIDER=firebase
ENVTOOL_FIREBASE_PROJECT=your-project-id
ENVTOOL_FIREBASE_PATH=envtool
ENVTOOL_FIREBASE_CREDENTIALS=path/to/credentials.json  # Optional if using Firebase CLI login
```

#### Git
```
ENVTOOL_SYNC_PROVIDER=git
ENVTOOL_GIT_REPO=https://github.com/username/repo.git
ENVTOOL_GIT_BRANCH=main
ENVTOOL_GIT_USERNAME=your-username  # For private repositories
ENVTOOL_GIT_PASSWORD=your-password  # For private repositories, consider using a token instead
```

## Common Use Cases

### CI/CD Pipeline Integration

#### Encrypting Sensitive Values in CI
```bash
# In your CI script
export ENVCIPHER_KEY=$(cat secrets/env_key.txt)
envtool encrypt .env.template --out .env.production --only "DB_PASSWORD,API_KEYS,SECRETS"
```

#### Generating Environment-Specific Files
```bash
# Generate staging environment
envtool generate --schema env.schema.json --out .env.staging --set "APP_ENV=staging,DEBUG=true"

# Generate production environment
envtool generate --schema env.schema.json --out .env.production --set "APP_ENV=production,DEBUG=false"
```

#### Validation Before Deployment
```bash
# Fail pipeline if .env is invalid
envtool validate .env.production --strict || exit 1
```

### Team Collaboration

#### Sharing Standard Environment Setup
```bash
# Developer onboarding
envtool pull --env development --out .env.development
```

#### Updating Team Environment Files
```bash
# After adding new required variables
envtool generate --schema env.schema.json --out .env.template
envtool diff .env.template .env --merge .env.updated --preserve-existing
```

### Multi-Environment Management

#### Synchronizing Across Environments
```bash
# Check for differences between environments
envtool diff .env.staging .env.production

# Push staging environment to remote
envtool push .env.staging --env staging
```

#### Rotating Secrets
```bash
# Generate new secrets and update environment file
NEW_API_KEY=$(openssl rand -base64 32)
envtool generate --from .env --out .env.new --set "API_KEY=$NEW_API_KEY"
```

## Schema Format

EnvTool uses a JSON schema format for generating .env files. Example:

```json
{
  "version": "1.0",
  "name": "Sample Application Configuration",
  "description": "Configuration for a sample application",
  "variables": {
    "APP_NAME": {
      "description": "Name of the application",
      "type": "string",
      "default": "My Application",
      "required": true
    },
    "DEBUG": {
      "description": "Enable debug mode",
      "type": "boolean",
      "default": true,
      "required": false
    },
    "PORT": {
      "description": "Port number for the application server",
      "type": "number",
      "default": 3000,
      "required": true
    },
    "DATABASE_URL": {
      "description": "Database connection string",
      "type": "string",
      "format": "uri",
      "secret": true,
      "required": true
    },
    "API_KEYS": {
      "description": "Comma-separated list of API keys",
      "type": "array",
      "items": {
        "type": "string"
      },
      "default": [],
      "secret": true
    },
    "CACHE_TTL": {
      "description": "Cache time-to-live in seconds",
      "type": "number",
      "default": 300,
      "minimum": 0,
      "maximum": 86400
    }
  },
  "sections": [
    {
      "name": "Application Settings",
      "description": "Basic application configuration",
      "variables": ["APP_NAME", "DEBUG", "PORT"]
    },
    {
      "name": "External Services",
      "description": "Configuration for external services",
      "variables": ["DATABASE_URL", "API_KEYS", "CACHE_TTL"]
    }
  ]
}
```

### Schema Properties

| Field | Description |
|-------|-------------|
| version | Schema version |
| name | Name of the configuration |
| description | Description of the configuration |
| variables | Object containing variable definitions |
| sections | Array of sections for organizing variables |

### Variable Properties

| Field | Description |
|-------|-------------|
| description | Description of the variable |
| type | Data type: string, number, boolean, or array |
| default | Default value when not specified |
| required | Whether the variable is required |
| secret | Whether the variable should be treated as sensitive |
| format | Special format (e.g., uri, email) |
| minimum | Minimum value for number type |
| maximum | Maximum value for number type |
| pattern | Regex pattern for validation |
| enum | Array of allowed values |

## Development

### Project Structure

```
envtool/
├── main.go                  # Main entry point
├── internal/                # Internal packages
│   ├── validator/           # Validation functionality
│   ├── differ/              # Comparison functionality
│   ├── crypto/              # Encryption functionality
│   ├── sync/                # Remote sync functionality
│   ├── generator/           # Template generation
│   ├── audit/               # Change auditing
│   └── linter/              # Linting functionality
└── pkg/                     # Reusable packages
```

### Adding New Features

1. Create a new package in the appropriate directory
2. Update the main.go file to include the new feature
3. Add tests to ensure functionality works as expected
4. Update documentation as needed

## Troubleshooting

### Common Issues

#### Permission Errors

If you encounter permission errors when writing files:

```
Error: failed to write to file: permission denied
```

Make sure your user account has write permissions to the target directory.

#### Remote Sync Issues

For issues with remote synchronization:

```
Error: failed to connect to remote provider
```

Check your network connection and verify the provider credentials are correct.

#### Encryption/Decryption Problems

For issues with encryption or decryption:

```
Error: failed to decrypt file: invalid key
```

Ensure the `ENVCIPHER_KEY` environment variable is set correctly and matches the key used for encryption.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests to ensure they pass
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

# Adolfo Laviana Lora