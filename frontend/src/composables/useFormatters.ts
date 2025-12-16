/**
 * Shared formatting utilities used across all views
 * Replaces duplicated formatDate, formatBytes, formatDuration functions
 */
export function useFormatters() {
  /**
   * Format a timestamp to a localized date string
   */
  function formatDate(timestamp?: string): string {
    if (!timestamp) return ''
    return new Date(timestamp).toLocaleDateString()
  }

  /**
   * Format a timestamp to a localized date and time string
   */
  function formatDateTime(timestamp?: string): string {
    if (!timestamp) return 'Never'
    return new Date(timestamp).toLocaleString()
  }

  /**
   * Format bytes to human-readable size (B, KB, MB, GB, TB)
   */
  function formatBytes(bytes?: number): string {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  /**
   * Format a timestamp to relative duration (e.g., "5m ago", "2h ago")
   */
  function formatDuration(timestamp: string): string {
    if (!timestamp) return 'Unknown'
    const start = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - start.getTime()

    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) {
      const hours = Math.floor(diff / 3600000)
      const mins = Math.floor((diff % 3600000) / 60000)
      return `${hours}h ${mins}m`
    }
    return `${Math.floor(diff / 86400000)}d ago`
  }

  /**
   * Format a timestamp to relative time for display (e.g., "Just now", "5m ago")
   */
  function formatRelativeTime(timestamp: string): string {
    if (!timestamp) return ''
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    
    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
    return date.toLocaleDateString()
  }

  return {
    formatDate,
    formatDateTime,
    formatBytes,
    formatDuration,
    formatRelativeTime
  }
}
