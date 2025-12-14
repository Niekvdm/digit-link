# digit-link Setup Guide

This guide covers setting up digit-link with authentication enabled.

## Initial Server Setup

### 1. Build the Server

```bash
make build-server
```

### 2. Create the Initial Admin Account

Run the server with the `--setup-admin` flag:

```bash
./build/bin/digit-link-server --setup-admin
```

This will output:

```
================================================================================
                         ADMIN ACCOUNT CREATED
================================================================================
Username: admin
Token:    abc123...xyz789
--------------------------------------------------------------------------------
IMPORTANT: Save this token securely. It will not be shown again!
Use this token to access the admin dashboard and API.
================================================================================
```

**Save this token immediately!** It's the only way to access the admin dashboard.

### 3. Start the Server

```bash
# Set environment variables (optional)
export DOMAIN=tunnel.yourdomain.com
export PORT=8080
export DB_PATH=data/digit-link.db

# Start the server
./build/bin/digit-link-server
```

### Alternative: Auto-Create Admin on Startup

Instead of using `--setup-admin`, you can set the `ADMIN_TOKEN` environment variable:

```bash
export ADMIN_TOKEN=your-secure-token-here
./build/bin/digit-link-server
```

The server will automatically create an admin account with this token if no admin exists.

## Admin Dashboard Setup

### 1. Access the Dashboard

Navigate to your server domain in a browser:

```
https://tunnel.yourdomain.com/
```

### 2. Login

Enter your admin token to access the dashboard.

### 3. Add IP Addresses to Whitelist

Before any clients can connect, you must whitelist their IP addresses:

1. Go to **Whitelist** in the navigation
2. Enter an IP address or CIDR range (e.g., `192.168.1.0/24`)
3. Add a description (optional)
4. Click **Add**

### 4. Create User Accounts

1. Go to **Accounts** in the navigation
2. Click **Create Account**
3. Enter a username
4. Check "Make this an admin account" if needed
5. Click **Create**
6. **Copy the generated token** and share it with the user securely

## Client Setup

### 1. Build the Client

```bash
make build-client
```

### 2. Get Your Token

Ask your administrator for:
- Your authentication token
- Confirmation that your IP is whitelisted

### 3. Connect

```bash
./build/bin/digit-link \
  --server tunnel.yourdomain.com \
  --subdomain myapp \
  --port 3000 \
  --token YOUR_TOKEN
```

Or use an environment variable:

```bash
export DIGIT_LINK_TOKEN=YOUR_TOKEN
./build/bin/digit-link --server tunnel.yourdomain.com --subdomain myapp --port 3000
```

### 4. Verify Connection

On successful connection, you'll see:

```
Connected! Public URL: https://myapp.tunnel.yourdomain.com
Forwarding to localhost:3000
```

## Troubleshooting

### "Authentication required: provide a valid token"

- Ensure you're using the `--token` flag with a valid token
- Check that your account is active in the admin dashboard

### "IP address not whitelisted"

- Ask your administrator to add your IP to the whitelist
- Check your current public IP at https://whatismyip.com
- Remember that your IP may change if you're on a dynamic connection

### "Invalid token"

- Verify the token is correct (no extra spaces)
- The account may have been deactivated
- Ask admin to regenerate your token

### Can't access admin dashboard

- Ensure you saved the initial admin token
- Try creating a new admin via `ADMIN_TOKEN` environment variable
- Check the database file permissions

## Database Management

The SQLite database is stored at the path specified by `DB_PATH` (default: `data/digit-link.db`).

### Backup

```bash
cp data/digit-link.db data/digit-link.db.backup
```

### Reset

To start fresh, delete the database file:

```bash
rm data/digit-link.db
./build/bin/digit-link-server --setup-admin
```

## Production Deployment

### Kubernetes/k3s

1. Create a secret for the admin token:

```bash
kubectl create secret generic digit-link-secrets \
  --from-literal=admin-token=your-secure-token
```

2. Configure the deployment environment variables:

```yaml
env:
  - name: ADMIN_TOKEN
    valueFrom:
      secretKeyRef:
        name: digit-link-secrets
        key: admin-token
  - name: DOMAIN
    value: tunnel.yourdomain.com
  - name: DB_PATH
    value: /data/digit-link.db
```

3. Mount a persistent volume for the database:

```yaml
volumes:
  - name: data
    persistentVolumeClaim:
      claimName: digit-link-data
volumeMounts:
  - name: data
    mountPath: /data
```

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e DOMAIN=tunnel.yourdomain.com \
  -e ADMIN_TOKEN=your-secure-token \
  -v digit-link-data:/data \
  digit-link-server
```

## Security Best Practices

1. **Use HTTPS**: Always serve the server behind an HTTPS-terminating proxy
2. **Protect Admin Token**: Never commit tokens to version control
3. **Rotate Tokens**: Periodically regenerate user tokens
4. **Audit Whitelist**: Regularly review whitelisted IPs
5. **Monitor Tunnels**: Check the dashboard for unexpected connections
6. **Backup Database**: Regularly backup the SQLite database
