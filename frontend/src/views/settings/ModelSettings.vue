<template>
  <div class="model-settings">
    <div class="section-header">
      <h2>{{ $t('modelSettings.title') }}</h2>
      <p class="section-description">{{ $t('modelSettings.description') }}</p>
      
      <!-- Built-in model info -->
      <div class="builtin-models-info">
        <div class="info-box">
          <div class="info-header">
            <t-icon name="info-circle" class="info-icon" />
            <span class="info-title">Built-in Models</span>
          </div>
          <div class="info-content">
            <p>Built-in models are visible to all tenants, hide sensitive information, and cannot be edited or deleted.</p>
            <p class="doc-link">
              <t-icon name="link" class="link-icon" />
              <a href="https://github.com/Tencent/WeKnora/blob/main/docs/BUILTIN_MODELS.md" target="_blank" rel="noopener noreferrer">
                View the built-in model management guide
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Conversational models -->
    <div class="model-category-section" data-model-type="chat">
      <div class="category-header">
        <div class="header-info">
          <h3>{{ $t('modelSettings.chat.title') }}</h3>
          <p>{{ $t('modelSettings.chat.desc') }}</p>
        </div>
        <t-button size="small" theme="primary" @click="openAddDialog('chat')" class="add-model-btn">
          <template #icon>
            <t-icon name="add" class="add-icon" />
          </template>
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
      
      <div v-if="chatModels.length > 0" class="model-list-container">
        <div v-for="model in chatModels" :key="model.id" class="model-card" :class="{ 'builtin-model': model.isBuiltin }">
          <div class="model-info">
            <div class="model-name">
              {{ model.name }}
              <t-tag v-if="model.isBuiltin" theme="primary" size="small">Built-in</t-tag>
            </div>
            <div class="model-meta">
              <span class="source-tag">{{ model.source === 'local' ? 'Ollama' : $t('modelSettings.source.remote') }}</span>
              <!-- <span class="model-id">{{ model.modelName }}</span> -->
            </div>
          </div>
          <div class="model-actions">
            <t-dropdown 
              :options="getModelOptions('chat', model)" 
              @click="(data: any) => handleMenuAction(data, 'chat', model)"
              placement="bottom-right"
              attach="body"
            >
              <t-button variant="text" shape="square" size="small" class="more-btn">
                <t-icon name="more" />
              </t-button>
            </t-dropdown>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <p class="empty-text">{{ $t('modelSettings.chat.empty') }}</p>
        <t-button theme="default" variant="outline" size="small" @click="openAddDialog('chat')">
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
    </div>

    <!-- Embedding models -->
    <div class="model-category-section" data-model-type="embedding">
      <div class="category-header">
        <div class="header-info">
          <h3>{{ $t('modelSettings.embedding.title') }}</h3>
          <p>{{ $t('modelSettings.embedding.desc') }}</p>
        </div>
        <t-button size="small" theme="primary" @click="openAddDialog('embedding')" class="add-model-btn">
          <template #icon>
            <t-icon name="add" class="add-icon" />
          </template>
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
      
      <div v-if="embeddingModels.length > 0" class="model-list-container">
        <div v-for="model in embeddingModels" :key="model.id" class="model-card" :class="{ 'builtin-model': model.isBuiltin }">
          <div class="model-info">
            <div class="model-name">
              {{ model.name }}
              <t-tag v-if="model.isBuiltin" theme="primary" size="small">Built-in</t-tag>
            </div>
            <div class="model-meta">
              <span class="source-tag">{{ model.source === 'local' ? 'Ollama' : $t('modelSettings.source.remote') }}</span>
              <!-- <span class="model-id">{{ model.modelName }}</span> -->
              <span v-if="model.dimension" class="dimension">{{ $t('model.editor.dimensionLabel') }}: {{ model.dimension }}</span>
            </div>
          </div>
          <div class="model-actions">
            <t-dropdown 
              :options="getModelOptions('embedding', model)" 
              @click="(data: any) => handleMenuAction(data, 'embedding', model)"
              placement="bottom-right"
              attach="body"
            >
              <t-button variant="text" shape="square" size="small" class="more-btn">
                <t-icon name="more" />
              </t-button>
            </t-dropdown>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <p class="empty-text">{{ $t('modelSettings.embedding.empty') }}</p>
        <t-button theme="default" variant="outline" size="small" @click="openAddDialog('embedding')">
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
    </div>

    <!-- ReRank models -->
    <div class="model-category-section" data-model-type="rerank">
      <div class="category-header">
        <div class="header-info">
          <h3>{{ $t('modelSettings.rerank.title') }}</h3>
          <p>{{ $t('modelSettings.rerank.desc') }}</p>
        </div>
        <t-button size="small" theme="primary" @click="openAddDialog('rerank')" class="add-model-btn">
          <template #icon>
            <t-icon name="add" class="add-icon" />
          </template>
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
      
      <div v-if="rerankModels.length > 0" class="model-list-container">
        <div v-for="model in rerankModels" :key="model.id" class="model-card" :class="{ 'builtin-model': model.isBuiltin }">
          <div class="model-info">
            <div class="model-name">
              {{ model.name }}
              <t-tag v-if="model.isBuiltin" theme="primary" size="small">Built-in</t-tag>
            </div>
            <div class="model-meta">
              <span class="source-tag">{{ model.source === 'local' ? 'Ollama' : $t('modelSettings.source.remote') }}</span>
              <!-- <span class="model-id">{{ model.modelName }}</span> -->
            </div>
          </div>
          <div class="model-actions">
            <t-dropdown 
              :options="getModelOptions('rerank', model)" 
              @click="(data: any) => handleMenuAction(data, 'rerank', model)"
              placement="bottom-right"
              attach="body"
            >
              <t-button variant="text" shape="square" size="small" class="more-btn">
                <t-icon name="more" />
              </t-button>
            </t-dropdown>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <p class="empty-text">{{ $t('modelSettings.rerank.empty') }}</p>
        <t-button theme="default" variant="outline" size="small" @click="openAddDialog('rerank')">
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
    </div>

    <!-- VLLM vision models -->
    <div class="model-category-section" data-model-type="vllm">
      <div class="category-header">
        <div class="header-info">
          <h3>{{ $t('modelSettings.vllm.title') }}</h3>
          <p>{{ $t('modelSettings.vllm.desc') }}</p>
        </div>
        <t-button size="small" theme="primary" @click="openAddDialog('vllm')" class="add-model-btn">
          <template #icon>
            <t-icon name="add" class="add-icon" />
          </template>
          {{ $t('modelSettings.actions.addModel') }}
        </t-button>
      </div>
      
      <div v-if="vllmModels.length > 0" class="model-list-container">
        <div v-for="model in vllmModels" :key="model.id" class="model-card" :class="{ 'builtin-model': model.isBuiltin }">
          <div class="model-info">
            <div class="model-name">
              {{ model.name }}
              <t-tag v-if="model.isBuiltin" theme="primary" size="small">Built-in</t-tag>
            </div>
            <div class="model-meta">
              <span class="source-tag">{{ model.source === 'local' ? 'Ollama' : $t('modelSettings.source.openaiCompatible') }}</span>
              <!-- <span class="model-id">{{ model.modelName }}</span> -->
            </div>
          </div>
          <div class="model-actions">
            <t-dropdown 
              :options="getModelOptions('vllm', model)" 
              @click="(data: any) => handleMenuAction(data, 'vllm', model)"
              placement="bottom-right"
              attach="body"
            >
              <t-button variant="text" shape="square" size="small" class="more-btn">
                <t-icon name="more" />
              </t-button>
            </t-dropdown>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <p class="empty-text">No VLLM vision models yet</p>
        <t-button theme="default" variant="outline" size="small" @click="openAddDialog('vllm')">
          Add model
        </t-button>
      </div>
    </div>

    <!-- Model editor dialog -->
    <ModelEditorDialog
      v-model:visible="showDialog"
      :model-type="currentModelType"
      :model-data="editingModel"
      @confirm="handleModelSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import ModelEditorDialog from '@/components/ModelEditorDialog.vue'
import { listModels, createModel, updateModel as updateModelAPI, deleteModel as deleteModelAPI, type ModelConfig } from '@/api/model'

const { t } = useI18n()

const showDialog = ref(false)
const currentModelType = ref<'chat' | 'embedding' | 'rerank' | 'vllm'>('chat')
const editingModel = ref<any>(null)
const loading = ref(true)

// Model list data
const allModels = ref<ModelConfig[]>([])

// Filter and deduplicate models by type
const chatModels = computed(() => 
  deduplicateModels(
    allModels.value
      .filter(m => m.type === 'KnowledgeQA')
      .map(convertToLegacyFormat)
  )
)

const embeddingModels = computed(() => 
  deduplicateModels(
    allModels.value
      .filter(m => m.type === 'Embedding')
      .map(convertToLegacyFormat)
  )
)

const rerankModels = computed(() => 
  deduplicateModels(
    allModels.value
      .filter(m => m.type === 'Rerank')
      .map(convertToLegacyFormat)
  )
)

const vllmModels = computed(() => 
  deduplicateModels(
    allModels.value
      .filter(m => m.type === 'VLLM')
      .map(convertToLegacyFormat)
  )
)

// Convert backend model format to the legacy frontend shape
function convertToLegacyFormat(model: ModelConfig) {
  return {
    id: model.id!,
    name: model.name,
    source: model.source,
    modelName: model.name,  // Use display name as the model identifier
    baseUrl: model.parameters.base_url || '',
    apiKey: model.parameters.api_key || '',
    dimension: model.parameters.embedding_parameters?.dimension,
    isBuiltin: model.is_builtin || false
  }
}

// Deduplication helper: compare all fields except id and keep the first match
function deduplicateModels(models: any[]) {
  const seen = new Map<string, any>()
  
  return models.filter(model => {
    // Build a signature that excludes the id for comparison
    const signature = JSON.stringify({
      name: model.name,
      source: model.source,
      modelName: model.modelName,
      baseUrl: model.baseUrl,
      apiKey: model.apiKey,
      dimension: model.dimension
    })
    
    if (seen.has(signature)) {
      return false
    }
    
    seen.set(signature, model)
    return true
  })
}

// Load the model list
const loadModels = async () => {
  loading.value = true
  try {
    // Fetch all models regardless of type
    const models = await listModels()
    allModels.value = models
  } catch (error: any) {
    console.error('Failed to load model list:', error)
    MessagePlugin.error(error.message || 'Failed to load model list')
  } finally {
    loading.value = false
  }
}

// Open the add model dialog
const openAddDialog = (type: 'chat' | 'embedding' | 'rerank' | 'vllm') => {
  currentModelType.value = type
  editingModel.value = null
  showDialog.value = true
}

// Edit a model
const editModel = (type: 'chat' | 'embedding' | 'rerank' | 'vllm', model: any) => {
  // Built-in models cannot be edited
  if (model.isBuiltin) {
    MessagePlugin.warning('Built-in models cannot be edited')
    return
  }
  currentModelType.value = type
  editingModel.value = { ...model }
  showDialog.value = true
}

// Save model
const handleModelSave = async (modelData: any) => {
  try {
    // Validate fields
    if (!modelData.modelName || !modelData.modelName.trim()) {
      MessagePlugin.warning(t('modelSettings.toasts.nameRequired'))
      return
    }
    
    if (modelData.modelName.trim().length > 100) {
      MessagePlugin.warning(t('modelSettings.toasts.nameTooLong'))
      return
    }
    
    // Remote models require a baseUrl
    if (modelData.source === 'remote') {
      if (!modelData.baseUrl || !modelData.baseUrl.trim()) {
        MessagePlugin.warning(t('modelSettings.toasts.baseUrlRequired'))
        return
      }
      
      // Validate Base URL format
      try {
        new URL(modelData.baseUrl.trim())
      } catch {
        MessagePlugin.warning(t('modelSettings.toasts.baseUrlInvalid'))
        return
      }
    }
    
    // Embedding models must include a dimension
    if (currentModelType.value === 'embedding') {
      if (!modelData.dimension || modelData.dimension < 128 || modelData.dimension > 4096) {
        MessagePlugin.warning(t('modelSettings.toasts.dimensionInvalid'))
        return
      }
    }
    
    // Convert frontend format to backend payload
    const apiModelData: ModelConfig = {
      name: modelData.modelName.trim(), // Use modelName as name and trim whitespace
      type: getModelType(currentModelType.value),
      source: modelData.source,
      description: '',
      parameters: {
        base_url: modelData.baseUrl?.trim() || '',
        api_key: modelData.apiKey?.trim() || '',
        ...(currentModelType.value === 'embedding' && modelData.dimension ? {
          embedding_parameters: {
            dimension: modelData.dimension,
            truncate_prompt_tokens: 0
          }
        } : {})
      }
    }

    if (editingModel.value && editingModel.value.id) {
      // Update existing model
      await updateModelAPI(editingModel.value.id, apiModelData)
      MessagePlugin.success(t('modelSettings.toasts.updated'))
    } else {
      // Create a new model
      await createModel(apiModelData)
      MessagePlugin.success(t('modelSettings.toasts.added'))
    }
    
    // Reload the model list
    await loadModels()
  } catch (error: any) {
    console.error('Failed to save model:', error)
    MessagePlugin.error(error.message || t('modelSettings.toasts.saveFailed'))
  }
}

// Delete model
const deleteModel = async (type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) => {
  // Check if the model is built-in
  const model = allModels.value.find(m => m.id === modelId)
  if (model?.is_builtin) {
    MessagePlugin.warning('Built-in models cannot be deleted')
    return
  }
  
  try {
    await deleteModelAPI(modelId)
    MessagePlugin.success(t('modelSettings.toasts.deleted'))
    // Reload the model list
    await loadModels()
  } catch (error: any) {
    console.error('Failed to delete model:', error)
    MessagePlugin.error(error.message || t('modelSettings.toasts.deleteFailed'))
  }
}

// Get model action menu options
const getModelOptions = (type: 'chat' | 'embedding' | 'rerank' | 'vllm', model: any) => {
  const options: any[] = []
  
  // Built-in models cannot be edited or deleted
  if (model.isBuiltin) {
    return options
  }
  
  // Edit option
  options.push({
    content: t('common.edit'),
    value: `edit-${type}-${model.id}`
  })
  
  // Delete option
  options.push({
    content: t('common.delete'),
    value: `delete-${type}-${model.id}`,
    theme: 'error'
  })
  
  return options
}

// Handle menu actions
const handleMenuAction = (data: { value: string }, type: 'chat' | 'embedding' | 'rerank' | 'vllm', model: any) => {
  const value = data.value
  
  if (value.indexOf('edit-') === 0) {
    editModel(type, model)
  } else if (value.indexOf('delete-') === 0) {
    // Confirm with the user
    if (confirm(t('modelSettings.confirmDelete'))) {
      deleteModel(type, model.id)
    }
  }
}

// Map to backend model types
function getModelType(type: 'chat' | 'embedding' | 'rerank' | 'vllm'): 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM' {
  const typeMap = {
    chat: 'KnowledgeQA' as const,
    embedding: 'Embedding' as const,
    rerank: 'Rerank' as const,
    vllm: 'VLLM' as const
  }
  return typeMap[type]
}

// Load the model list when the component mounts
onMounted(() => {
  loadModels()
})
</script>

<style lang="less" scoped>
.model-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: #333333;
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: #666666;
    margin: 0 0 20px 0;
    line-height: 1.5;
  }
}

.model-category-section {
  margin-bottom: 32px;
  padding-bottom: 32px;
  border-bottom: 1px solid #e5e7eb;

  &:last-child {
    margin-bottom: 0;
    padding-bottom: 0;
    border-bottom: none;
  }
}

.category-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;

  .header-info {
    flex: 1;

    h3 {
      font-size: 15px;
      font-weight: 500;
      color: #333333;
      margin: 0 0 4px 0;
    }

    p {
      font-size: 13px;
      color: #666666;
      margin: 0;
      line-height: 1.5;
    }
  }
}

// Styling tweaks for the add-model button
:deep(.add-model-btn) {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  height: 32px;
  padding: 0 16px;
  font-size: 14px;
  flex-shrink: 0;

  .add-icon {
    font-size: 14px;
    width: 14px;
    height: 14px;
  }
}

.model-list-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.model-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #fafafa;
  transition: all 0.15s ease;
  position: relative;
  overflow: visible;

  &:hover {
    border-color: #07C05F;
    background: #f9fdfb;
    box-shadow: 0 1px 4px rgba(7, 192, 95, 0.08);
  }

  // Built-in model styles
  &.builtin-model {
    background: #f8f9fa;
    border-color: #d9d9d9;

    &:hover {
      border-color: #c0c0c0;
      background: #f5f6f7;
      box-shadow: none;
    }

    .model-info {
      .model-name {
        color: #666666;
      }

      .model-meta {
        .source-tag {
          background: #e5e5e5;
          color: #999999;
        }
      }
    }
  }
}

.model-info {
  flex: 1;
  min-width: 0;

  .model-name {
    font-size: 14px;
    font-weight: 500;
    color: #333333;
    margin-bottom: 6px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .model-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12px;
    color: #666666;

    .source-tag {
      padding: 2px 8px;
      background: #e5e7eb;
      border-radius: 3px;
      font-size: 11px;
      font-weight: 500;
    }

    .model-id {
      font-family: monospace;
      color: #666666;
    }

    .dimension {
      color: #999999;
    }
  }
}

.model-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s ease;
  position: relative;
  z-index: 1001;

  .more-btn {
    color: #999999;
    padding: 4px;

    &:hover {
      background: #f5f7fa;
      color: #333333;
    }
  }
}

.model-card:hover .model-actions {
  opacity: 1;
}

.empty-state {
  padding: 48px 0;
  text-align: center;

  .empty-text {
    font-size: 13px;
    color: #999999;
    margin: 0 0 16px 0;
  }
}

.builtin-models-info {
  margin-top: 16px;

  .info-box {
    background: #f0fdf6;
    border: 1px solid #d1fae5;
    border-left: 3px solid #07C05F;
    border-radius: 6px;
    padding: 16px;
  }

  .info-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .info-icon {
      font-size: 16px;
      color: #07C05F;
      flex-shrink: 0;
    }

    .info-title {
      font-size: 14px;
      font-weight: 500;
      color: #059669;
    }
  }

  .info-content {
    font-size: 13px;
    line-height: 1.6;
    color: #065f46;

    p {
      margin: 0 0 6px 0;

      &:last-child {
        margin-bottom: 0;
      }

      &.doc-link {
        margin-top: 10px;
        display: flex;
        align-items: center;
        gap: 6px;

        .link-icon {
          font-size: 13px;
          color: #07C05F;
          flex-shrink: 0;
        }

        a {
          color: #07C05F;
          text-decoration: none;
          font-weight: 500;
          transition: color 0.15s;

          &:hover {
            color: #059669;
            text-decoration: underline;
          }
        }
      }
    }
  }
}

// TDesign component overrides
:deep(.t-button) {
  &.add-model-btn {
    border-radius: 6px;
    font-weight: 500;
    transition: all 0.15s ease;

    &:hover {
      background: #06b04d;
      border-color: #06b04d;
    }

    &:active {
      background: #059642;
      border-color: #059642;
    }
  }

  &.t-size-s {
    height: 32px;
    padding: 0 12px;
    font-size: 13px;
    border-radius: 6px;

    &.t-button--variant-outline {
      color: #666666;
      border-color: #d9d9d9;

      &:hover {
        color: #07C05F;
        border-color: #07C05F;
        background: rgba(7, 192, 95, 0.04);
      }
    }
  }
}

// Tag style tweaks
:deep(.t-tag) {
  border-radius: 3px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  border: none;

  &.t-tag--theme-primary {
    background: #e0f2fe;
    color: #0369a1;
  }

  &.t-tag--theme-success {
    background: #dcfce7;
    color: #059669;
  }

  &.t-size-s {
    height: 20px;
    line-height: 16px;
  }
}

// Dropdown menu tweaks
:deep(.t-popup__content) {
  .t-dropdown__menu {
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
    padding: 4px;
    min-width: 140px;
  }

  .t-dropdown__item {
    padding: 8px 12px;
    border-radius: 4px;
    margin: 2px 0;
    font-size: 13px;
    color: #333333;
    transition: all 0.15s ease;
    cursor: pointer;

    &:hover {
      background: #f5f7fa;
      color: #07C05F;
    }

    &:active {
      background: #e8f5ed;
    }

    // Special styles for the delete item to override the default hover effects
    &.t-dropdown__item--theme-error {
      color: #f56c6c;

      &:hover {
        background: #fef0f0;
        color: #e53e3e !important; // Use !important to ensure the default styles are overridden
      }

      &:active {
        background: #fde2e2;
        color: #c53030;
      }
    }
  }
}
</style>
