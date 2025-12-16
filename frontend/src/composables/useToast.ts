import { ref, readonly } from 'vue'

export interface ToastOptions {
  type?: 'success' | 'error' | 'warning' | 'info'
  duration?: number
}

interface ToastItem {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
}

const toasts = ref<ToastItem[]>([])

let idCounter = 0

function show(message: string, options: ToastOptions = {}) {
  const id = `toast-${++idCounter}`
  const type = options.type || 'info'
  const duration = options.duration ?? 5000

  const toast: ToastItem = { id, type, message }
  toasts.value.push(toast)

  if (duration > 0) {
    setTimeout(() => {
      dismiss(id)
    }, duration)
  }

  return id
}

function dismiss(id: string) {
  const index = toasts.value.findIndex(t => t.id === id)
  if (index !== -1) {
    toasts.value.splice(index, 1)
  }
}

function success(message: string, duration?: number) {
  return show(message, { type: 'success', duration })
}

function error(message: string, duration?: number) {
  return show(message, { type: 'error', duration: duration ?? 8000 })
}

function warning(message: string, duration?: number) {
  return show(message, { type: 'warning', duration })
}

function info(message: string, duration?: number) {
  return show(message, { type: 'info', duration })
}

export function useToast() {
  return {
    toasts: readonly(toasts),
    show,
    dismiss,
    success,
    error,
    warning,
    info
  }
}
