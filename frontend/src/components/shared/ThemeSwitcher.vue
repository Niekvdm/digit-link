<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useThemeStore } from '@/stores/themeStore'
import { Palette, Check } from 'lucide-vue-next'

const themeStore = useThemeStore()

const isOpen = ref(false)
const openAbove = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)

const themes = computed(() => themeStore.themes)
const currentTheme = computed(() => themeStore.currentTheme)

async function toggleDropdown() {
  if (!isOpen.value) {
    // Calculate position before opening
    await nextTick()
    calculatePosition()
  }
  isOpen.value = !isOpen.value
}

function calculatePosition() {
  if (!triggerRef.value) return
  
  const triggerRect = triggerRef.value.getBoundingClientRect()
  const dropdownHeight = 280 // Approximate height of dropdown
  const spaceBelow = window.innerHeight - triggerRect.bottom
  const spaceAbove = triggerRect.top
  
  // Open above if not enough space below and more space above
  openAbove.value = spaceBelow < dropdownHeight && spaceAbove > spaceBelow
}

function selectTheme(themeId: string) {
  themeStore.setTheme(themeId)
  isOpen.value = false
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="dropdownRef" class="theme-switcher">
    <button 
      ref="triggerRef"
      class="theme-trigger"
      :class="{ 'theme-trigger--active': isOpen }"
      @click="toggleDropdown"
      aria-label="Change theme"
      title="Change theme"
    >
      <Palette class="w-4 h-4" />
      <span class="theme-trigger-label">Theme</span>
      <div 
        class="theme-trigger-swatch"
        :style="{ background: currentTheme.colors.primary }"
      />
    </button>

    <Transition :name="openAbove ? 'dropdown-up' : 'dropdown-down'">
      <div 
        v-if="isOpen" 
        class="theme-dropdown"
        :class="{ 'theme-dropdown--above': openAbove }"
      >
        <div class="theme-dropdown-header">
          <span class="theme-dropdown-title">Select Theme</span>
        </div>
        <div class="theme-dropdown-list">
          <button
            v-for="theme in themes"
            :key="theme.id"
            class="theme-option"
            :class="{ 'theme-option--active': theme.id === currentTheme.id }"
            @click="selectTheme(theme.id)"
          >
            <div class="theme-option-swatches">
              <div 
                class="theme-swatch theme-swatch--bg"
                :style="{ background: theme.colors.background }"
              />
              <div 
                class="theme-swatch theme-swatch--primary"
                :style="{ background: theme.colors.primary }"
              />
              <div 
                class="theme-swatch theme-swatch--secondary"
                :style="{ background: theme.colors.secondary }"
              />
            </div>
            <div class="theme-option-info">
              <span class="theme-option-name">{{ theme.name }}</span>
              <span class="theme-option-desc">{{ theme.description }}</span>
            </div>
            <Check 
              v-if="theme.id === currentTheme.id"
              class="theme-option-check"
            />
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.theme-switcher {
  position: relative;
}

.theme-trigger {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  background: transparent;
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.theme-trigger:hover,
.theme-trigger--active {
  background: var(--bg-elevated);
  border-color: var(--border-accent);
  color: var(--text-primary);
}

.theme-trigger-label {
  display: none;
}

@media (min-width: 640px) {
  .theme-trigger-label {
    display: inline;
  }
}

.theme-trigger-swatch {
  width: 14px;
  height: 14px;
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.theme-dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 240px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.4);
  z-index: 100;
}

.theme-dropdown--above {
  top: auto;
  bottom: calc(100% + 8px);
}

.theme-dropdown-header {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.theme-dropdown-title {
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-secondary);
}

.theme-dropdown-list {
  padding: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.theme-option {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.75rem;
  border-radius: 8px;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
}

.theme-option:hover {
  background: var(--bg-elevated);
}

.theme-option--active {
  background: rgba(var(--accent-primary-rgb), 0.1);
}

.theme-option--active:hover {
  background: rgba(var(--accent-primary-rgb), 0.15);
}

.theme-option-swatches {
  display: flex;
  gap: 2px;
  padding: 3px;
  background: var(--bg-deep);
  border-radius: 6px;
}

.theme-swatch {
  width: 16px;
  height: 24px;
  border-radius: 3px;
}

.theme-swatch--bg {
  border-radius: 3px 0 0 3px;
}

.theme-swatch--secondary {
  border-radius: 0 3px 3px 0;
}

.theme-option-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.theme-option-name {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-primary);
}

.theme-option-desc {
  font-size: 0.6875rem;
  color: var(--text-muted);
}

.theme-option-check {
  width: 16px;
  height: 16px;
  color: var(--accent-primary);
  flex-shrink: 0;
}

/* Dropdown transition - opening downward */
.dropdown-down-enter-active,
.dropdown-down-leave-active {
  transition: all 0.2s ease;
}

.dropdown-down-enter-from,
.dropdown-down-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Dropdown transition - opening upward */
.dropdown-up-enter-active,
.dropdown-up-leave-active {
  transition: all 0.2s ease;
}

.dropdown-up-enter-from,
.dropdown-up-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
