# digit-link Frontend Architecture

## Overview

The digit-link frontend is a Vue 3 Single Page Application (SPA) built with TypeScript, providing both an admin dashboard and organization portal interfaces.

## Technology Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| Vue | 3.x | UI framework |
| TypeScript | 5.x | Type safety |
| Vite | 5.x | Build tool |
| Pinia | 2.x | State management |
| Vue Router | 4.x | Client-side routing |

## Project Structure

```
frontend/
├── src/
│   ├── App.vue                    # Root component
│   ├── main.ts                    # Entry point
│   ├── style.css                  # Global styles
│   │
│   ├── components/
│   │   ├── layout/
│   │   │   ├── DynamicLayout.vue  # Layout wrapper
│   │   │   └── PortalShell.vue    # Main navigation shell
│   │   │
│   │   ├── shared/
│   │   │   ├── index.ts           # Shared exports
│   │   │   └── ThemeSwitcher.vue  # Theme toggle
│   │   │
│   │   └── ui/
│   │       ├── ConfirmDialog.vue  # Confirmation modal
│   │       ├── DataTable.vue      # Generic data table
│   │       ├── EmptyState.vue     # Empty content placeholder
│   │       ├── LoadingSpinner.vue # Loading indicator
│   │       ├── Modal.vue          # Base modal component
│   │       ├── PageHeader.vue     # Page title/actions
│   │       ├── Pagination.vue     # Table pagination
│   │       ├── PolicyEditor.vue   # Auth policy form
│   │       ├── SearchInput.vue    # Search field
│   │       ├── StatCard.vue       # Dashboard stat card
│   │       ├── StatusBadge.vue    # Status indicator
│   │       ├── Toast.vue          # Notification toast
│   │       └── TokenReveal.vue    # Token display with copy
│   │
│   ├── composables/
│   │   ├── api/
│   │   │   ├── index.ts           # API exports
│   │   │   ├── useAccounts.ts     # Account CRUD
│   │   │   ├── useAPIKeys.ts      # API key management
│   │   │   ├── useApplications.ts # Application CRUD
│   │   │   ├── useAuditLogs.ts    # Audit log queries
│   │   │   ├── useOrgAccounts.ts  # Org account management
│   │   │   ├── useOrganizations.ts# Organization CRUD
│   │   │   ├── useStats.ts        # Statistics queries
│   │   │   ├── useTunnels.ts      # Tunnel queries
│   │   │   └── useWhitelist.ts    # Whitelist management
│   │   │
│   │   ├── index.ts               # Composables exports
│   │   ├── useApi.ts              # Base API client
│   │   ├── useFormatters.ts       # Data formatters
│   │   ├── usePortalContext.ts    # Admin/Org context
│   │   └── useToast.ts            # Toast notifications
│   │
│   ├── router/
│   │   └── index.ts               # Route definitions
│   │
│   ├── stores/
│   │   ├── authStore.ts           # Authentication state
│   │   └── themeStore.ts          # Theme preferences
│   │
│   ├── styles/
│   │   └── themes/
│   │       ├── copper-dark.css    # Theme variant
│   │       ├── forest-moss.css    # Theme variant
│   │       ├── midnight-sapphire.css # Theme variant
│   │       └── sunset-rose.css    # Theme variant
│   │
│   ├── types/
│   │   ├── api.ts                 # API type definitions
│   │   └── index.ts               # Type exports
│   │
│   └── views/
│       ├── admin/                 # Admin dashboard views
│       │   ├── AccountDetailPage.vue
│       │   ├── AccountsPage.vue
│       │   ├── APIKeysPage.vue
│       │   ├── ApplicationDetailPage.vue
│       │   ├── ApplicationsPage.vue
│       │   ├── AuditPage.vue
│       │   ├── DashboardPage.vue
│       │   ├── MyAccountPage.vue
│       │   ├── OrganizationsPage.vue
│       │   ├── TunnelsPage.vue
│       │   └── WhitelistPage.vue
│       │
│       ├── org/                   # Organization portal views
│       │   ├── AccountDetailPage.vue
│       │   ├── AccountsPage.vue
│       │   ├── APIKeysPage.vue
│       │   ├── ApplicationDetailPage.vue
│       │   ├── ApplicationsPage.vue
│       │   ├── DashboardPage.vue
│       │   ├── MyAccountPage.vue
│       │   ├── SettingsPage.vue
│       │   └── WhitelistPage.vue
│       │
│       ├── SetupView.vue          # First-boot setup
│       └── UnifiedLoginView.vue   # Login page
│
├── index.html                     # HTML template
├── package.json                   # Dependencies
├── tsconfig.json                  # TypeScript config
├── tsconfig.app.json              # App-specific TS config
├── tsconfig.node.json             # Node TS config
├── vite.config.ts                 # Vite configuration
└── yarn.lock                      # Dependency lock
```

## State Management

### Auth Store (`authStore.ts`)

```typescript
interface AuthState {
  token: string | null
  userType: 'admin' | 'org'
  orgId: string | null
  orgName: string | null
  username: string | null
  isOrgAdminState: boolean
}

// Key computed properties
isAuthenticated: boolean    // !!token
isAdmin: boolean           // userType === 'admin'
isOrgUser: boolean         // userType === 'org'
isOrgAdmin: boolean        // isOrgUser && isOrgAdminState
```

**Storage:** localStorage with keys:
- `digit-link-token`
- `digit-link-user-type`
- `digit-link-org-id`
- `digit-link-org-name`
- `digit-link-username`
- `digit-link-is-org-admin`

### Theme Store (`themeStore.ts`)

Manages theme preferences with CSS custom properties.

## API Integration

### Base API Client (`useApi.ts`)

```typescript
function useApi() {
  async function request<T>(endpoint: string, options?: RequestOptions): Promise<T>
  function get<T>(endpoint: string): Promise<T>
  function post<T>(endpoint: string, body?: unknown): Promise<T>
  function put<T>(endpoint: string, body?: unknown): Promise<T>
  function del<T>(endpoint: string): Promise<T>
}
```

**Authentication handling:**
- Admin endpoints: `X-Admin-Token: <token>`
- Org endpoints: `Authorization: Bearer <token>`
- Automatic 401 handling → redirect to login

### Domain-Specific Composables

| Composable | Endpoints | Description |
|------------|-----------|-------------|
| `useAccounts` | `/admin/accounts/*` | Account CRUD operations |
| `useOrganizations` | `/admin/organizations/*` | Organization management |
| `useApplications` | `/admin/applications/*` | Application management |
| `useAPIKeys` | `/admin/api-keys/*` | API key management |
| `useWhitelist` | `/admin/whitelist/*` | IP whitelist management |
| `useTunnels` | `/admin/tunnels/*` | Tunnel queries |
| `useAuditLogs` | `/admin/audit/*` | Audit log queries |
| `useStats` | `/admin/stats` | Statistics |
| `useOrgAccounts` | `/org/accounts/*` | Org-scoped account ops |

## Routing

### Route Structure

```typescript
// Public routes
/login          → UnifiedLoginView
/setup          → SetupView

// Admin routes (requires isAdmin)
/admin          → DashboardPage
/admin/accounts → AccountsPage
/admin/accounts/:id → AccountDetailPage
/admin/organizations → OrganizationsPage
/admin/applications → ApplicationsPage
/admin/applications/:id → ApplicationDetailPage
/admin/tunnels  → TunnelsPage
/admin/whitelist → WhitelistPage
/admin/api-keys → APIKeysPage
/admin/audit    → AuditPage
/admin/my-account → MyAccountPage

// Org routes (requires isOrgUser)
/org            → DashboardPage
/org/accounts   → AccountsPage
/org/applications → ApplicationsPage
/org/whitelist  → WhitelistPage
/org/api-keys   → APIKeysPage
/org/settings   → SettingsPage
/org/my-account → MyAccountPage
```

### Route Guards

```typescript
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  
  // Check authentication
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return next({ name: 'login' })
  }
  
  // Check admin access
  if (to.path.startsWith('/admin') && !authStore.isAdmin) {
    return next({ name: 'login' })
  }
  
  // Check org access
  if (to.path.startsWith('/org') && !authStore.isOrgUser) {
    return next({ name: 'login' })
  }
  
  next()
})
```

## UI Components

### DataTable

Generic data table with sorting, pagination, and actions.

```vue
<DataTable
  :columns="columns"
  :data="items"
  :loading="loading"
  :sortable="true"
  @row-click="handleRowClick"
>
  <template #actions="{ row }">
    <button @click="edit(row)">Edit</button>
  </template>
</DataTable>
```

### Modal

Base modal component with customizable content.

```vue
<Modal :show="showModal" @close="showModal = false">
  <template #header>Modal Title</template>
  <template #body>Content here</template>
  <template #footer>
    <button @click="confirm">Confirm</button>
  </template>
</Modal>
```

### Toast

Notification system via composable.

```typescript
const { showToast } = useToast()

showToast({
  type: 'success',  // success | error | warning | info
  message: 'Operation completed',
  duration: 3000
})
```

### PolicyEditor

Auth policy configuration form.

```vue
<PolicyEditor
  v-model="policy"
  :auth-types="['basic', 'api_key', 'oidc']"
  @save="savePolicy"
/>
```

## Theming

### CSS Custom Properties

```css
:root {
  --bg-primary: #ffffff;
  --bg-secondary: #f8f9fa;
  --text-primary: #212529;
  --text-secondary: #6c757d;
  --accent-primary: #0d6efd;
  --border-color: #dee2e6;
  /* ... more properties */
}
```

### Available Themes

1. **Midnight Sapphire** (default dark)
2. **Copper Dark**
3. **Forest Moss**
4. **Sunset Rose**

### Theme Switching

```typescript
const themeStore = useThemeStore()
themeStore.setTheme('midnight-sapphire')
```

## Build & Deploy

### Development

```bash
cd frontend
yarn install
yarn dev
```

### Production Build

```bash
yarn build
# Output: dist/
```

### Embedding in Go Binary

The frontend is embedded into the Go binary:

```go
//go:embed public/*
var staticFS embed.FS

func getStaticFile(path string) ([]byte, string, bool) {
    // Serve from embedded filesystem
}
```

## Best Practices

### Component Guidelines

1. **Single Responsibility** - Each component does one thing well
2. **Props Down, Events Up** - Use props for data, events for actions
3. **Composition API** - Use `<script setup>` for all components
4. **Type Safety** - Define props and emits with TypeScript

### API Integration

1. **Use composables** - Don't call fetch directly in components
2. **Handle loading states** - Show loading indicators
3. **Handle errors** - Show user-friendly error messages
4. **Optimistic updates** - Update UI before API confirms

### State Management

1. **Minimal store state** - Only store what needs to persist
2. **Computed properties** - Derive values from state
3. **Actions for async** - Handle API calls in store actions

## Security Considerations

### Token Storage

⚠️ **Current Implementation:** Tokens stored in localStorage.

**Risk:** XSS attacks can steal tokens.

**Mitigations:**
- CSP headers prevent inline scripts
- Input sanitization
- Consider moving to HttpOnly cookies

### CSRF Protection

- SameSite cookie policy
- Origin validation on server

### Input Validation

- Client-side validation for UX
- Server-side validation for security
- Never trust client-side validation alone
