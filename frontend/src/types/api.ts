// API response types for digit-link admin panel

export interface Stats {
  activeTunnels: number
  activeAccounts: number
  whitelistEntries: number
  totalTunnels: number
  totalBytesSent?: number
  totalBytesReceived?: number
  applicationCount?: number
  totalConnections?: number
}

export interface Tunnel {
  id: string
  subdomain: string
  url: string
  accountId: string
  createdAt: string
}

export interface TunnelsResponse {
  active: Tunnel[]
}

export interface Account {
  id: string
  username: string
  isAdmin: boolean
  active: boolean
  createdAt: string
  lastUsed?: string
  orgId?: string
  orgName?: string
  hasPassword?: boolean
}

export interface AccountsResponse {
  accounts: Account[]
}

export interface CreateAccountRequest {
  username: string
  password?: string
  isAdmin: boolean
  orgId?: string
}

export interface CreateAccountResponse {
  success: boolean
  token?: string
  error?: string
}

export interface RegenerateTokenResponse {
  success: boolean
  token?: string
  error?: string
}

export interface SetAccountOrgRequest {
  orgId: string // Empty string to unlink
}

export interface SetAccountOrgResponse {
  success: boolean
  orgId?: string
  orgName?: string
  error?: string
}

export interface SetAccountPasswordRequest {
  password: string
}

export interface SetAccountPasswordResponse {
  success: boolean
  error?: string
}

export interface WhitelistEntry {
  id: string
  ipRange: string
  description?: string
  createdAt: string
}

export interface WhitelistResponse {
  entries: WhitelistEntry[]
}

export interface AddWhitelistRequest {
  ipRange: string
  description?: string
}

export interface AddWhitelistResponse {
  success: boolean
  error?: string
}

export interface DeleteResponse {
  success: boolean
  error?: string
}

export interface SetupStatusResponse {
  needsSetup: boolean
}

export interface SetupInitRequest {
  username: string
  password: string
  autoWhitelist: boolean
}

export interface SetupInitResponse {
  success: boolean
  pendingToken?: string
  accountId?: string
  username?: string
  error?: string
}

export interface SetupTOTPResponse {
  success: boolean
  secret?: string
  url?: string
  error?: string
}

export interface SetupCompleteRequest {
  pendingToken: string
  code: string
}

export interface SetupCompleteResponse {
  success: boolean
  token?: string
  error?: string
}

export interface ApiError {
  error: string
  error_description?: string
}

// ============================================
// Organizations
// ============================================

export interface Organization {
  id: string
  name: string
  createdAt: string
  appCount?: number
  hasPolicy?: boolean
}

export interface OrganizationsResponse {
  organizations: Organization[]
}

export interface CreateOrganizationRequest {
  name: string
}

export interface CreateOrganizationResponse {
  success: boolean
  organization?: Organization
  error?: string
}

// ============================================
// Applications
// ============================================

export type AuthMode = 'inherit' | 'disabled' | 'custom'
export type AuthType = 'basic' | 'api_key' | 'oidc'

export interface TunnelStats {
  totalConnections: number
  activeCount: number
  bytesSent: number
  bytesReceived: number
}

export interface Application {
  id: string
  orgId: string
  orgName?: string
  subdomain: string
  name: string
  authMode: AuthMode
  authType?: AuthType
  createdAt: string
  hasPolicy?: boolean
  isActive?: boolean
  activeTunnelCount?: number
  stats?: TunnelStats
}

export interface ApplicationsResponse {
  applications: Application[]
}

export interface CreateApplicationRequest {
  orgId: string
  subdomain: string
  name: string
}

export interface CreateApplicationResponse {
  success: boolean
  application?: Application
  error?: string
}

export interface UpdateApplicationRequest {
  name: string
  authMode: AuthMode
  authType?: AuthType
  subdomain?: string
}

// ============================================
// Auth Policies
// ============================================

export interface OrgAuthPolicy {
  orgId: string
  authType: AuthType
  oidcIssuerUrl?: string
  oidcClientId?: string
  oidcScopes?: string[]
  oidcAllowedDomains?: string[]
  oidcRequiredClaims?: Record<string, string>
}

export interface AppAuthPolicy {
  appId: string
  authType: AuthType
  oidcIssuerUrl?: string
  oidcClientId?: string
  oidcScopes?: string[]
  oidcAllowedDomains?: string[]
  oidcRequiredClaims?: Record<string, string>
}

export interface PolicyResponse {
  policy: OrgAuthPolicy | AppAuthPolicy | null
}

export interface SetPolicyRequest {
  authType: AuthType
  basicUsername?: string
  basicPassword?: string
  oidcIssuerUrl?: string
  oidcClientId?: string
  oidcClientSecret?: string
  oidcScopes?: string[]
  oidcAllowedDomains?: string[]
  oidcRequiredClaims?: Record<string, string>
}

// ============================================
// API Keys
// ============================================

export interface APIKey {
  id: string
  orgId?: string
  appId?: string
  keyPrefix: string
  description: string
  createdAt: string
  lastUsed?: string
  expiresAt?: string
}

export interface APIKeysResponse {
  keys: APIKey[]
}

export interface CreateAPIKeyRequest {
  orgId: string
  appId?: string
  description: string
  expiresIn?: number // days
}

export interface CreateAPIKeyResponse {
  success: boolean
  key?: APIKey
  rawKey?: string // Only returned once at creation
  error?: string
}

// ============================================
// Audit Log
// ============================================

export interface AuditEvent {
  id: string
  timestamp: string
  orgId?: string
  appId?: string
  authType: string
  success: boolean
  failureReason?: string
  sourceIp: string
  userIdentity?: string
  keyId?: string
}

export interface AuditEventsResponse {
  events: AuditEvent[]
  total: number
  limit: number
  offset: number
}

export interface AuthStats {
  totalAttempts: number
  successCount: number
  failureCount: number
  uniqueIps: number
  failuresToday: number
}
