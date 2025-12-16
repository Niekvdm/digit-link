import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Theme {
  id: string
  name: string
  description: string
  colors: {
    primary: string
    secondary: string
    background: string
  }
}

export const THEMES: Theme[] = [
  {
    id: 'copper-dark',
    name: 'Copper Dark',
    description: 'Warm industrial',
    colors: {
      primary: '#c9956c',
      secondary: '#4a9f7e',
      background: '#1a1a1a'
    }
  },
  {
    id: 'midnight-sapphire',
    name: 'Midnight Sapphire',
    description: 'Cool professional',
    colors: {
      primary: '#6b8cce',
      secondary: '#5ab3a8',
      background: '#0f1419'
    }
  },
  {
    id: 'forest-moss',
    name: 'Forest Moss',
    description: 'Organic natural',
    colors: {
      primary: '#7fae7a',
      secondary: '#c9957a',
      background: '#141816'
    }
  },
  {
    id: 'sunset-rose',
    name: 'Sunset Rose',
    description: 'Warm elegant',
    colors: {
      primary: '#d4847a',
      secondary: '#c9a86c',
      background: '#1a1717'
    }
  }
]

const STORAGE_KEY = 'digit-link-theme'

export const useThemeStore = defineStore('theme', () => {
  const currentThemeId = ref<string>(localStorage.getItem(STORAGE_KEY) || 'copper-dark')

  const currentTheme = computed(() => 
    THEMES.find(t => t.id === currentThemeId.value) || THEMES[0]
  )

  const themes = computed(() => THEMES)

  function setTheme(themeId: string) {
    const theme = THEMES.find(t => t.id === themeId)
    if (!theme) return

    document.documentElement.dataset.theme = themeId
    localStorage.setItem(STORAGE_KEY, themeId)
    currentThemeId.value = themeId
  }

  function initTheme() {
    const savedTheme = localStorage.getItem(STORAGE_KEY)
    const themeId = savedTheme && THEMES.find(t => t.id === savedTheme) 
      ? savedTheme 
      : 'copper-dark'
    
    document.documentElement.dataset.theme = themeId
    currentThemeId.value = themeId
  }

  return {
    currentThemeId,
    currentTheme,
    themes,
    setTheme,
    initTheme
  }
})
