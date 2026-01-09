package auth

// BasicLoginTemplate is the HTML template for the Basic Auth login page
// Styled to match the UnifiedLoginView.vue design
const BasicLoginTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sign In - {{.Realm}}</title>
  <style>
    :root {
      --bg-deep: #0a0a0b;
      --bg-surface: #111113;
      --bg-elevated: #19191b;
      --text-primary: #fafafa;
      --text-secondary: #a1a1a6;
      --text-muted: #5c5c66;
      --border-subtle: #232328;
      --border-accent: #2d2d35;
      --accent-primary: #6ee7b7;
      --accent-primary-rgb: 110, 231, 183;
      --accent-primary-dim: #5dd4a6;
      --accent-red: #f87171;
      --accent-red-rgb: 248, 113, 113;
    }

    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: var(--bg-deep);
      color: var(--text-primary);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 2rem;
      position: relative;
      overflow: hidden;
    }

    /* Background pattern */
    .bg-pattern {
      position: fixed;
      inset: 0;
      pointer-events: none;
      z-index: 0;
      opacity: 0.12;
      background-image:
        linear-gradient(var(--border-subtle) 1px, transparent 1px),
        linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
      background-size: 60px 60px;
    }

    .bg-gradient {
      position: fixed;
      inset: 0;
      pointer-events: none;
      z-index: 0;
      background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-primary-rgb), 0.08) 0%, transparent 50%),
                  radial-gradient(ellipse at 70% 80%, rgba(var(--accent-primary-rgb), 0.05) 0%, transparent 50%);
    }

    .container {
      position: relative;
      z-index: 10;
      width: 100%;
      max-width: 400px;
      animation: fadeIn 0.4s ease-out;
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(10px); }
      to { opacity: 1; transform: translateY(0); }
    }

    /* Logo section */
    .logo-section {
      text-align: center;
      margin-bottom: 2.5rem;
    }

    .logo {
      position: relative;
      width: 56px;
      height: 56px;
      margin: 0 auto 1.25rem;
      border: 2px solid var(--accent-primary);
      border-radius: 14px;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .logo-inner {
      width: 20px;
      height: 20px;
      background: var(--accent-primary);
      border-radius: 4px;
      transform: rotate(45deg);
    }

    .logo-ring {
      position: absolute;
      inset: -4px;
      border: 1px solid var(--accent-primary);
      border-radius: 18px;
      opacity: 0.3;
    }

    .brand-title {
      font-size: 2rem;
      font-weight: 600;
      letter-spacing: -0.02em;
      margin-bottom: 0.25rem;
    }

    .brand-subtitle {
      font-size: 0.875rem;
      color: var(--text-secondary);
    }

    /* Card */
    .card {
      background: var(--bg-surface);
      border: 1px solid var(--border-subtle);
      border-radius: 16px;
      overflow: hidden;
      position: relative;
    }

    .card-accent {
      position: absolute;
      top: 0;
      left: 32px;
      right: 32px;
      height: 2px;
      background: linear-gradient(90deg, transparent, var(--accent-primary), transparent);
    }

    .card-header {
      padding: 1.5rem 1.5rem 0;
    }

    .card-title {
      font-size: 1.25rem;
      font-weight: 600;
      margin-bottom: 0.25rem;
    }

    .card-description {
      font-size: 0.875rem;
      color: var(--text-secondary);
    }

    /* Subdomain badge */
    .subdomain-badge {
      display: inline-flex;
      align-items: center;
      gap: 0.375rem;
      padding: 0.375rem 0.625rem;
      background: rgba(var(--accent-primary-rgb), 0.15);
      color: var(--accent-primary);
      border-radius: 6px;
      font-size: 0.75rem;
      font-weight: 500;
      margin-top: 1rem;
    }

    .subdomain-badge svg {
      width: 14px;
      height: 14px;
    }

    /* Form */
    .form {
      padding: 1.5rem;
    }

    .error-message {
      display: flex;
      align-items: flex-start;
      gap: 0.625rem;
      padding: 0.875rem 1rem;
      background: rgba(var(--accent-red-rgb), 0.1);
      border: 1px solid rgba(var(--accent-red-rgb), 0.3);
      border-radius: 8px;
      margin-bottom: 1.25rem;
      font-size: 0.875rem;
      color: var(--accent-red);
      animation: shake 0.4s ease-out;
    }

    @keyframes shake {
      0%, 100% { transform: translateX(0); }
      25% { transform: translateX(-5px); }
      75% { transform: translateX(5px); }
    }

    .error-message svg {
      width: 16px;
      height: 16px;
      flex-shrink: 0;
      margin-top: 1px;
    }

    .field {
      margin-bottom: 1rem;
    }

    .field:last-of-type {
      margin-bottom: 1.5rem;
    }

    label {
      display: block;
      font-size: 0.75rem;
      font-weight: 500;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-secondary);
      margin-bottom: 0.625rem;
    }

    .input-wrapper {
      position: relative;
    }

    .input-icon {
      position: absolute;
      left: 1rem;
      top: 50%;
      transform: translateY(-50%);
      width: 18px;
      height: 18px;
      color: var(--text-muted);
      pointer-events: none;
    }

    input {
      width: 100%;
      padding: 0.875rem 1rem 0.875rem 2.75rem;
      background: var(--bg-deep);
      border: 1px solid var(--border-subtle);
      border-radius: 8px;
      font-family: inherit;
      font-size: 0.9375rem;
      color: var(--text-primary);
      transition: all 0.2s;
    }

    input::placeholder {
      color: var(--text-muted);
    }

    input:focus {
      outline: none;
      border-color: var(--accent-primary);
      box-shadow: 0 0 0 3px rgba(var(--accent-primary-rgb), 0.12);
    }

    input:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .submit-btn {
      width: 100%;
      padding: 0.9375rem 1.5rem;
      background: var(--accent-primary);
      color: var(--bg-deep);
      border: none;
      border-radius: 8px;
      font-family: inherit;
      font-size: 0.9375rem;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.2s;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
    }

    .submit-btn:hover:not(:disabled) {
      background: var(--accent-primary-dim);
      transform: translateY(-1px);
    }

    .submit-btn:disabled {
      opacity: 0.7;
      cursor: not-allowed;
      transform: none;
    }

    .submit-btn svg {
      width: 16px;
      height: 16px;
    }

    .spinner {
      width: 16px;
      height: 16px;
      border: 2px solid transparent;
      border-top-color: currentColor;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    /* Footer */
    .footer {
      text-align: center;
      margin-top: 1.5rem;
      font-size: 0.75rem;
      color: var(--text-muted);
    }

    .footer a {
      color: var(--accent-primary);
      text-decoration: none;
      transition: color 0.2s;
    }

    .footer a:hover {
      color: var(--text-primary);
      text-decoration: underline;
    }

    /* Hidden return URL */
    .hidden {
      display: none;
    }
  </style>
</head>
<body>
  <div class="bg-pattern"></div>
  <div class="bg-gradient"></div>

  <div class="container">
    <!-- Logo -->
    <div class="logo-section">
      <div class="logo">
        <div class="logo-inner"></div>
        <div class="logo-ring"></div>
      </div>
      <h1 class="brand-title">digit-link</h1>
      <p class="brand-subtitle">Secure Tunnel Infrastructure</p>
    </div>

    <!-- Login Card -->
    <div class="card">
      <div class="card-accent"></div>

      <div class="card-header">
        <h2 class="card-title">Authentication Required</h2>
        <p class="card-description">Enter your credentials to access this resource</p>
        <div class="subdomain-badge">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          </svg>
          <span>{{.Subdomain}}</span>
        </div>
      </div>

      <form class="form" method="POST" action="{{.LoginURL}}" id="loginForm">
        {{if .Error}}
        <div class="error-message">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          <span>{{.Error}}</span>
        </div>
        {{end}}

        <div class="field">
          <label for="username">Username</label>
          <div class="input-wrapper">
            <svg class="input-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <input
              type="text"
              id="username"
              name="username"
              placeholder="Enter your username"
              autocomplete="username"
              required
              autofocus
              value="{{.Username}}"
            />
          </div>
        </div>

        <div class="field">
          <label for="password">Password</label>
          <div class="input-wrapper">
            <svg class="input-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            <input
              type="password"
              id="password"
              name="password"
              placeholder="Enter your password"
              autocomplete="current-password"
              required
            />
          </div>
        </div>

        <input type="hidden" name="return" value="{{.ReturnURL}}" />
        <input type="hidden" name="subdomain" value="{{.Subdomain}}" />

        <button type="submit" class="submit-btn" id="submitBtn">
          <span id="btnText">Sign In</span>
          <svg id="btnArrow" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="5" y1="12" x2="19" y2="12"/>
            <polyline points="12 5 19 12 12 19"/>
          </svg>
        </button>
      </form>
    </div>

    <!-- Footer -->
    <div class="footer">
      <p>Secure infrastructure by <a href="https://digit.zone" target="_blank" rel="noopener">digit.zone</a></p>
    </div>
  </div>

  <script>
    const form = document.getElementById('loginForm');
    const btn = document.getElementById('submitBtn');
    const btnText = document.getElementById('btnText');
    const btnArrow = document.getElementById('btnArrow');

    form.addEventListener('submit', function() {
      btn.disabled = true;
      btnText.textContent = 'Signing in...';
      btnArrow.outerHTML = '<div class="spinner"></div>';
    });
  </script>
</body>
</html>`
