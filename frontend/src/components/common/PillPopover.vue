<template>
  <div ref="rootRef" class="relative inline-block text-left">
    <!-- Capsule Trigger -->
    <button
      ref="triggerRef"
      type="button"
      :disabled="disabled"
      class="pill-trigger group inline-flex items-center justify-between h-[28px] rounded-full border text-xs font-semibold transition-all duration-150 select-none outline-none cursor-pointer"
      :class="[
        disabled
          ? 'cursor-not-allowed opacity-50 bg-gray-100 text-gray-400 border-gray-200 dark:bg-dark-800 dark:border-dark-700'
          : isOpen
            ? 'bg-gray-100/90 text-gray-900 border-gray-400 dark:bg-dark-700 dark:text-white dark:border-dark-500 ring-2 ring-gray-900/10 dark:ring-white/10'
            : 'bg-white text-gray-800 border-gray-300 hover:bg-gray-50 hover:text-gray-900 hover:border-gray-400 dark:bg-dark-800 dark:text-gray-200 dark:border-dark-600 dark:hover:bg-dark-700 dark:hover:border-dark-500 shadow-xs',
        triggerClass
      ]"
      :style="{ padding: '0 26px 0 12px' }"
      :aria-haspopup="'listbox'"
      :aria-expanded="isOpen"
      @click="toggle"
      @keydown.esc="close"
    >
      <span class="truncate pr-1.5 font-medium">
        {{ selectedLabel || placeholder }}
      </span>
      <!-- 9px wide, 6px high triangle indicator -->
      <svg
        class="w-[9px] h-[6px] text-gray-500 group-hover:text-gray-700 dark:text-gray-400 dark:group-hover:text-gray-200 transition-transform duration-200 shrink-0"
        :class="{ 'rotate-180': isOpen }"
        viewBox="0 0 9 6"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d="M4.5 6L0 0h9L4.5 6z" />
      </svg>
    </button>

    <!-- Floating Popover Panel -->
    <Transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="transform scale-95 opacity-0 -translate-y-1"
      enter-to-class="transform scale-100 opacity-100 translate-y-0"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="transform scale-100 opacity-100 translate-y-0"
      leave-to-class="transform scale-95 opacity-0 -translate-y-1"
    >
      <div
        v-if="isOpen"
        class="pill-panel absolute mt-1.5 z-30 p-1.5 rounded-[14px] border border-gray-200/90 bg-white shadow-xl shadow-gray-900/10 dark:border-dark-700 dark:bg-dark-800 dark:shadow-black/50 overflow-hidden"
        :class="[
          align === 'end' ? 'right-0' : 'left-0',
          panelClass
        ]"
        :style="{
          minWidth: minWidth,
          maxWidth: 'min(420px, 82vw)',
          maxHeight: '340px'
        }"
        role="listbox"
      >
        <!-- Optional Group Title -->
        <div
          v-if="groupTitle"
          class="px-2.5 py-1 text-[9.5px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500 select-none"
        >
          {{ groupTitle }}
        </div>

        <!-- Options Container with aesthetic spacing -->
        <div class="space-y-0.5 overflow-y-auto max-h-[300px] px-0.5 py-0.5">
          <div
            v-for="opt in options"
            :key="String(opt.value)"
            class="pill-row group flex items-center justify-between min-h-[28px] px-3 py-1 rounded-full text-xs font-normal transition-colors duration-150 cursor-pointer select-none"
            :class="[
              opt.disabled
                ? 'opacity-45 cursor-not-allowed text-gray-400'
                : isSelected(opt.value)
                  ? 'bg-gray-100 dark:bg-dark-700 font-semibold text-gray-900 dark:text-white'
                  : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100/70 dark:hover:bg-dark-700/60'
            ]"
            role="option"
            :aria-selected="isSelected(opt.value)"
            @click="select(opt)"
          >
            <div class="flex items-center gap-2 truncate pr-2">
              <span class="truncate">{{ opt.label }}</span>
              <span v-if="opt.hint" class="text-[10px] text-gray-400 dark:text-gray-500 font-normal">
                {{ opt.hint }}
              </span>
            </div>

            <!-- Checkmark for selected row -->
            <svg
              v-if="isSelected(opt.value)"
              class="w-3.5 h-3.5 text-gray-900 dark:text-white shrink-0 ml-auto"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fill-rule="evenodd"
                d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z"
                clip-rule="evenodd"
              />
            </svg>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

export interface PillOption<T = string | number> {
  value: T
  label: string
  hint?: string
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    options: PillOption[]
    placeholder?: string
    groupTitle?: string
    align?: 'start' | 'end'
    disabled?: boolean
    minWidth?: string
    triggerClass?: string
    panelClass?: string
  }>(),
  {
    modelValue: '',
    placeholder: '请选择',
    groupTitle: '',
    align: 'start',
    disabled: false,
    minWidth: '220px',
    triggerClass: '',
    panelClass: ''
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', val: string | number): void
  (e: 'change', val: string | number): void
}>()

const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const isOpen = ref(false)

const selectedOption = computed(() => {
  return props.options.find(opt => opt.value === props.modelValue)
})

const selectedLabel = computed(() => {
  return selectedOption.value?.label ?? ''
})

function isSelected(val: string | number) {
  return props.modelValue === val
}

function toggle() {
  if (props.disabled) return
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}

function select(opt: PillOption) {
  if (opt.disabled) return
  close()
  if (opt.value !== props.modelValue) {
    emit('update:modelValue', opt.value)
    emit('change', opt.value)
  }
}

function handlePointerDown(e: MouseEvent) {
  if (!isOpen.value) return
  if (rootRef.value && !rootRef.value.contains(e.target as Node)) {
    close()
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && isOpen.value) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
})
</script>
