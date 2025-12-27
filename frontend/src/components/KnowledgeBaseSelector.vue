<template>
  <div v-if="visible" class="kb-overlay" @click="close">
    <div class="kb-dropdown" @click.stop @wheel.stop :style="dropdownStyle">
      <!-- Search -->
      <div class="kb-search">
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          :placeholder="$t('knowledgeBase.searchPlaceholder')"
          class="kb-search-input"
          @keydown.down.prevent="moveSelection(1)"
          @keydown.up.prevent="moveSelection(-1)"
          @keydown.enter.prevent="toggleSelection"
          @keydown.esc="close"
        />
      </div>

      <!-- List -->
      <div class="kb-list" ref="kbList" @wheel.stop>
        <div
          v-for="(kb, index) in filteredKnowledgeBases"
          :key="kb.id"
          :class="['kb-item', { selected: isSelected(kb.id), highlighted: highlightedIndex === index }]"
          @click="toggleKb(kb.id)"
          @mouseenter="highlightedIndex = index"
        >
          <div class="kb-item-left">
            <div class="checkbox" :class="{ checked: isSelected(kb.id) }">
              <svg v-if="isSelected(kb.id)" width="12" height="12" viewBox="0 0 12 12" fill="none">
                <path d="M10 3L4.5 8.5L2 6" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </div>
            <div class="kb-icon" :class="{ 'faq': kb.type === 'faq' }">
              <svg v-if="kb.type === 'faq'" width="14" height="14" viewBox="0 0 24 24" fill="none">
                <path d="M12 22C17.5228 22 22 17.5228 22 12C22 6.47715 17.5228 2 12 2C6.47715 2 2 6.47715 2 12C2 17.5228 6.47715 22 12 22Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="M9 9C9 7.89543 9.89543 7 11 7H13C14.1046 7 15 7.89543 15 9C15 10.1046 14.1046 11 13 11H12V14" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                <circle cx="12" cy="17" r="1" fill="currentColor"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none">
                <path d="M22 19C22 19.5304 21.7893 20.0391 21.4142 20.4142C21.0391 20.7893 20.5304 21 20 21H4C3.46957 21 2.96086 20.7893 2.58579 20.4142C2.21071 20.0391 2 19.5304 2 19V5C2 4.46957 2.21071 3.96086 2.58579 3.58579C2.96086 3.21071 3.46957 3 4 3H9L11 6H20C20.5304 6 21.0391 6.21071 21.4142 6.58579C21.7893 6.96086 22 7.46957 22 8V19Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </div>
            <div class="kb-name-wrap">
              <span class="kb-name">{{ kb.name }}</span>
              <span class="kb-docs">({{ kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count || 0) }})</span>
            </div>
          </div>
        </div>

        <div v-if="filteredKnowledgeBases.length === 0" class="kb-empty">
          {{ searchQuery ? $t('knowledgeBase.noMatch') : $t('knowledgeBase.noKnowledge') }}
        </div>
      </div>

      <!-- Bottom actions -->
      <div class="kb-actions">
        <button @click="selectAll" class="kb-btn">{{ $t('common.selectAll') }}</button>
        <button @click="clearAll" class="kb-btn">{{ $t('common.clear') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { useI18n } from 'vue-i18n'

interface KnowledgeBase {
  id: string
  name: string
  type?: 'document' | 'faq'
  knowledge_count?: number
  chunk_count?: number
  embedding_model_id?: string
  summary_model_id?: string
}

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  anchorEl?: any | null // Supports DOM node, ref, component instance
  dropdownWidth?: number
  offsetY?: number
}>()

const emit = defineEmits(['close', 'update:visible'])

const settingsStore = useSettingsStore()

// Local state
const searchQuery = ref('')
const highlightedIndex = ref(0)
const knowledgeBases = ref<KnowledgeBase[]>([])
const searchInput = ref<HTMLInputElement | null>(null)
const kbList = ref<HTMLElement | null>(null)
const dropdownStyle = ref<Record<string, string>>({})

// props defaults
const dropdownWidth = props.dropdownWidth ?? 300
const offsetY = props.offsetY ?? 8

// Filter: only show initialized (with embedding & summary)
const filteredKnowledgeBases = computed(() => {
  const valid = knowledgeBases.value.filter(
    k => k.embedding_model_id && k.summary_model_id
  )
  if (!searchQuery.value) return valid
  const q = searchQuery.value.toLowerCase()
  return valid.filter(k => k.name.toLowerCase().includes(q))
})

const selectedKbIds = computed(() => settingsStore.settings.selectedKnowledgeBases || [])

// helper: resolve real DOM element from props.anchorEl (supports multiple formats)
const resolveAnchorEl = () => {
  const a = props.anchorEl
  if (!a) return null
  // If it's a Vue ref: take .value
  if (typeof a === 'object' && 'value' in a) {
    return a.value ?? null
  }
  // If it's a component instance (may have $el)
  if (typeof a === 'object' && '$el' in a) {
    // @ts-ignore
    return a.$el ?? null
  }
  // Direct DOM node or DOMRect
  return a
}

const isSelected = (id: string) => selectedKbIds.value.includes(id)

const toggleKb = (id: string) => {
  isSelected(id) ? settingsStore.removeKnowledgeBase(id) : settingsStore.addKnowledgeBase(id)
}

const toggleSelection = () => {
  const kb = filteredKnowledgeBases.value[highlightedIndex.value]
  if (kb) toggleKb(kb.id)
}

const moveSelection = (dir: number) => {
  const max = filteredKnowledgeBases.value.length
  if (max === 0) return
  highlightedIndex.value = Math.max(0, Math.min(max - 1, highlightedIndex.value + dir))
  nextTick(() => {
    const items = kbList.value?.querySelectorAll('.kb-item')
    items?.[highlightedIndex.value]?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

const selectAll = () => settingsStore.selectKnowledgeBases(filteredKnowledgeBases.value.map(k => k.id))
const clearAll = () => settingsStore.clearKnowledgeBases()

const close = () => {
  emit('update:visible', false)
  emit('close')
}

const loadKnowledgeBases = async () => {
  try {
    const res: any = await listKnowledgeBases()
    if (res?.data && Array.isArray(res.data)) knowledgeBases.value = res.data
  } catch (e) {
    console.error(t('knowledgeBase.loadingFailed'), e)
  }
}

// Calculate dropdown position: horizontally centered with button midpoint, handle viewport boundaries
const updateDropdownPosition = () => {
  const anchor = resolveAnchorEl()
  
  // fallback function
  const applyFallback = () => {
    const vw = window.innerWidth;
    const topFallback = Math.max(80, window.innerHeight / 2 - 160);
    dropdownStyle.value = {
      position: 'fixed',
      width: `${dropdownWidth}px`,
      left: `${Math.round((vw - dropdownWidth) / 2)}px`,
      top: `${Math.round(topFallback)}px`,
      transform: 'none',
      margin: '0',
      padding: '0',
    };
  };
  
  if (!anchor) {
    applyFallback()
    return
  }

  // Get anchor's bounding rect (relative to viewport)
  let rect: DOMRect | null = null
  try {
    if (typeof anchor.getBoundingClientRect === 'function') {
      rect = anchor.getBoundingClientRect()
      console.log('[KB Selector] Button rect:', {
        top: rect.top,
        bottom: rect.bottom,
        left: rect.left,
        right: rect.right,
        width: rect.width,
        height: rect.height
      })
    } else if (anchor.width !== undefined && anchor.left !== undefined) {
      // Already a DOMRect
      rect = anchor as DOMRect
    }
  } catch (e) {
    console.error('[KnowledgeBaseSelector] Error getting bounding rect:', e)
  }
  
  if (!rect || rect.width === 0 || rect.height === 0) {
    applyFallback()
    return
  }

  const vw = window.innerWidth
  const vh = window.innerHeight
  
  // Left-align to the left edge of the trigger element
  // Use Math.floor instead of Math.round to avoid pixel alignment issues
  let left = Math.floor(rect.left)
  
  // Boundary handling: don't exceed viewport (leave 16px margin)
  const minLeft = 16
  const maxLeft = Math.max(16, vw - dropdownWidth - 16)
  left = Math.max(minLeft, Math.min(maxLeft, left))

  // Vertical positioning: close to button, use reasonable height to avoid empty space
  const preferredDropdownHeight = 280 // Preferred height (compact and sufficient)
  const maxDropdownHeight = 360 // Max height
  const minDropdownHeight = 200 // Min height
  const topMargin = 20 // Top margin
  const spaceBelow = vh - rect.bottom // Space below
  const spaceAbove = rect.top // Space above
  
  console.log('[KB Selector] Space check:', {
    spaceBelow,
    spaceAbove,
    windowHeight: vh
  })
  
  let actualHeight: number
  let shouldOpenBelow: boolean
  
  // Prioritize space below
  if (spaceBelow >= minDropdownHeight + offsetY) {
    // Sufficient space below, pop down
    actualHeight = Math.min(preferredDropdownHeight, spaceBelow - offsetY - 16)
    shouldOpenBelow = true
    console.log('[KB Selector] Position: below button', { actualHeight })
  } else {
    // Pop up, prefer preferredHeight, extend to maxHeight only if necessary
    const availableHeight = spaceAbove - offsetY - topMargin
    if (availableHeight >= preferredDropdownHeight) {
      // Sufficient space for preferred height
      actualHeight = preferredDropdownHeight
    } else {
      // Not enough space, use available space (but not less than min height)
      actualHeight = Math.max(minDropdownHeight, availableHeight)
    }
    shouldOpenBelow = false
    console.log('[KB Selector] Position: above button', { actualHeight })
  }
  
  // Use different positioning based on pop direction
  if (shouldOpenBelow) {
    // Pop down: use top positioning
    const top = Math.floor(rect.bottom + offsetY)
    console.log('[KB Selector] Opening below, top:', top)
    dropdownStyle.value = {
      position: 'fixed',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      top: `${top}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none',
      margin: '0',
      padding: '0'
    }
  } else {
    // Pop up: use bottom positioning
    const bottom = vh - rect.top + offsetY
    console.log('[KB Selector] Opening above, bottom:', bottom)
    dropdownStyle.value = {
      position: 'fixed',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      bottom: `${bottom}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none',
      margin: '0',
      padding: '0'
    }
  }
}

// Event listener references for cleanup
let resizeHandler: (() => void) | null = null
let scrollHandler: (() => void) | null = null

// Handle visibility changes
watch(() => props.visible, async (v) => {
  if (v) {
    await loadKnowledgeBases();
    // Wait for DOM to render before calculating position
    await nextTick();
    // Update position multiple times to ensure accuracy
    requestAnimationFrame(() => {
      updateDropdownPosition();
      requestAnimationFrame(() => {
        updateDropdownPosition();
        setTimeout(() => {
          updateDropdownPosition();
        }, 50);
      });
    });
    // Ensure focus
    nextTick(() => searchInput.value?.focus());
    // Listen for resize/scroll for fine-tuning (use passive for performance)
    resizeHandler = () => updateDropdownPosition();
    scrollHandler = () => updateDropdownPosition();
    window.addEventListener('resize', resizeHandler, { passive: true });
    window.addEventListener('scroll', scrollHandler, { passive: true, capture: true });
  } else {
    searchQuery.value = '';
    highlightedIndex.value = 0;
    // Cleanup event listeners
    if (resizeHandler) {
      window.removeEventListener('resize', resizeHandler);
      resizeHandler = null;
    }
    if (scrollHandler) {
      window.removeEventListener('scroll', scrollHandler, { capture: true });
      scrollHandler = null;
    }
  }
});
</script>

<style scoped lang="less">
// Ensure all elements use border-box box model
.kb-overlay,
.kb-overlay *,
.kb-overlay *::before,
.kb-overlay *::after {
  box-sizing: border-box;
}

.kb-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: transparent;
  /* Don't block click-through, but prevent touch scrolling */
  touch-action: none;
}

/* Dropdown panel uses fixed positioning relative to viewport */
.kb-dropdown {
  position: fixed !important;
  background: #fff;
  border: 1px solid #e7e9eb;
  border-radius: 10px;
  box-shadow: 0 6px 28px rgba(15, 23, 42, 0.08);
  overflow: hidden;
  animation: fadeIn 0.15s ease-out;
  z-index: 10000;
  margin: 0;
  /* Ensure accurate positioning, use scale animation instead of translate */
  transform-origin: top left;
  display: flex;
  flex-direction: column;
}

/* Width is controlled by JS (dropdownWidth), internal styles only */
.kb-search {
  padding: 8px 10px;
  border-bottom: 1px solid #f1f3f4;
}
.kb-search-input {
  width: 100%;
  padding: 6px 10px;
  font-size: 12px;
  border: 1px solid #eef1f2;
  border-radius: 6px;
  background: #fbfdfc;
  outline: none;
  transition: border 0.12s;
}
.kb-search-input:focus {
  border-color: #10b981;
  background: #fff;
}

.kb-list {
  flex: 1;
  min-height: 0; /* Allow flex child elements to shrink */
  max-height: 260px;
  overflow-y: auto;
  padding: 6px 8px;
  /* Ensure scrolling is restricted within this container */
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}

.kb-item {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.12s;
  margin-bottom: 4px;
}
.kb-item:last-child { margin-bottom: 0; }

.kb-item:hover,
.kb-item.highlighted { background: #f6f8f7; }

.kb-item.selected { background: #eefdf5; }

.kb-item-left {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.checkbox {
  width: 16px; height: 16px;
  border-radius: 3px;
  border: 1.5px solid #d7dadd;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.checkbox.checked {
  background: #10b981;
  border-color: #10b981;
}
.checkbox.checked svg {
  width: 10px;
  height: 10px;
}
.kb-icon {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #059669;
  
  &.faq {
    color: #0052d9;
  }
}
.kb-name-wrap { display:flex; flex-direction: row; align-items: center; gap: 4px; min-width: 0; }
.kb-name { font-size: 12px; color: #222; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; line-height: 1.4; }
.kb-docs { font-size: 11px; color: #8b9196; flex-shrink: 0; }

.kb-empty { padding: 20px 8px; text-align: center; color: #9aa0a6; font-size: 12px; }

.kb-actions {
  display: flex;
  gap: 8px;
  padding: 8px 10px;
  border-top: 1px solid #f2f4f5;
  background: #fafcfc;
}
.kb-btn {
  flex: 1;
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid #e1e5e6;
  background: #fff;
  font-size: 12px;
  color: #52575a;
  cursor: pointer;
  transition: all 0.12s;
}
.kb-btn:hover {
  border-color: #10b981;
  color: #10b981;
  background: #f0fdf6;
}

@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}
</style>
