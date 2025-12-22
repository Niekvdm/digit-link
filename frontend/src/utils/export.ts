/**
 * Export utility functions for CSV and JSON downloads
 * Uses PapaParse for CSV conversion and native Blob API for file downloads
 */

import Papa from 'papaparse'

export interface ExportOptions {
  /** Custom filename (without extension) */
  filename?: string
  /** Include timestamp in filename for uniqueness */
  includeTimestamp?: boolean
}

/**
 * Trigger a file download in the browser
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/**
 * Generate filename with optional timestamp
 */
function generateFilename(
  baseName: string,
  extension: string,
  includeTimestamp: boolean
): string {
  if (includeTimestamp) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    return `${baseName}-${timestamp}.${extension}`
  }
  return `${baseName}.${extension}`
}

/**
 * Export data to CSV file using PapaParse
 *
 * @param data - Array of objects to export
 * @param options - Export options (filename, timestamp)
 *
 * @example
 * exportToCSV(usageSnapshots, { filename: 'usage-export', includeTimestamp: true })
 */
export function exportToCSV<T extends Record<string, unknown>>(
  data: T[],
  options: ExportOptions = {}
): void {
  const {
    filename = 'export',
    includeTimestamp = true
  } = options

  if (!data || data.length === 0) {
    throw new Error('No data to export')
  }

  const csv = Papa.unparse(data)
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const fullFilename = generateFilename(filename, 'csv', includeTimestamp)

  downloadBlob(blob, fullFilename)
}

/**
 * Export data to JSON file
 *
 * @param data - Array of objects to export
 * @param options - Export options (filename, timestamp)
 *
 * @example
 * exportToJSON(usageSnapshots, { filename: 'usage-export', includeTimestamp: true })
 */
export function exportToJSON<T>(
  data: T[],
  options: ExportOptions = {}
): void {
  const {
    filename = 'export',
    includeTimestamp = true
  } = options

  if (!data || data.length === 0) {
    throw new Error('No data to export')
  }

  const json = JSON.stringify(data, null, 2)
  const blob = new Blob([json], { type: 'application/json' })
  const fullFilename = generateFilename(filename, 'json', includeTimestamp)

  downloadBlob(blob, fullFilename)
}

/**
 * Export data to specified format
 *
 * @param data - Array of objects to export
 * @param format - Export format ('csv' or 'json')
 * @param options - Export options (filename, timestamp)
 */
export function exportData<T extends Record<string, unknown>>(
  data: T[],
  format: 'csv' | 'json',
  options: ExportOptions = {}
): void {
  if (format === 'csv') {
    exportToCSV(data, options)
  } else {
    exportToJSON(data, options)
  }
}
