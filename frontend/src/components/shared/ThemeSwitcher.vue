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
  <div ref="dropdownRef" class="relative">
    <button 
      ref="triggerRef"
      class="flex items-center gap-2 py-2 px-3 rounded-xs bg-transparent border border-border-subtle text-text-secondary text-[0.8125rem] cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:border-border-accent hover:text-text-primary"
      :class="{ 'bg-bg-elevated border-border-accent text-text-primary': isOpen }"
      @click="toggleDropdown"
      aria-label="Change theme"
      title="Change theme"
    >
      <Palette class="w-4 h-4" />
      <span class="hidden sm:inline">Theme</span>
      <div 
        class="w-3.5 h-3.5 rounded border border-white/15"
        :style="{ background: currentTheme?.colors.primary }"
      />
    </button>

    <Transition :name="openAbove ? 'dropdown-up' : 'dropdown-down'">
      <div 
        v-if="isOpen" 
        class="absolute right-0 w-60 bg-bg-surface border border-border-subtle rounded-xs overflow-hidden shadow-[0_10px_40px_rgba(0,0,0,0.4)] z-[100]"
        :class="openAbove ? 'bottom-[calc(100%+8px)]' : 'top-[calc(100%+8px)]'"
      >
        <div class="py-3.5 px-4 border-b border-border-subtle">
          <span class="text-xs font-medium uppercase tracking-wider text-text-secondary">Select Theme</span>
        </div>
        <div class="p-2 flex flex-col gap-1">
          <button
            v-for="theme in themes"
            :key="theme.id"
            class="flex items-center gap-3 w-full p-3 rounded-xs bg-transparent border-none cursor-pointer transition-all duration-150 text-left hover:bg-bg-elevated"
            :class="{ 'bg-[rgba(var(--accent-primary-rgb),0.1)] hover:bg-[rgba(var(--accent-primary-rgb),0.15)]': theme.id === currentTheme?.id }"
            @click="selectTheme(theme.id)"
          >
            <div class="flex gap-0.5 p-[3px] bg-bg-deep rounded-xs">
              <div 
                class="w-4 h-6 rounded-l"
                :style="{ background: theme.colors.background }"
              />
              <div 
                class="w-4 h-6"
                :style="{ background: theme.colors.primary }"
              />
              <div 
                class="w-4 h-6 rounded-r"
                :style="{ background: theme.colors.secondary }"
              />
            </div>
            <div class="flex-1 min-w-0 flex flex-col gap-0.5">
              <span class="text-[0.8125rem] font-medium text-text-primary">{{ theme.name }}</span>
              <span class="text-[0.6875rem] text-text-muted">{{ theme.description }}</span>
            </div>
            <Check 
              v-if="theme.id === currentTheme?.id"
              class="w-4 h-4 text-accent-primary shrink-0"
            />
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
  @reference "../../style.css";
/* Dropdown transition - opening downward */
.dropdown-down-enter-active,
.dropdown-down-leave-active {
  @apply transition-all duration-200 ease-out;
}

.dropdown-down-enter-from,
.dropdown-down-leave-to {
  @apply opacity-0 -translate-y-2;
}

/* Dropdown transition - opening upward */
.dropdown-up-enter-active,
.dropdown-up-leave-active {
  @apply transition-all duration-200 ease-out;
}

.dropdown-up-enter-from,
.dropdown-up-leave-to {
  @apply opacity-0 translate-y-2;
}
</style>
