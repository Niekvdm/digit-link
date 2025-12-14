# digit-link Setup Guide

This guide covers setting up digit-link with authentication enabled.

## Initial Server Setup

### 1. Build the Server

```bash
make build-server
```

### 2. Create the Initial Admin Account

There are three ways to create the initial admin account:

#### Option A: Web-Based First-Boot Setup (Recommended for k3s/Kubernetes)

Simply start the server and navigate to your domain. If no admin account exists, you'll be automatically redirected to a setup wizard:

1. Deploy and start the server
2. Navigate to `https://tunnel.yourdomain.com/`
3. The first-boot setup page will appear
4. Choose a username (default: "admin")
5. Optionally auto-whitelist your current IP
6. Click "Create Admin Account"
7. **Save the generated token immediately** - it will only be shown once!

This is the preferred method for containerized deployments where CLI access is limited.

#### Option B: Command Line Setup

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

#### Option C: Environment Variable

Set the `ADMIN_TOKEN` environment variable before starting:

```bash
export ADMIN_TOKEN=your-secure-token-here
./build/bin/digit-link-server
```

The server will automatically create an admin account with this token if no admin exists.

### 3. Start the Server

```bash
# Set environment variables (optional)
export DOMAIN=tunnel.yourdomain.com
export PORT=8080
export DB_PATH=data/digit-link.db

# Start the server
./build/bin/digit-link-server
```

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

### First-boot setup not appearing

- Verify no admin account exists in the database
- To reset, delete the database file and restart the server
- Check browser console for JavaScript errors
- Ensure you're accessing the main domain (not a subdomain)

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

The recommended setup for Kubernetes uses the **first-boot web wizard**:

1. Deploy the server with a persistent volume (no admin token needed):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: digit-link
spec:
  replicas: 1
  selector:
    matchLabels:
      app: digit-link
  template:
    metadata:
      labels:
        app: digit-link
    spec:
      containers:
        - name: digit-link
          image: digit-link-server:latest
          ports:
            - containerPort: 8080
          env:
            - name: DOMAIN
              value: tunnel.yourdomain.com
            - name: DB_PATH
              value: /data/digit-link.db
          volumeMounts:
            - name: data
              mountPath: /data
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: digit-link-data
```

2. Create a PersistentVolumeClaim for the database:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: digit-link-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

3. Create a Service and Ingress:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: digit-link
spec:
  selector:
    app: digit-link
  ports:
    - port: 80
      targetPort: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: digit-link
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
spec:
  tls:
    - hosts:
        - tunnel.yourdomain.com
        - "*.tunnel.yourdomain.com"
      secretName: digit-link-tls
  rules:
    - host: tunnel.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: digit-link
                port:
                  number: 80
```

4. **First-Boot Setup**: Navigate to `https://tunnel.yourdomain.com/` and complete the setup wizard. Your admin token will be generated and displayed - save it securely!

#### Alternative: Pre-configured Admin Token

If you prefer to set the admin token via environment variable:

```yaml
env:
  - name: ADMIN_TOKEN
    valueFrom:
      secretKeyRef:
        name: digit-link-secrets
        key: admin-token
```

Create the secret first:

```bash
kubectl create secret generic digit-link-secrets \
  --from-literal=admin-token=your-secure-token
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
