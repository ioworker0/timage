# timage

A lightweight Docker image management CLI tool with zero dependency on local Docker installation.

## Features

- **Pull & Push Images**: Download and upload images from/to any Docker registry v2 compliant registry
- **Tag Management**: Tag local images with different names
- **Image Removal**: Remove local images
- **Registry Authentication**: Login to private registries (Harbor, Docker Hub, etc.)
- **List Images**: View all locally stored images
- **Proxy Support**: HTTP/HTTPS/SOCKS5 proxy support for pulling images through firewalls
- **Multi-Architecture**: Automatic handling of manifest lists and OCI image indexes
- **Progress Display**: Visual progress bars during image operations

## Installation

### Build from source

```bash
go build -o timage
```

### Install

```bash
sudo mv timage /usr/local/bin/
```

## Usage

### Pull an image

```bash
# Pull from Docker Hub
./timage pull busybox:latest

# Pull from private registry
./timage pull harbor.example.com/project/image:v1.0
```

### Push an image

```bash
# Tag first
./timage tag busybox:latest harbor.example.com/project/image:v1.0

# Then push
./timage push harbor.example.com/project/image:v1.0
```

### Tag an image

```bash
./timage tag source-image:latest target-image:v1.0
```

### List local images

```bash
./timage list
```

### Remove an image

```bash
./timage rm harbor.example.com/project/image:v1.0
```

### Login to a registry

```bash
# Login to Harbor
./timage login harbor.example.com -u username -p password

# Login to Docker Hub
./timage login docker.io -u username -p password
```

### Using proxy

```bash
# Using flag
./timage pull busybox:latest -x http://127.0.0.1:7890

# Using SOCKS5 proxy
./timage pull busybox:latest -x socks5://127.0.0.1:1080

# Or set environment variable
export TIMAGE_PROXY=http://127.0.0.1:7890
./timage pull busybox:latest
```

## Configuration

Credentials and configurations are stored in `~/.timage/`:

```
~/.timage/
├── config.json      # Registry credentials and proxy settings
└── images/          # Local image storage
    └── image_name/
        ├── manifest.json
        ├── config.json
        └── layers/
            └── sha256:...
```

## Registry Support

- Docker Hub
- Harbor
- Any Docker Registry v2 compliant registry

## Authentication

timage supports both Basic authentication and Bearer token authentication:
- Private registries like Harbor typically use Basic auth
- Docker Hub uses Bearer token authentication

## Multi-Architecture Images

When pulling multi-architecture images, timage automatically:
1. Detects manifest lists and OCI image indexes
2. Selects the appropriate manifest for your system (amd64/linux)
3. Downloads the correct layers for your platform

## Storage Format

Images are stored in a simple directory structure:
- Manifests are stored in Docker manifest v2 schema 2 format
- Configs and layers are stored as separate files
- No deduplication (simpler implementation)

## Limitations

- Does not support running containers (only image management)
- No image deduplication
- Manifest schema 2 only

## Examples

### Complete workflow

```bash
# 1. Login to your private registry
./timage login harbor.example.com -u myuser -p mypass

# 2. Pull an image from Docker Hub
./timage pull nginx:latest

# 3. Tag it for your registry
./timage tag nginx:latest harbor.example.com/project/nginx:v1.0

# 4. Push to your registry
./timage push harbor.example.com/project/nginx:v1.0

# 5. List local images
./timage list

# 6. Remove when done
./timage rm nginx:latest
```

### Using with proxy

```bash
# Set proxy globally
export TIMAGE_PROXY=http://proxy.example.com:8080

# Pull through proxy
./timage pull busybox:latest

# Push through proxy
./timage push harbor.example.com/project/image:tag
```

## Troubleshooting

### Authentication failures

Make sure you're using the correct username and password for your registry:
- Harbor: Use your Harbor username/password
- Docker Hub: Use your Docker Hub username/password

### Proxy issues

If proxy doesn't work, try:
1. Check if the proxy URL is correct
2. Ensure the proxy is running and accessible
3. Try without proxy to confirm the issue

### Pull errors

- Verify the image name and tag are correct
- Check your network connection
- Ensure the registry is accessible

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
