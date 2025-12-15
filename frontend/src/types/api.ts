// API response types for digit-link admin panel

export interface Stats {
  activeTunnels: number
  activeAccounts: number
  whitelistEntries: number
  totalTunnels: number
  totalBytesSent?: number
  totalBytesReceived?: number
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
}

export interface AccountsResponse {
  accounts: Account[]
}

export interface CreateAccountRequest {
  username: string
  isAdmin: boolean
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
  autoWhitelist: boolean
}

export interface SetupInitResponse {
  success?: boolean
  token?: string
  error?: string
}

export interface ApiError {
  error: string
  error_description?: string
}
