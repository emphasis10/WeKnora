<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch, nextTick, h } from "vue";
import { useRoute, useRouter } from 'vue-router';
import { onBeforeRouteUpdate } from 'vue-router';
import { MessagePlugin } from "tdesign-vue-next";
import { useSettingsStore } from '@/stores/settings';
import { useUIStore } from '@/stores/ui';
import { listKnowledgeBases, searchKnowledge, batchQueryKnowledge } from '@/api/knowledge-base';
import { stopSession } from '@/api/chat';
import KnowledgeBaseSelector from './KnowledgeBaseSelector.vue';
import MentionSelector from './MentionSelector.vue';
import { getCaretCoordinates } from '@/utils/caret';
import { listModels, type ModelConfig } from '@/api/model';
import { getTenantWebSearchConfig } from '@/api/web-search';
import { getConversationConfig, updateConversationConfig, type ConversationConfig } from '@/api/system';
import { useI18n } from 'vue-i18n';

const route = useRoute();
const router = useRouter();
const settingsStore = useSettingsStore();
const uiStore = useUIStore();
let query = ref("");
const showKbSelector = ref(false);
const atButtonRef = ref<HTMLElement>();
const showAgentModeSelector = ref(false);
const agentModeButtonRef = ref<HTMLElement>();
const agentModeDropdownStyle = ref<Record<string, string>>({});

// Mention related state
const showMention = ref(false);
const mentionQuery = ref("");
const mentionItems = ref<Array<{ id: string; name: string; type: 'kb' | 'file'; kbType?: 'document' | 'faq'; count?: number; kbName?: string }>>([]);
const mentionActiveIndex = ref(0);
const mentionStyle = ref<Record<string, string>>({});
const textareaRef = ref<any>(null); // Ref to t-textarea component
const mentionStartPos = ref(0);
const isComposing = ref(false);
const isMentionTriggeredByButton = ref(false);
const mentionHasMore = ref(false);
const mentionLoading = ref(false);
const mentionOffset = ref(0);
const MENTION_PAGE_SIZE = 20;

const props = defineProps({
  isReplying: {
    type: Boolean,
    required: false
  },
  sessionId: {
    type: String,
    required: false
  },
  assistantMessageId: {
    type: String,
    required: false
  }
});

const isAgentEnabled = computed(() => settingsStore.isAgentEnabled);
const isWebSearchEnabled = computed(() => settingsStore.isWebSearchEnabled);
const selectedKbIds = computed(() => settingsStore.settings.selectedKnowledgeBases || []);
const selectedFileIds = computed(() => settingsStore.settings.selectedFiles || []);
const isWebSearchConfigured = ref(false);

// Get info of selected knowledge bases
const knowledgeBases = ref<Array<{ id: string; name: string; type?: 'document' | 'faq'; knowledge_count?: number; chunk_count?: number }>>([]);
const fileList = ref<Array<{ id: string; name: string }>>([]);

const selectedKbs = computed(() => {
  return knowledgeBases.value.filter(kb => selectedKbIds.value.includes(kb.id));
});

const selectedFiles = computed(() => {
  // If we have file details in fileList, use them.
  // Otherwise we might show ID or Loading...
  return selectedFileIds.value.map((id: string) => {
    const found = fileList.value.find(f => f.id === id);
    return found || { id, name: 'Loading...' };
  });
});

  // Merge all selected items (for display in input field)
  const allSelectedItems = computed(() => {
    const kbs = selectedKbs.value.map(kb => ({ 
      ...kb, 
      type: 'kb' as const,
      kbType: kb.type // Keep original type for display distinction
    }));
    const files = selectedFiles.value.map((f: { id: string; name: string }) => ({ 
      ...f, 
      type: 'file' as const 
    }));
    return [...kbs, ...files];
  });

// Remove selected item
const removeSelectedItem = (item: { id: string; type: 'kb' | 'file' }) => {
  if (item.type === 'kb') {
    settingsStore.removeKnowledgeBase(item.id);
  } else {
    settingsStore.removeFile(item.id);
  }
};

// Model related state
const availableModels = ref<ModelConfig[]>([]);
// Use computed to read from store and sync back to store via setter
const selectedModelId = computed({
  get: () => settingsStore.conversationModels.selectedChatModelId || '',
  set: (val: string) => settingsStore.updateConversationModels({ selectedChatModelId: val })
});
const conversationConfig = ref<ConversationConfig | null>(null);
const modelsLoading = ref(false);
const showModelSelector = ref(false);
const modelButtonRef = ref<HTMLElement>();
const modelDropdownStyle = ref<Record<string, string>>({});

const { t } = useI18n();

// Displayed knowledge base tags (up to 2)
const displayedKbs = computed(() => selectedKbs.value.slice(0, 2));
const remainingCount = computed(() => Math.max(0, selectedKbs.value.length - 2));

// Calculate input field placeholder based on different states
const inputPlaceholder = computed(() => {
  const hasKnowledge = allSelectedItems.value.length > 0;
  const hasWebSearch = isWebSearchEnabled.value && isWebSearchConfigured.value;
  
  if (hasKnowledge && hasWebSearch) {
    // Has knowledge base + Has web search
    return t('input.placeholderKbAndWeb');
  } else if (hasKnowledge) {
    // Has knowledge base + No web search
    return t('input.placeholderWithContext');
  } else if (hasWebSearch) {
    // No knowledge base + Has web search
    return t('input.placeholderWebOnly');
  } else {
    // No knowledge base + No web search (pure model conversation)
    return t('input.placeholder');
  }
});

// Load knowledge base list
const loadKnowledgeBases = async () => {
  try {
    const response: any = await listKnowledgeBases();
    if (response.data && Array.isArray(response.data)) {
      const validKbs = response.data.filter((kb: any) => 
        kb.embedding_model_id && kb.embedding_model_id !== '' &&
        kb.summary_model_id && kb.summary_model_id !== ''
      );
      knowledgeBases.value = validKbs;
      
      // Clear invalid knowledge base IDs (deleted or not in active list)
      const validKbIds = new Set(validKbs.map((kb: any) => kb.id));
      const currentSelectedIds = settingsStore.settings.selectedKnowledgeBases || [];
      const validSelectedIds = currentSelectedIds.filter((id: string) => validKbIds.has(id));
      
      // If there are invalid IDs, update store
      if (validSelectedIds.length !== currentSelectedIds.length) {
        settingsStore.selectKnowledgeBases(validSelectedIds);
      }
    }
  } catch (error) {
    console.error('Failed to load knowledge bases:', error);
  }
};

const loadFiles = async () => {
  const ids = selectedFileIds.value;
  if (ids.length === 0) return;
  
  // Filter out files we already have info for
  const missingIds = ids.filter((id: string) => !fileList.value.find(f => f.id === id));
  if (missingIds.length === 0) return;

  try {
    const query = new URLSearchParams();
    missingIds.forEach((id: string) => query.append('ids', id));
    const res: any = await batchQueryKnowledge(query.toString());
    if (res.data) {
      const newFiles = res.data.map((f: any) => ({ id: f.id, name: f.title || f.file_name }));
      fileList.value = [...fileList.value, ...newFiles];
    }
  } catch (e) {
    console.error("Failed to load files", e);
  }
};

watch(selectedFileIds, () => {
  loadFiles();
}, { immediate: true });

const loadWebSearchConfig = async () => {
  try {
    const response: any = await getTenantWebSearchConfig();
    const config = response?.data;
    const configured = !!(config && config.provider);
    isWebSearchConfigured.value = configured;

    if (!configured && settingsStore.isWebSearchEnabled) {
      settingsStore.toggleWebSearch(false);
    }
  } catch (error) {
    console.error('Failed to load web search config:', error);
    isWebSearchConfigured.value = false;
    if (settingsStore.isWebSearchEnabled) {
      settingsStore.toggleWebSearch(false);
    }
  }
};

const loadConversationConfig = async () => {
  try {
    const response = await getConversationConfig();
    conversationConfig.value = response.data;
    settingsStore.updateConversationModels({
      summaryModelId: response.data?.summary_model_id || '',
      rerankModelId: response.data?.rerank_model_id || '',
    });
    if (!selectedModelId.value) {
      selectedModelId.value = response.data?.summary_model_id || '';
    }
    ensureModelSelection();
  } catch (error) {
    console.error('Failed to load conversation config:', error);
  }
};

const loadChatModels = async () => {
  if (modelsLoading.value) return;
  modelsLoading.value = true;
  try {
    const models = await listModels('KnowledgeQA');
    availableModels.value = Array.isArray(models) ? models : [];
    ensureModelSelection();
  } catch (error) {
    console.error('Failed to load chat models:', error);
    availableModels.value = [];
  } finally {
    modelsLoading.value = false;
  }
};

const ensureModelSelection = () => {
  if (selectedModelId.value) {
    return;
  }
  if (conversationConfig.value?.summary_model_id) {
    selectedModelId.value = conversationConfig.value.summary_model_id;
    return;
  }
  if (availableModels.value.length > 0) {
    selectedModelId.value = availableModels.value[0].id || '';
  }
};

const handleGoToConversationModels = () => {
  showModelSelector.value = false;
  router.push('/platform/settings');
  setTimeout(() => {
    const event = new CustomEvent('settings-nav', {
      detail: { section: 'models', subsection: 'chat' },
    });
    window.dispatchEvent(event);
  }, 100);
};

const handleModelChange = async (value: string | number | Array<string | number> | undefined) => {
  const normalized = Array.isArray(value) ? value[0] : value;
  const val = normalized !== undefined && normalized !== null ? String(normalized) : '';

  if (!val) {
    selectedModelId.value = '';
    return;
  }
  if (val === '__add_model__') {
    selectedModelId.value = conversationConfig.value?.summary_model_id || '';
    handleGoToConversationModels();
    return;
  }
  
  // Save to backend
  try {
    if (conversationConfig.value) {
      const updatedConfig = {
        ...conversationConfig.value,
        summary_model_id: val
      };
      const response = await updateConversationConfig(updatedConfig);
      
      // Update local state
      conversationConfig.value = response.data;
      selectedModelId.value = val;
      showModelSelector.value = false;
      
      // Sync to store
      settingsStore.updateConversationModels({
        summaryModelId: val,
        rerankModelId: conversationConfig.value?.rerank_model_id || '',
      });
      
      MessagePlugin.success(t('conversationSettings.toasts.chatModelSaved'));
    }
  } catch (error) {
    console.error('Failed to save model config:', error);
    MessagePlugin.error(t('conversationSettings.toasts.saveFailed'));
    // Restore to previous value
    selectedModelId.value = conversationConfig.value?.summary_model_id || '';
  }
};

const selectedModel = computed(() => {
  return availableModels.value.find(model => model.id === selectedModelId.value);
});

const updateModelDropdownPosition = () => {
  const anchor = modelButtonRef.value;
  if (!anchor) {
    modelDropdownStyle.value = {
      position: 'fixed',
      top: '50%',
      left: '50%',
      transform: 'translate(-50%, -50%)',
    };
    return;
  }
  
  // Get button position relative to viewport
  const rect = anchor.getBoundingClientRect();
  console.log('[Model Dropdown] Button rect:', {
    top: rect.top,
    bottom: rect.bottom,
    left: rect.left,
    right: rect.right,
    width: rect.width,
    height: rect.height
  });
  
  const dropdownWidth = 280;
  const offsetY = 8;
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  
  // Left align to the left edge of the trigger element
  // Use Math.floor instead of Math.round to avoid pixel alignment issues
  let left = Math.floor(rect.left);
  
  // Boundary handling: do not exceed viewport left/right (keep 16px margin)
  const minLeft = 16;
  const maxLeft = Math.max(16, vw - dropdownWidth - 16);
  left = Math.max(minLeft, Math.min(maxLeft, left));

  // Vertical positioning: close to button, use reasonable height to avoid empty space
  const preferredDropdownHeight = 280; // Preferred height (compact and sufficient)
  const maxDropdownHeight = 360; // Maximum height
  const minDropdownHeight = 200; // Minimum height
  const topMargin = 20; // Top margin
  const spaceBelow = vh - rect.bottom; // Remaining space below
  const spaceAbove = rect.top; // Remaining space above
  
  console.log('[Model Dropdown] Space check:', {
    spaceBelow,
    spaceAbove,
    windowHeight: vh
  });
  
  let actualHeight: number;
  let shouldOpenBelow: boolean;
  
  // Prioritize space below
  if (spaceBelow >= minDropdownHeight + offsetY) {
    // Enough space below, pop down
    actualHeight = Math.min(preferredDropdownHeight, spaceBelow - offsetY - 16);
    shouldOpenBelow = true;
    console.log('[Model Dropdown] Position: below button', { actualHeight });
  } else {
    // Pop up, prioritize preferredHeight, extend to maxHeight if necessary
    const availableHeight = spaceAbove - offsetY - topMargin;
    if (availableHeight >= preferredDropdownHeight) {
      // Enough space to display preferred height
      actualHeight = preferredDropdownHeight;
    } else {
      // Not enough space, use available space (but not less than minimum height)
      actualHeight = Math.max(minDropdownHeight, availableHeight);
    }
    shouldOpenBelow = false;
    console.log('[Model Dropdown] Position: above button', { actualHeight });
  }
  
  // Use different positioning methods based on pop direction
  if (shouldOpenBelow) {
    // Pop down: use top positioning, left aligned
    const top = Math.floor(rect.bottom + offsetY);
    console.log('[Model Dropdown] Opening below, top:', top);
    modelDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      top: `${top}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important',
      padding: '0 !important'
    };
  } else {
    // Pop up: use bottom positioning, left aligned
    const bottom = vh - rect.top + offsetY;
    console.log('[Model Dropdown] Opening above, bottom:', bottom);
    modelDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      bottom: `${bottom}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important',
      padding: '0 !important'
    };
  }
  
  console.log('[Model Dropdown] Applied style:', modelDropdownStyle.value);
};

// Mention Logic
let lastMentionQuery = '';
const loadMentionItems = async (q: string, resetIndex = true, append = false) => {
  console.log('[Mention] loadMentionItems called with query:', q, 'append:', append);
  
  if (!append) {
    mentionOffset.value = 0;
  }
  
  // Fetch KBs (always show, filter by query) - only on first load
  let kbItems: any[] = [];
  if (!append) {
    const kbs = knowledgeBases.value.filter(kb => 
      !q || kb.name.toLowerCase().includes(q.toLowerCase())
    );
    kbItems = kbs.map(kb => ({ 
      id: kb.id, 
      name: kb.name, 
      type: 'kb' as const, 
      kbType: kb.type || 'document',
      count: kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count || 0)
    }));
  }
  
  // Fetch Files from API
  let fileItems: any[] = [];
  mentionLoading.value = true;
  try {
    const res: any = await searchKnowledge(q || '', mentionOffset.value, MENTION_PAGE_SIZE);
    console.log('[Mention] searchKnowledge response:', res);
    if (res.data && Array.isArray(res.data)) {
      fileItems = res.data.map((f: any) => ({ 
        id: f.id, 
        name: f.title || f.file_name, 
        type: 'file' as const,
        kbName: f.knowledge_base_name || ''
      }));
    }
    mentionHasMore.value = res.has_more || false;
    mentionOffset.value += fileItems.length;
  } catch (e) {
    console.error('[Mention] searchKnowledge error:', e);
    mentionHasMore.value = false;
  } finally {
    mentionLoading.value = false;
  }
  
  if (append) {
    // Append file items to existing list
    mentionItems.value = [...mentionItems.value, ...fileItems];
  } else {
    mentionItems.value = [...kbItems, ...fileItems];
  }
  console.log('[Mention] Total items:', mentionItems.value.length, { kbItems: kbItems.length, fileItems: fileItems.length });
  
  // Only reset index if query changed or explicitly requested
  if (resetIndex || q !== lastMentionQuery) {
    mentionActiveIndex.value = 0;
  }
  // Ensure index is within bounds
  if (mentionActiveIndex.value >= mentionItems.value.length) {
    mentionActiveIndex.value = Math.max(0, mentionItems.value.length - 1);
  }
  lastMentionQuery = q;
};

const loadMoreMentionItems = () => {
  if (mentionHasMore.value && !mentionLoading.value) {
    loadMentionItems(lastMentionQuery, false, true);
  }
};

const getTextareaEl = () => {
  if (!textareaRef.value) return null;
  // If it's a native element
  if (textareaRef.value instanceof HTMLTextAreaElement) return textareaRef.value;
  // If it's a component wrapper
  const el = textareaRef.value.$el || textareaRef.value;
  if (!el) return null;
  if (el.tagName === 'TEXTAREA') return el as HTMLTextAreaElement;
  return el.querySelector('textarea');
};

const onInput = (val: string | InputEvent) => {
  // If in composition mode, do not process search logic, wait for compositionend
  if (isComposing.value) return;

  // TDesign t-textarea passes the value directly, not an event
  const inputVal = typeof val === 'string' ? val : query.value;
  
  const textarea = getTextareaEl();
  if (!textarea) {
    console.warn('[Mention] Could not get textarea element');
    return;
  }
  
  const cursor = textarea.selectionStart;
  const textBeforeCursor = inputVal.slice(0, cursor);
  
  console.log('[Mention] onInput called', { inputVal, cursor, textBeforeCursor, showMention: showMention.value });
  
  if (showMention.value) {
    // If not triggered by button, check for @ symbol
    if (!isMentionTriggeredByButton.value) {
      if (!inputVal || inputVal.length <= mentionStartPos.value || inputVal.charAt(mentionStartPos.value) !== '@') {
        showMention.value = false;
        return;
      }
    }

    // If triggered by button, mentionStartPos points to cursor position (before virtual @), so it should not be deleted to the left
    // But if user deletes previous content resulting in shorter length, handle it
    if (cursor < mentionStartPos.value) {
      showMention.value = false;
      return;
    }
    
    // Get query
    // If triggered by button, mentionStartPos is starting position, no need to +1 to skip @
    const start = isMentionTriggeredByButton.value ? mentionStartPos.value : mentionStartPos.value + 1;
    const q = inputVal.slice(start, cursor);
    
    if (q.includes(' ')) {
      showMention.value = false;
      return;
    }
    // Only reload if query changed
    if (q !== mentionQuery.value) {
      mentionQuery.value = q;
      loadMentionItems(q, true); // Reset index when query changes
    }
  } else {
    if (textBeforeCursor.endsWith('@')) {
      console.log('[Mention] @ detected, opening menu');
      isMentionTriggeredByButton.value = false;
      mentionStartPos.value = cursor - 1;
      showMention.value = true;
      mentionQuery.value = "";
      
      const coords = getCaretCoordinates(textarea, cursor);
      const rect = textarea.getBoundingClientRect();
      const scrollTop = textarea.scrollTop;
      const menuHeight = 320; // Estimated maximum height
      
      let left = rect.left + coords.left;
      // Prevent menu from going off-screen horizontally
      if (left + 300 > window.innerWidth) {
        left = window.innerWidth - 300 - 10;
      }
      
      // Cursor actual top position relative to viewport
      const cursorAbsoluteTop = rect.top + coords.top - scrollTop;
      const lineHeight = coords.height; // Cursor height

      // Check vertical space below cursor
      const spaceBelow = window.innerHeight - (cursorAbsoluteTop + lineHeight);
      
      if (spaceBelow < menuHeight && cursorAbsoluteTop > menuHeight) {
         // Show above cursor (using bottom positioning)
         // bottom distance = viewport height - cursor top position
         const bottom = window.innerHeight - cursorAbsoluteTop;
         mentionStyle.value = {
           left: `${left}px`,
           bottom: `${bottom}px`,
           top: 'auto'
         };
      } else {
         // Show below cursor (using top positioning)
         const top = cursorAbsoluteTop + lineHeight;
         mentionStyle.value = {
           left: `${left}px`,
           top: `${top}px`,
           bottom: 'auto'
         };
      }
      
      loadMentionItems("");
    }
  }
};

const onCompositionStart = () => {
  isComposing.value = true;
};

const onCompositionEnd = (e: CompositionEvent) => {
  isComposing.value = false;
  // Manually trigger onInput logic
  // Note: in compositionend, v-model might not be updated yet, or updated but we need the latest value
  // TDesign textarea might need nextTick
  nextTick(() => {
    onInput(query.value);
  });
};

const triggerMention = () => {
  const textarea = getTextareaEl();
  if (!textarea) return;
  
  // Close other selectors
  showAgentModeSelector.value = false;
  showModelSelector.value = false;

  textarea.focus();
  
  // Show menu directly without inserting @
  showMention.value = true;
  isMentionTriggeredByButton.value = true;
  mentionQuery.value = "";
  mentionStartPos.value = textarea.selectionStart;
  
  const rect = textarea.getBoundingClientRect();
  const menuHeight = 320;
  
  // Determine space above input box
  const spaceAbove = rect.top;
  const spaceBelow = window.innerHeight - rect.bottom;
  
  // Prioritize showing above, unless space above is insufficient and space below is sufficient
  if (spaceAbove > menuHeight || spaceAbove > spaceBelow) {
    // Show above textarea
    mentionStyle.value = {
      left: `${rect.left}px`,
      bottom: `${window.innerHeight - rect.top + 8}px`, // 8px padding
      top: 'auto'
    };
  } else {
    // Show below textarea
    mentionStyle.value = {
      left: `${rect.left}px`,
      top: `${rect.bottom + 8}px`,
      bottom: 'auto'
    };
  }
  
  loadMentionItems("");
};

const onMentionSelect = (item: any) => {
  if (item.type === 'kb') {
      settingsStore.addKnowledgeBase(item.id);
  } else if (item.type === 'file') {
      settingsStore.addFile(item.id);
      // Add to local cache immediately
      if (!fileList.value.find(f => f.id === item.id)) {
        fileList.value.push(item);
      }
  }
  
  const textarea = getTextareaEl();
  if (textarea) {
    // If triggered by typing @, need to delete @ and subsequent query text
    if (!isMentionTriggeredByButton.value) {
      const cursor = textarea.selectionStart;
      const textBeforeAt = query.value.slice(0, mentionStartPos.value);
      const textAfterCursor = query.value.slice(cursor);
      query.value = textBeforeAt + textAfterCursor;
      
      nextTick(() => {
        textarea.selectionStart = textarea.selectionEnd = mentionStartPos.value;
        textarea.focus();
      });
    } else {
      // If triggered by button and user typed query words, need to delete query words
      const cursor = textarea.selectionStart;
      if (cursor > mentionStartPos.value) {
         const textBeforeStart = query.value.slice(0, mentionStartPos.value);
         const textAfterCursor = query.value.slice(cursor);
         query.value = textBeforeStart + textAfterCursor;
         
         nextTick(() => {
           textarea.selectionStart = textarea.selectionEnd = mentionStartPos.value;
           textarea.focus();
         });
      } else {
         // Focus directly
         textarea.focus();
      }
    }
  }
  
  showMention.value = false;
};

const removeFile = (id: string) => {
  settingsStore.removeFile(id);
};

const toggleModelSelector = () => {
  // Mutually exclusive: close others
  showMention.value = false;
  showAgentModeSelector.value = false;

  showModelSelector.value = !showModelSelector.value;
  if (showModelSelector.value) {
    if (!availableModels.value.length) {
      loadChatModels();
    }
    // Update position multiple times to ensure accuracy
    nextTick(() => {
      updateModelDropdownPosition();
      requestAnimationFrame(() => {
        updateModelDropdownPosition();
        setTimeout(() => {
          updateModelDropdownPosition();
        }, 50);
      });
    });
  }
};

const closeModelSelector = () => {
  showModelSelector.value = false;
};

// Close Agent mode selector (click outside)
const closeAgentModeSelector = () => {
  showAgentModeSelector.value = false;
};

const closeMentionSelector = (e: MouseEvent) => {
  const target = e.target as HTMLElement;
  // If clicked inside input box area, do not close Mention list (controlled by cursor logic)
  if (target.closest('.rich-input-container')) {
    return;
  }
  showMention.value = false;
};

// Window event handlers
let resizeHandler: (() => void) | null = null;
let scrollHandler: (() => void) | null = null;

onMounted(() => {
  loadKnowledgeBases();
  loadWebSearchConfig();
  loadConversationConfig();
  loadChatModels();
  
  // If entering from within a knowledge base, select it automatically
  const kbId = (route.params as any)?.kbId as string;
  if (kbId && !selectedKbIds.value.includes(kbId)) {
    settingsStore.addKnowledgeBase(kbId);
  }

  // Listen for clicks outside to close dropdown menu
  document.addEventListener('click', closeAgentModeSelector);
  document.addEventListener('click', closeModelSelector);
  document.addEventListener('click', closeMentionSelector);
  
  // Listen for window resize and scroll to recalculate position
  resizeHandler = () => {
    if (showModelSelector.value) {
      updateModelDropdownPosition();
    }
    if (showAgentModeSelector.value) {
      updateAgentModeDropdownPosition();
    }
  };
  scrollHandler = () => {
    if (showModelSelector.value) {
      updateModelDropdownPosition();
    }
    if (showAgentModeSelector.value) {
      updateAgentModeDropdownPosition();
    }
  };
  
  window.addEventListener('resize', resizeHandler, { passive: true });
  window.addEventListener('scroll', scrollHandler, { passive: true, capture: true });
});

onUnmounted(() => {
  document.removeEventListener('click', closeAgentModeSelector);
  document.removeEventListener('click', closeModelSelector);
  document.removeEventListener('click', closeMentionSelector);
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler);
  }
  if (scrollHandler) {
    window.removeEventListener('scroll', scrollHandler, { capture: true });
  }
});

// Listen for route changes
watch(() => route.params.kbId, (newKbId) => {
  if (newKbId && typeof newKbId === 'string' && !selectedKbIds.value.includes(newKbId)) {
    settingsStore.addKnowledgeBase(newKbId);
  }
});

watch(() => uiStore.showSettingsModal, (visible, prevVisible) => {
  if (prevVisible && !visible) {
    loadWebSearchConfig();
  }
});

watch([selectedKbIds, selectedFileIds], ([kbIds, fileIds]) => {
  if (!kbIds.length && !fileIds.length) {
    closeModelSelector();
  }
}, { deep: true });

const emit = defineEmits(['send-msg', 'stop-generation']);

const createSession = (val: string) => {
  if (!val.trim()) {
    MessagePlugin.info(t('input.messages.enterContent'));
    return;
  }
  if (props.isReplying) {
    return MessagePlugin.error(t('input.messages.replying'));
  }
  // Get @mentioned knowledge bases and files info
  const mentionedItems = allSelectedItems.value.map(item => ({
    id: item.id,
    name: item.name,
    type: item.type,
    kb_type: item.type === 'kb' ? (item.kbType || 'document') : undefined
  }));
  emit('send-msg', val, selectedModelId.value, mentionedItems);
  clearvalue();
}

const updateAgentModeDropdownPosition = () => {
  const anchor = agentModeButtonRef.value;
  
  if (!anchor) {
    agentModeDropdownStyle.value = {
      position: 'fixed',
      top: '50%',
      left: '50%',
      transform: 'translate(-50%, -50%)'
    };
    return;
  }

  const rect = anchor.getBoundingClientRect();
  const dropdownWidth = 200;
  const offsetY = 8;
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  
  // Horizontal position: left aligned
  let left = Math.floor(rect.left);
  const minLeft = 16;
  const maxLeft = Math.max(16, vw - dropdownWidth - 16);
  left = Math.max(minLeft, Math.min(maxLeft, left));
  
  // Vertical position: close to button, use reasonable height to avoid empty space
  const preferredDropdownHeight = 140; // Agent mode selector has less content, use smaller preferred height
  const maxDropdownHeight = 150;
  const minDropdownHeight = 100;
  const topMargin = 20;
  const spaceBelow = vh - rect.bottom;
  const spaceAbove = rect.top;
  
  console.log('[Agent Dropdown] Space check:', {
    spaceBelow,
    spaceAbove,
    windowHeight: vh
  });
  
  let actualHeight: number;
  
  // Prioritize space below
  if (spaceBelow >= minDropdownHeight + offsetY) {
    // Enough space below, pop down
    actualHeight = Math.min(preferredDropdownHeight, spaceBelow - offsetY - 16);
    const top = Math.floor(rect.bottom + offsetY);
    
    agentModeDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      top: `${top}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important',
      padding: '0 !important',
    };
    console.log('[Agent Dropdown] Position: below button', { actualHeight });
  } else {
    // Pop up, use bottom positioning to ensure it's close to button
    const availableHeight = spaceAbove - offsetY - topMargin;
    if (availableHeight >= preferredDropdownHeight) {
      actualHeight = preferredDropdownHeight;
    } else {
      actualHeight = Math.max(minDropdownHeight, availableHeight);
    }
    
    const bottom = vh - rect.top + offsetY;
    
    agentModeDropdownStyle.value = {
      position: 'fixed !important',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      bottom: `${bottom}px`, // Use bottom positioning to ensure it's close to button
      maxHeight: `${actualHeight}px`,
      transform: 'none !important',
      margin: '0 !important',
      padding: '0 !important',
    };
    console.log('[Agent Dropdown] Position: above button', { actualHeight, bottom });
  }
};

const toggleAgentModeSelector = () => {
  // If Agent is not ready, show prompt
  if (!settingsStore.isAgentReady && !isAgentEnabled.value) {
    toggleAgentMode();
    return;
  }
  
  // Mutually exclusive
  showMention.value = false;
  showModelSelector.value = false;

  showAgentModeSelector.value = !showAgentModeSelector.value;
  if (showAgentModeSelector.value) {
    // Update position multiple times to ensure accuracy
    nextTick(() => {
      updateAgentModeDropdownPosition();
      requestAnimationFrame(() => {
        updateAgentModeDropdownPosition();
        setTimeout(() => {
          updateAgentModeDropdownPosition();
        }, 50);
      });
    });
  }
}

const selectAgentMode = (mode: 'normal' | 'agent') => {
  if (mode === 'agent' && !settingsStore.isAgentReady) {
    toggleAgentMode();
    showAgentModeSelector.value = false;
    return;
  }
  
  const shouldEnableAgent = mode === 'agent';
  if (shouldEnableAgent !== isAgentEnabled.value) {
    settingsStore.toggleAgent(shouldEnableAgent);
    MessagePlugin.success(shouldEnableAgent ? t('input.messages.agentSwitchedOn') : t('input.messages.agentSwitchedOff'));
  }
  showAgentModeSelector.value = false;
}

const clearvalue = () => {
  query.value = "";
}

const onKeydown = (val: string, event: { e: { preventDefault(): unknown; keyCode: number; shiftKey: any; ctrlKey: any; }; }) => {
  if (showMention.value) {
    if (event.e.keyCode === 38) { // Up
      event.e.preventDefault();
      mentionActiveIndex.value = Math.max(0, mentionActiveIndex.value - 1);
      return;
    }
    if (event.e.keyCode === 40) { // Down
      event.e.preventDefault();
      mentionActiveIndex.value = Math.min(mentionItems.value.length - 1, mentionActiveIndex.value + 1);
      return;
    }
    if (event.e.keyCode === 13) { // Enter
      event.e.preventDefault();
      if (mentionItems.value[mentionActiveIndex.value]) {
        onMentionSelect(mentionItems.value[mentionActiveIndex.value]);
      }
      return;
    }
    if (event.e.keyCode === 27) { // Esc
        showMention.value = false;
        return;
    }
  }

  // Backspace: when input box is empty and there are selected items, delete the last selected item
  if (event.e.keyCode === 8) { // Backspace
    const textarea = getTextareaEl();
    if (textarea && textarea.selectionStart === 0 && textarea.selectionEnd === 0 && query.value === '') {
      const items = allSelectedItems.value;
      if (items.length > 0) {
        event.e.preventDefault();
        const lastItem = items[items.length - 1];
        removeSelectedItem(lastItem);
        return;
      }
    }
  }

  if ((event.e.keyCode == 13 && event.e.shiftKey) || (event.e.keyCode == 13 && event.e.ctrlKey)) {
    return;
  }
  if (event.e.keyCode == 13) {
    event.e.preventDefault();
    createSession(val)
  }
}

const handleGoToWebSearchSettings = () => {
  uiStore.openSettings('websearch');
  if (route.path !== '/platform/settings') {
    router.push('/platform/settings');
  }
};

const handleGoToAgentSettings = () => {
  // Use uiStore to open settings and jump to agent section
  uiStore.openSettings('agent');
  // If not on settings page, navigate to settings page
  if (route.path !== '/platform/settings') {
    router.push('/platform/settings');
  }
}

// Get reasons why Agent is not ready
const getAgentNotReadyReasons = (): string[] => {
  const reasons: string[] = []
  const config = settingsStore.agentConfig || { allowedTools: [] }
  const models = settingsStore.conversationModels || { summaryModelId: '', rerankModelId: '' }
  
  if (!config.allowedTools || config.allowedTools.length === 0) {
    reasons.push(t('input.agentMissingAllowedTools'))
  }
  if (!models.summaryModelId || models.summaryModelId.trim() === '') {
    reasons.push(t('input.agentMissingSummaryModel'))
  }
  if (!models.rerankModelId || models.rerankModelId.trim() === '') {
    reasons.push(t('input.agentMissingRerankModel'))
  }
  
  return reasons
}

const toggleAgentMode = () => {
  // If enabling Agent, check if it's ready first
  // Note: isAgentReady is calculated from store, ensure config in store is up to date
  if (!isAgentEnabled.value) {
    // Try to enable Agent, check if it's ready first
    const agentReady = settingsStore.isAgentReady
    if (!agentReady) {
      const reasons = getAgentNotReadyReasons()
      const reasonsText = reasons.join(', ')
      
      // Create custom message with jump link
      const messageContent = h('div', { style: 'display: flex; flex-direction: column; gap: 8px; max-width: 320px;' }, [
        h('span', { style: 'color: #333; line-height: 1.5;' }, t('input.messages.agentNotReadyDetail', { reasons: reasonsText })),
        h('a', {
          href: '#',
          onClick: (e: Event) => {
            e.preventDefault();
            handleGoToAgentSettings();
          },
          style: 'color: #07C05F; text-decoration: none; font-weight: 500; cursor: pointer; align-self: flex-start;',
          onMouseenter: (e: Event) => {
            (e.target as HTMLElement).style.textDecoration = 'underline';
          },
          onMouseleave: (e: Event) => {
            (e.target as HTMLElement).style.textDecoration = 'none';
          }
        }, t('input.goToSettings'))
      ]);
      
      MessagePlugin.warning({
        content: () => messageContent,
        duration: 5000
      });
      return
    }
  }
  
  // Toggle Agent status normally
  settingsStore.toggleAgent(!isAgentEnabled.value);
  const message = isAgentEnabled.value ? t('input.messages.agentEnabled') : t('input.messages.agentDisabled');
  MessagePlugin.success(message);
}

const toggleWebSearch = () => {
  // Mutually exclusive: although not a popup, closing others during operation improves experience
  showMention.value = false;
  showModelSelector.value = false;
  showAgentModeSelector.value = false;

  if (!isWebSearchConfigured.value) {
    const messageContent = h('div', { style: 'display: flex; flex-direction: column; gap: 6px; max-width: 280px;' }, [
      h('span', { style: 'color: #333; line-height: 1.5;' }, t('input.messages.webSearchNotConfigured')),
      h('a', {
        href: '#',
        onClick: (e: Event) => {
          e.preventDefault();
          handleGoToWebSearchSettings();
        },
        style: 'color: #07C05F; text-decoration: none; font-weight: 500; cursor: pointer; align-self: flex-start;',
        onMouseenter: (e: Event) => {
          (e.target as HTMLElement).style.textDecoration = 'underline';
        },
        onMouseleave: (e: Event) => {
          (e.target as HTMLElement).style.textDecoration = 'none';
        }
      }, t('input.goToSettings'))
    ]);
    MessagePlugin.warning({
      content: () => messageContent,
      duration: 5000
    });
    return;
  }

  const currentValue = settingsStore.isWebSearchEnabled;
  const newValue = !currentValue;
  settingsStore.toggleWebSearch(newValue);
  MessagePlugin.success(newValue ? t('input.messages.webSearchEnabled') : t('input.messages.webSearchDisabled'));
};

const toggleKbSelector = () => {
  showKbSelector.value = !showKbSelector.value;
}

const removeKb = (kbId: string) => {
  settingsStore.removeKnowledgeBase(kbId);
}

const handleStop = async () => {
  if (!props.sessionId) {
    MessagePlugin.warning(t('input.messages.sessionMissing'));
    return;
  }
  
  if (!props.assistantMessageId) {
    console.error('[Stop] Assistant message ID is empty');
    MessagePlugin.warning(t('input.messages.messageMissing'));
    return;
  }
  
  console.log('[Stop] Stopping generation for message:', props.assistantMessageId);
  
  // Send stop event to notify parent component to clear loading status immediately
  emit('stop-generation');
  
  try {
    await stopSession(props.sessionId, props.assistantMessageId);
    MessagePlugin.success(t('input.messages.stopSuccess'));
  } catch (error) {
    console.error('Failed to stop session:', error);
    MessagePlugin.error(t('input.messages.stopFailed'));
  }
}

onBeforeRouteUpdate((to, from, next) => {
  clearvalue()
  next()
})

</script>
<template>
  <div class="answers-input">
    <!-- Rich text input box container -->
    <div class="rich-input-container">
      <!-- Selected knowledge bases and file tags (displayed at the top inside input box) -->
      <div v-if="allSelectedItems.length > 0" class="selected-tags-inline">
        <span 
          v-for="item in allSelectedItems" 
          :key="item.id" 
          class="inline-tag"
          :class="[
            item.type === 'kb' ? (item.kbType === 'faq' ? 'faq-tag' : 'kb-tag') : 'file-tag'
          ]"
        >
          <span class="tag-icon">
            <t-icon v-if="item.type === 'kb'" :name="item.kbType === 'faq' ? 'chat-bubble-help' : 'folder'" />
            <t-icon v-else name="file" />
          </span>
          <span class="tag-name">{{ item.name }}</span>
          <span class="tag-remove" @click="removeSelectedItem(item)">×</span>
        </span>
      </div>
      
      <!-- Actual input box -->
      <t-textarea 
        ref="textareaRef"
        v-model="query" 
        :placeholder="inputPlaceholder" 
        name="description" 
        :autosize="true" 
        @keydown="onKeydown" 
        @input="onInput"
        @compositionstart="onCompositionStart"
        @compositionend="onCompositionEnd"
      />
    </div>
    
    <!-- Mention Selector -->
    <Teleport to="body">
      <MentionSelector
        :visible="showMention"
        :style="mentionStyle"
        :items="mentionItems"
        :hasMore="mentionHasMore"
        :loading="mentionLoading"
        v-model:activeIndex="mentionActiveIndex"
        @select="onMentionSelect"
        @loadMore="loadMoreMentionItems"
      />
    </Teleport>
    
    <!-- Control bar -->
    <div class="control-bar">
      <!-- Left control buttons -->
      <div class="control-left">
        <!-- Agent mode toggle button -->
        <div 
          ref="agentModeButtonRef"
          class="control-btn agent-mode-btn"
          :class="{ 
            'active': isAgentEnabled,
            'agent-active': isAgentEnabled
          }"
          @click.stop="toggleAgentModeSelector"
        >
          <img 
            v-if="isAgentEnabled"
            :src="getImgSrc('agent-active.svg')" 
            :alt="$t('input.agentMode')" 
            class="control-icon agent-icon"
          />
          <svg 
            v-else
            width="18" 
            height="18" 
            viewBox="0 0 24 24" 
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="control-icon normal-mode-icon"
          >
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <span class="agent-mode-text">
            {{ isAgentEnabled ? $t('input.agentMode') : $t('input.normalMode') }}
          </span>
          <svg 
            width="12" 
            height="12" 
            viewBox="0 0 12 12" 
            fill="currentColor"
            class="dropdown-arrow"
            :class="{ 'rotate': showAgentModeSelector }"
          >
            <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z"/>
          </svg>
        </div>

        <!-- Agent mode selection dropdown menu -->
        <Teleport to="body">
          <div v-if="showAgentModeSelector" class="agent-mode-selector-overlay" @click="closeAgentModeSelector">
            <div 
              class="agent-mode-selector-dropdown"
              :style="agentModeDropdownStyle"
              @click.stop
            >
              <div 
                class="agent-mode-option"
                :class="{ 'selected': !isAgentEnabled }"
                @click="selectAgentMode('normal')"
              >
                <div class="agent-mode-option-main">
                  <span class="agent-mode-option-name">{{ $t('input.normalMode') }}</span>
                  <span class="agent-mode-option-desc">{{ $t('input.normalModeDesc') }}</span>
                </div>
                <svg 
                  v-if="!isAgentEnabled"
                  width="16" 
                  height="16" 
                  viewBox="0 0 16 16" 
                  fill="currentColor"
                  class="check-icon"
                >
                  <path d="M13.5 4.5L6 12L2.5 8.5L3.5 7.5L6 10L12.5 3.5L13.5 4.5Z"/>
                </svg>
              </div>
              <div 
                class="agent-mode-option"
                :class="{ 
                  'selected': isAgentEnabled,
                  'disabled': !settingsStore.isAgentReady && !isAgentEnabled 
                }"
                @click="selectAgentMode('agent')"
              >
                <div class="agent-mode-option-main">
                  <span class="agent-mode-option-name">{{ $t('input.agentMode') }}</span>
                  <span class="agent-mode-option-desc">{{ $t('input.agentModeDesc') }}</span>
                </div>
                <svg 
                  v-if="isAgentEnabled"
                  width="16" 
                  height="16" 
                  viewBox="0 0 16 16" 
                  fill="currentColor"
                  class="check-icon"
                >
                  <path d="M13.5 4.5L6 12L2.5 8.5L3.5 7.5L6 10L12.5 3.5L13.5 4.5Z"/>
                </svg>
                <div v-if="!settingsStore.isAgentReady && !isAgentEnabled" class="agent-mode-warning">
                  <t-tooltip :content="$t('input.agentNotReadyTooltip')" placement="left">
                    <t-icon name="error-circle" class="warning-icon" />
                  </t-tooltip>
                </div>
              </div>
              <div v-if="!settingsStore.isAgentReady && !isAgentEnabled" class="agent-mode-footer">
                <a 
                  href="#"
                  @click.prevent="handleGoToAgentSettings"
                  class="agent-mode-link"
                >
                  {{ $t('input.goToSettings') }}
                </a>
              </div>
            </div>
          </div>
        </Teleport>

        <!-- WebSearch toggle button -->
        <t-tooltip placement="top">
          <template #content>
            <span v-if="isWebSearchConfigured">{{ isWebSearchEnabled ? $t('input.webSearch.toggleOff') : $t('input.webSearch.toggleOn') }}</span>
            <div v-else class="websearch-tooltip-disabled">
              <span>{{ $t('input.webSearch.notConfigured') }}</span>
              <a href="#" @click.prevent="handleGoToWebSearchSettings">{{ $t('input.goToSettings') }}</a>
            </div>
          </template>
          <div 
            class="control-btn websearch-btn"
            :class="{ 'active': isWebSearchEnabled && isWebSearchConfigured, 'disabled': !isWebSearchConfigured }"
            @click.stop="toggleWebSearch"
          >
            <svg 
              width="18" 
              height="18" 
              viewBox="0 0 18 18" 
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              class="control-icon websearch-icon"
              :class="{ 'active': isWebSearchEnabled && isWebSearchConfigured }"
            >
              <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.2" fill="none"/>
              <path d="M 9 2 A 3.5 7 0 0 0 9 16" stroke="currentColor" stroke-width="1.2" fill="none"/>
              <path d="M 9 2 A 3.5 7 0 0 1 9 16" stroke="currentColor" stroke-width="1.2" fill="none"/>
              <line x1="2.94" y1="5.5" x2="15.06" y2="5.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
              <line x1="2.94" y1="12.5" x2="15.06" y2="12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
            </svg>
          </div>
        </t-tooltip>

        <!-- @ Knowledge base/file selection button -->
        <t-tooltip :content="allSelectedItems.length > 0 ? $t('input.knowledgeBaseWithCount', { count: allSelectedItems.length }) : $t('input.knowledgeBase')">
          <div 
            ref="atButtonRef"
            class="control-btn kb-btn"
            :class="{ 'active': allSelectedItems.length > 0 }"
            @click.stop
            @mousedown.prevent="triggerMention"
          >
            <img :src="getImgSrc('at-icon.svg')" alt="@" class="control-icon" />
            <span v-if="allSelectedItems.length > 0" class="kb-count">{{ allSelectedItems.length }}</span>
          </div>
        </t-tooltip>

        <!-- Model display -->
        <div class="model-display">
          <div
            ref="modelButtonRef"
            class="model-selector-trigger"
            @click.stop="toggleModelSelector"
          >
            <span class="model-selector-name">
              {{ selectedModel?.name || $t('input.notConfigured') }}
            </span>
            <svg 
              width="12" 
              height="12" 
              viewBox="0 0 12 12" 
              fill="currentColor"
              class="model-dropdown-arrow"
              :class="{ 'rotate': showModelSelector }"
            >
              <path d="M2.5 4.5L6 8L9.5 4.5H2.5Z"/>
            </svg>
          </div>
        </div>
      </div>

      <Teleport to="body">
        <div v-if="showModelSelector" class="model-selector-overlay" @click="closeModelSelector">
            <div class="model-selector-dropdown" :style="modelDropdownStyle" @click.stop>
            <div class="model-selector-header">
              <span>{{ $t('conversationSettings.models.chatGroupLabel') }}</span>
              <button class="model-selector-add" type="button" @click="handleModelChange('__add_model__')">
                <span class="add-icon">+</span>
                  <span class="add-text">{{ $t('input.addModel') }}</span>
              </button>
            </div>
            <div class="model-selector-content">
              <div
                v-for="model in availableModels"
                :key="model.id"
                class="model-option"
                :class="{ selected: model.id === selectedModelId }"
                @click="handleModelChange(model.id || '')"
              >
                <div class="model-option-main">
                  <span class="model-option-name">{{ model.name }}</span>
                  <span v-if="model.source === 'remote'" class="model-badge-remote">{{ $t('input.remote') }}</span>
                  <span v-else-if="model.parameters?.parameter_size" class="model-badge-local">
                    {{ model.parameters.parameter_size }}
                  </span>
                </div>
                <div v-if="model.description" class="model-option-desc">
                  {{ model.description }}
                </div>
              </div>
              <div v-if="availableModels.length === 0" class="model-option empty">
                {{ $t('input.noModel') }}
              </div>
            </div>
          </div>
        </div>
      </Teleport>

      <!-- Right control button group -->
      <div class="control-right">
        <!-- Stop button (only shown when replying) -->
        <t-tooltip 
          v-if="isReplying"
          :content="$t('input.stopGeneration')"
          placement="top"
        >
          <div 
            @click="handleStop" 
            class="control-btn stop-btn"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <rect x="5" y="5" width="6" height="6" rx="1" />
            </svg>
          </div>
        </t-tooltip>

        <!-- Send button -->
      <div 
          v-if="!isReplying"
        @click="createSession(query)" 
        class="control-btn send-btn"
        :class="{ 'disabled': !query.length }"
      >
        <img src="../assets/img/sending-aircraft.svg" :alt="$t('input.send')" />
        </div>
      </div>
    </div>

    <!-- Knowledge base selection dropdown (using Teleport to body to avoid parent container positioning issues) -->
    <Teleport to="body">
    <KnowledgeBaseSelector
      v-model:visible="showKbSelector"
        :anchorEl="atButtonRef"
      @close="showKbSelector = false"
    />
    </Teleport>
  </div>
</template>
<script lang="ts">
const getImgSrc = (url: string) => {
  return new URL(`/src/assets/img/${url}`, import.meta.url).href;
}
</script>
<style scoped lang="less">
.answers-input {
  position: absolute;
  z-index: 99;
  bottom: 60px;
  left: 50%;
  transform: translateX(-400px);
}

/* Rich text input box container */
.rich-input-container {
  position: relative;
  width: 800px;
  background: var(--td-bg-color-container, #FFF);
  border-radius: 12px;
  border: 1px solid var(--td-component-border, #E7E7E7);
  box-shadow: 0 6px 6px 0 rgba(0, 0, 0, 0.04), 0 12px 12px -1px rgba(0, 0, 0, 0.08);
  
  &:focus-within {
    border-color: var(--td-brand-color, #07C05F);
  }
}

/* Selected tags (top inside input box) */
.selected-tags-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 12px 16px 8px;
  border-bottom: 1px solid var(--td-component-border, #f0f0f0);
}

.inline-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  cursor: default;
  transition: all 0.15s;
  background: var(--td-bg-color-secondarycontainer, #f3f3f3);
  border: 1px solid transparent;
  color: var(--td-text-color-primary, #333);
  
  /* KB - Document (Greenish tint) */
  &.kb-tag {
    background: rgba(16, 185, 129, 0.08);
    color: #059669;
    
    .tag-icon {
      color: #10b981;
    }
  }

  /* KB - FAQ (Blueish tint) */
  &.faq-tag {
    background: rgba(0, 82, 217, 0.08);
    color: #0052d9;
    
    .tag-icon {
      color: #0052d9;
    }
  }
  
  /* File (Orange tint) */
  &.file-tag {
    background: rgba(237, 123, 47, 0.08);
    color: #e65100;
    
    .tag-icon {
      color: #ed7b2f;
    }
  }
  
  .tag-icon {
    font-size: 14px;
    display: flex;
    align-items: center;
  }
  
  .tag-name {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: currentColor;
  }
  
  .tag-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    margin-left: 2px;
    border-radius: 50%;
    font-size: 14px;
    line-height: 1;
    cursor: pointer;
    opacity: 0.5;
    transition: opacity 0.15s, background 0.15s;
    color: currentColor;
    
    &:hover {
      opacity: 1;
      background: rgba(0, 0, 0, 0.1);
    }
  }
}

:deep(.t-textarea__inner) {
  width: 100%;
  max-height: 200px !important;
  min-height: 120px !important;
  resize: none;
  color: var(--td-text-color-primary, #000000e6);
  font-size: 16px;
  font-weight: 400;
  line-height: 24px;
  font-family: var(--td-font-family, "PingFang SC");
  padding: 12px 16px 56px 16px;
  border-radius: 0 0 12px 12px;
  border: none;
  box-sizing: border-box;
  background: transparent;
  box-shadow: none;

  &:focus {
    border: none;
    box-shadow: none;
  }

  &::placeholder {
    color: var(--td-text-color-placeholder, #00000066);
    font-family: var(--td-font-family, "PingFang SC");
    font-size: 16px;
    font-weight: 400;
    line-height: 24px;
  }
}

/* textarea style when no tags are selected */
.rich-input-container:not(:has(.selected-tags-inline)) :deep(.t-textarea__inner) {
  border-radius: 12px;
  padding-top: 16px;
}

/* Control bar */
.control-bar {
  position: absolute;
  bottom: 12px;
  left: 16px;
  right: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
  max-height: 56px;
  z-index: 10;
  background: linear-gradient(to bottom, rgba(255, 255, 255, 0) 0%, var(--td-bg-color-container, #fff) 40%, var(--td-bg-color-container, #fff) 100%);
  pointer-events: auto;
  padding-top: 8px;
}

.control-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  flex-wrap: wrap;
  min-width: 0;
}

.control-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer, #f5f5f5);
  cursor: pointer;
  transition: background 0.12s;
  user-select: none;
  flex-shrink: 0;

  &:hover {
    background: var(--td-bg-color-secondarycontainer-hover, #e6e6e6);
  }

  &.disabled {
    opacity: 0.5;
    cursor: not-allowed;
    
    &:hover {
      background: var(--td-bg-color-secondarycontainer, #f5f5f5);
    }
  }
}

.agent-mode-btn {
  height: 28px;
  padding: 0 10px;
  min-width: auto;
  font-weight: 500;
  border: 1px solid transparent;
  transition: background 0.12s, border-color 0.12s;
  position: relative;
  
  &.active,
  &.agent-active {
    background: linear-gradient(135deg, rgba(16, 185, 129, 0.15) 0%, rgba(16, 185, 129, 0.1) 100%);
    border-color: rgba(16, 185, 129, 0.4);
    box-shadow: 0 2px 6px rgba(16, 185, 129, 0.12);
    
    .agent-mode-text {
      color: #07C05F;
      font-weight: 600;
    }
    
    .agent-icon {
      filter: brightness(0) saturate(100%) invert(58%) sepia(87%) saturate(1234%) hue-rotate(95deg) brightness(98%) contrast(89%);
    }
    
    .dropdown-arrow {
      color: #07C05F;
    }
    
    &:hover {
      background: linear-gradient(135deg, rgba(16, 185, 129, 0.2) 0%, rgba(16, 185, 129, 0.15) 100%);
      border-color: rgba(16, 185, 129, 0.6);
    }
  }
  
  &:not(.agent-active) {
    background: var(--td-bg-color-container, #fff);
    border-color: var(--td-component-border, #e0e0e0);
    
    .agent-mode-text {
      color: var(--td-text-color-secondary, #666);
    }
    
    .normal-mode-icon {
      color: var(--td-text-color-secondary, #666);
    }
    
    &:hover {
      background: var(--td-bg-color-container-hover, #fff);
      border-color: var(--td-component-stroke, #b0b0b0);
    }
  }
}

.agent-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.agent-mode-text {
  font-size: 13px;
  color: var(--td-text-color-secondary, #666);
  font-weight: 500;
  white-space: nowrap;
  margin: 0 4px;
}

.control-icon {
  width: 18px;
  height: 18px;
}

.kb-btn {
  height: 28px;
  padding: 0 10px;
  min-width: auto;
  position: relative;
  
  &.active {
    background: rgba(16, 185, 129, 0.1);
    color: #07C05F;
    
    &:hover {
      background: rgba(16, 185, 129, 0.15);
    }
  }
}

.kb-count {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  background: #07C05F;
  color: white;
  font-size: 10px;
  font-weight: 600;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.kb-btn-text {
  font-size: 13px;
  color: var(--td-text-color-secondary, #666);
  font-weight: 500;
  white-space: nowrap;
}

.kb-btn.active .kb-btn-text {
  color: #07C05F;
}

.websearch-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  min-width: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  
  &.active {
    background: rgba(16, 185, 129, 0.1);
    
    .websearch-icon {
      color: #07C05F;
    }
    
    &:hover {
      background: rgba(16, 185, 129, 0.15);
    }
  }
  
  &:not(.active) {
    .websearch-icon {
      color: var(--td-text-color-secondary, #666);
    }
    
    &:hover {
      background: var(--td-bg-color-secondarycontainer-hover, #f0f0f0);
      
      .websearch-icon {
        color: var(--td-text-color-primary, #333);
      }
    }
  }
}

:global(.websearch-tooltip-disabled) {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 220px;
  font-size: 12px;
  color: var(--td-text-color-secondary, #666);
}

:global(.websearch-tooltip-disabled a) {
  color: #07C05F;
  font-weight: 500;
  text-decoration: none;
}

:global(.websearch-tooltip-disabled a:hover) {
  text-decoration: underline;
}

.websearch-icon {
  width: 18px;
  height: 18px;
}

.dropdown-arrow {
  width: 10px;
  height: 10px;
  margin-left: 2px;
  transition: transform 0.12s;
  
  &.rotate {
    transform: rotate(180deg);
  }
}

.control-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stop-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  background: rgba(16, 185, 129, 0.08);
  color: #07C05F;
  border: 1.5px solid rgba(16, 185, 129, 0.2);
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  
  &:hover {
    background: rgba(16, 185, 129, 0.12);
    border-color: #07C05F;
  }
  
  &:active {
    background: rgba(16, 185, 129, 0.15);
  }
  
  svg {
    display: none;
  }
  
  &::before {
    content: '';
    width: 12px;
    height: 12px;
    background: #07C05F;
    border-radius: 50%;
    display: block;
  }
}

.send-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  background-color: #07C05F;
  
  &:hover:not(.disabled) {
    background-color: #059669;
  }
  
  &.disabled {
    background-color: #b5eccf;
  }
  
  img {
    width: 16px;
    height: 16px;
  }
}

/* Model display style */
.model-display {
  display: flex;
  align-items: center;
  margin-left: auto;
  flex-shrink: 0;
}

.model-selector-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  min-width: 100px;
  height: 22px;
  border-radius: 6px;
  border: 1px solid rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.1);
  transition: background 0.12s, border-color 0.12s;
  cursor: pointer;
}

.model-selector-trigger:hover {
  background: rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.45);
}

.model-selector-trigger.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.model-selector-trigger.disabled:hover {
  background: rgba(16, 185, 129, 0.1);
  border-color: rgba(16, 185, 129, 0.3);
}

.model-selector-name {
  flex: 1;
  font-size: 12px;
  font-weight: 600;
  color: #07C05F;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-dropdown-arrow {
  width: 10px;
  height: 10px;
  color: #07C05F;
  flex-shrink: 0;
  transition: transform 0.12s;
  
  &.rotate {
    transform: rotate(180deg);
  }
}

.model-selector-trigger.disabled .model-dropdown-arrow {
  color: rgba(16, 185, 129, 0.4);
}

.model-selector-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
  background: transparent;
  touch-action: none;
}

.model-selector-dropdown {
  position: fixed !important;
  z-index: 9999;
  background: var(--td-bg-color-container, #fff);
  border-radius: 10px;
  box-shadow: var(--td-shadow-2, 0 6px 28px rgba(15, 23, 42, 0.08));
  border: 1px solid var(--td-component-border, #e7e9eb);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  margin: 0 !important;
  padding: 0 !important;
  transform: none !important;
}

.model-selector-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  border-bottom: 1px solid var(--td-component-border, #f2f4f5);
  background: var(--td-bg-color-secondarycontainer, #fafcfc);
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-primary, #222);
}

.model-selector-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  padding: 6px 8px;
}

.model-selector-add {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  border: 1px solid var(--td-component-border, #e1e5e6);
  background: var(--td-bg-color-container, #fff);
  color: var(--td-text-color-secondary, #52575a);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.12s;
}

.model-selector-add .add-icon {
  font-size: 12px;
  line-height: 1;
}

.model-selector-add:hover {
  border-color: #10b981;
  color: #10b981;
  background: #f0fdf6;
}

.model-option {
  padding: 6px 8px;
  cursor: pointer;
  transition: background 0.12s;
  border-radius: 6px;
  margin-bottom: 4px;
  
  &:last-child {
    margin-bottom: 0;
  }
  
  &:hover {
    background: var(--td-bg-color-container-hover, #f6f8f7);
  }
  
  &.selected {
    background: var(--td-brand-color-light, #eefdf5);
    
    .model-option-name {
      color: #10b981;
      font-weight: 600;
    }
  }
  
  &.empty {
    color: var(--td-text-color-disabled, #9aa0a6);
    cursor: default;
    text-align: center;
    padding: 20px 8px;
    
    &:hover {
      background: transparent;
    }
  }
}

.model-option-main {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 1px;
}

.model-option-name {
  font-size: 12px;
  color: var(--td-text-color-primary, #222);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.4;
}

.model-option-desc {
  font-size: 11px;
  color: var(--td-text-color-secondary, #8b9196);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 1px;
}

.model-badge-remote,
.model-badge-local {
  display: inline-block;
  padding: 1px 5px;
  font-size: 10px;
  border-radius: 3px;
  font-weight: 500;
  flex-shrink: 0;
}

.model-badge-remote {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.model-badge-local {
  background: rgba(139, 145, 150, 0.1);
  color: #52575a;
}

/* Agent mode selection dropdown menu */
.agent-mode-selector-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
  background: transparent;
  touch-action: none;
}

.agent-mode-selector-dropdown {
  position: fixed !important;
  z-index: 9999;
  background: var(--td-bg-color-container, #fff);
  border-radius: 10px;
  box-shadow: var(--td-shadow-2, 0 6px 28px rgba(15, 23, 42, 0.08));
  border: 1px solid var(--td-component-border, #e7e9eb);
  overflow: hidden;
  padding: 6px 8px;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  margin: 0 !important;
  padding: 0 !important;
  transform: none !important;
}

.agent-mode-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  cursor: pointer;
  transition: background 0.12s;
  border-radius: 6px;
  position: relative;
  margin: 4px 6px;
  
  &:hover:not(.disabled) {
    background: var(--td-bg-color-container-hover, #f6f8f7);
  }
  
  &.disabled {
    opacity: 0.6;
    cursor: not-allowed;
    
    &:hover {
      background: transparent;
    }
  }
  
  &.selected {
    background: var(--td-brand-color-light, #eefdf5);
    
    .agent-mode-option-name {
      color: #10b981;
      font-weight: 700;
    }
  }
}

.agent-mode-option-main {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}

.agent-mode-option-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-primary, #222);
  line-height: 1.4;
  transition: color 0.12s;
}

.agent-mode-option-desc {
  font-size: 11px;
  color: var(--td-text-color-secondary, #8b9196);
  line-height: 1.3;
}

.check-icon {
  width: 14px;
  height: 14px;
  color: #10b981;
  flex-shrink: 0;
  margin-left: 6px;
}

.agent-mode-warning {
  display: flex;
  align-items: center;
  margin-left: 6px;
  
  .warning-icon {
    color: #ff9800;
    font-size: 14px;
  }
}

.agent-mode-footer {
  padding: 6px 10px;
  border-top: 1px solid var(--td-component-border, #f2f4f5);
  margin-top: 2px;
  background: var(--td-bg-color-secondarycontainer, #fafcfc);
}

.agent-mode-link {
  color: #10b981;
  text-decoration: none;
  font-size: 11px;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  transition: all 0.12s;
  
  &:hover {
    color: #059669;
    text-decoration: underline;
  }
}
</style>


