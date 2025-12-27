<template>
  <div class="agent-settings">
    <div v-if="activeSection === 'modes'">
      <div class="section-header">
        <h2>{{ $t('settings.conversationStrategy') }}</h2>
        <p class="section-description">{{ $t('conversationSettings.description') }}</p>
      </div>

      <t-tabs v-model="activeTab" class="conversation-tabs">
      <!-- Agent Mode Settings Tab -->
      <t-tab-panel value="agent" :label="$t('conversationSettings.agentMode')">
        <div class="tab-content">
          <!-- Agent Status Display -->
          <div class="agent-status-row">
        <div class="status-label">
          <label>{{ $t('agentSettings.status.label') }}</label>
        </div>
        <div class="status-control">
          <div class="status-badge" :class="{ ready: isAgentReady }">
            <t-icon 
              v-if="isAgentReady" 
              name="check-circle-filled" 
              class="status-icon"
            />
            <t-icon 
              v-else 
              name="error-circle-filled" 
              class="status-icon"
            />
            <span class="status-text">
              {{ isAgentReady ? $t('agentSettings.status.ready') : $t('agentSettings.status.notReady') }}
            </span>
          </div>
          <span v-if="!isAgentReady" class="status-hint">
            {{ agentStatusMessage }}
            <t-link v-if="needsModelConfig" @click="handleGoToModelSettings" theme="primary">
              {{ $t('agentSettings.status.goConfigureModels') }}
            </t-link>
          </span>
          <p v-if="!isAgentReady" class="status-tip">
            <t-icon name="info-circle" class="tip-icon" />
            {{ $t('agentSettings.status.hint') }}
          </p>
        </div>
      </div>

          <!-- Model Recommendation Hint -->
          <div class="model-recommendation-box">
            <div class="recommendation-header">
              <t-icon name="info-circle" class="recommendation-icon" />
              <span class="recommendation-title">{{ $t('agentSettings.modelRecommendation.title') }}</span>
            </div>
            <div class="recommendation-content">
              <p>{{ $t('agentSettings.modelRecommendation.content') }}</p>
            </div>
          </div>

          <div class="settings-group">

      <!-- Max Iterations -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('agentSettings.maxIterations.label') }}</label>
          <p class="desc">{{ $t('agentSettings.maxIterations.desc') }}</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
          <t-slider 
            v-model="localMaxIterations" 
            :min="1" 
            :max="30" 
            :step="1"
            :marks="{ 1: '1', 5: '5', 10: '10', 15: '15', 20: '20', 25: '25', 30: '30' }"
            @change="handleMaxIterationsChangeDebounced"
              style="width: 200px;"
          />
            <span class="value-display">{{ localMaxIterations }}</span>
          </div>
        </div>
      </div>

      <!-- Temperature Parameter -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('agentSettings.temperature.label') }}</label>
          <p class="desc">{{ $t('agentSettings.temperature.desc') }}</p>
        </div>
        <div class="setting-control">
          <div class="slider-with-value">
          <t-slider 
            v-model="localTemperature" 
            :min="0" 
            :max="1" 
            :step="0.1"
            :marks="{ 0: '0', 0.5: '0.5', 1: '1' }"
            @change="handleTemperatureChange"
              style="width: 200px;"
          />
            <span class="value-display">{{ localTemperature.toFixed(1) }}</span>
          </div>
        </div>
      </div>

      <!-- Allowed Tools -->
      <div class="setting-row vertical">
        <div class="setting-info">
          <label>{{ $t('agentSettings.allowedTools.label') }}</label>
          <p class="desc">{{ $t('agentSettings.allowedTools.desc') }}</p>
        </div>
        <div class="setting-control full-width allowed-tools-display">
          <div v-if="displayAllowedTools.length" class="allowed-tool-list">
            <div
              v-for="tool in displayAllowedTools"
              :key="tool.name"
              class="allowed-tool-chip"
            >
              <span class="allowed-tool-label">{{ tool.label }}</span>
              <span
                v-if="tool.description"
                class="allowed-tool-desc"
              >
                {{ tool.description }}
              </span>
            </div>
          </div>
          <p v-else class="allowed-tools-empty">
            {{ $t('agentSettings.allowedTools.empty') }}
          </p>
        </div>
      </div>

      <!-- System Prompt -->
      <div class="setting-row vertical">
        <div class="setting-info">
          <label>{{ $t('agentSettings.systemPrompt.label') }}</label>
          <p class="desc">{{ $t('agentSettings.systemPrompt.desc') }}</p>
          <div class="placeholder-hint">
            <p class="hint-title">{{ $t('agentSettings.systemPrompt.availablePlaceholders') }}</p>
            <ul class="placeholder-list">
              <li v-for="placeholder in availablePlaceholders" :key="placeholder.name">
                <code v-html="`{{${placeholder.name}}}`"></code> - {{ placeholder.label }}（{{ placeholder.description }}）
              </li>
            </ul>
            <p class="hint-tip">{{ $t('agentSettings.systemPrompt.hintPrefix') }} <code>&#123;&#123;</code> {{ $t('agentSettings.systemPrompt.hintSuffix') }}</p>
          </div>
        </div>
        <div class="setting-control full-width" style="position: relative;">
          <div class="prompt-header">
            <div class="prompt-toggle">
              <span class="prompt-toggle-label">{{ $t('agentSettings.systemPrompt.custom') }}</span>
              <t-switch
                v-model="localUseCustomSystemPrompt"
                :label="[$t('common.off'), $t('common.on')]"
                size="large"
                @change="handleUseCustomPromptToggle"
              />
            </div>
            <t-button
              v-if="localUseCustomSystemPrompt"
              theme="default"
              variant="outline"
              size="small"
              @click="handleResetToDefault"
              :loading="isResettingPrompt"
            >
              {{ $t('common.resetToDefault') }}
            </t-button>
          </div>
          <p class="prompt-tab-hint">
            {{ $t('agentSettings.systemPrompt.tabHint') }}
          </p>
          <p v-if="!localUseCustomSystemPrompt" class="prompt-disabled-hint">
            {{ $t('agentSettings.systemPrompt.disabledHint') }}
          </p>
          <div v-if="localUseCustomSystemPrompt" class="system-prompt-tabs">
            <t-tabs
              v-model="activeSystemPromptTab"
              class="system-prompt-variant-tabs"
            >
              <t-tab-panel value="web-enabled" :label="$t('agentSettings.systemPrompt.tabWebOn')">
                <div v-if="activeSystemPromptTab === 'web-enabled'" class="prompt-textarea-wrapper">
                  <t-textarea
                    ref="promptTextareaRef"
                    v-model="localSystemPromptWebEnabled"
                    :autosize="{ minRows: 15, maxRows: 30 }"
                    :placeholder="$t('agentSettings.systemPrompt.placeholder')"
                    @blur="handleSystemPromptChange('web-enabled', $event)"
                    @input="handlePromptInput"
                    @keydown="handlePromptKeydown"
                    :readonly="!localUseCustomSystemPrompt"
                    :class="{ 'prompt-textarea-readonly': !localUseCustomSystemPrompt }"
                    style="width: 100%; font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 13px;"
                  />
                </div>
              </t-tab-panel>
              <t-tab-panel value="web-disabled" :label="$t('agentSettings.systemPrompt.tabWebOff')">
                <div v-if="activeSystemPromptTab === 'web-disabled'" class="prompt-textarea-wrapper">
                  <t-textarea
                    ref="promptTextareaRef"
                    v-model="localSystemPromptWebDisabled"
                    :autosize="{ minRows: 15, maxRows: 30 }"
                    :placeholder="$t('agentSettings.systemPrompt.placeholder')"
                    @blur="handleSystemPromptChange('web-disabled', $event)"
                    @input="handlePromptInput"
                    @keydown="handlePromptKeydown"
                    :readonly="!localUseCustomSystemPrompt"
                    :class="{ 'prompt-textarea-readonly': !localUseCustomSystemPrompt }"
                    style="width: 100%; font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 13px;"
                  />
                </div>
              </t-tab-panel>
            </t-tabs>
          </div>
          <!-- Placeholder Prompt Popup -->
          <teleport to="body">
            <div
              v-if="localUseCustomSystemPrompt && showPlaceholderPopup && filteredPlaceholders.length > 0"
              class="placeholder-popup-wrapper"
              :style="popupStyle"
            >
              <div class="placeholder-popup">
              <div
                v-for="(placeholder, index) in filteredPlaceholders"
                :key="placeholder.name"
                class="placeholder-item"
                :class="{ active: selectedPlaceholderIndex === index }"
                @mousedown.prevent="insertPlaceholder(placeholder.name)"
                @mouseenter="selectedPlaceholderIndex = index"
              >
                  <div class="placeholder-name">
                    <code v-html="`{{${placeholder.name}}}`"></code>
                  </div>
                  <div class="placeholder-desc">{{ placeholder.description }}</div>
                </div>
              </div>
            </div>
          </teleport>
        </div>
      </div>
        </div>
      </div>
      </t-tab-panel>

      <!-- Normal Mode Settings Tab -->
      <t-tab-panel value="normal" :label="$t('conversationSettings.normalMode')">
        <div class="tab-content">
          <div class="settings-group">
            <!-- System Prompt (Normal Mode, Custom Switch) -->
            <div class="setting-row vertical">
              <div class="setting-info">
                <label>{{ $t('conversationSettings.systemPrompt.label') }}</label>
                <p class="desc">{{ $t('conversationSettings.systemPrompt.desc') }}</p>
              </div>
              <div class="setting-control full-width">
                <div class="prompt-header">
                  <div class="prompt-toggle">
                    <span class="prompt-toggle-label">{{ $t('conversationSettings.systemPrompt.custom') }}</span>
                    <t-switch
                      v-model="localUseCustomSystemPromptNormal"
                      :label="[$t('common.off'), $t('common.on')]"
                      size="large"
                      @change="handleUseCustomSystemPromptNormalToggle"
                    />
                  </div>
                </div>
                <p v-if="!localUseCustomSystemPromptNormal" class="prompt-disabled-hint">
                  {{ $t('conversationSettings.systemPrompt.disabledHint') }}
                </p>
                <div v-if="localUseCustomSystemPromptNormal" class="prompt-textarea-wrapper">
                  <t-textarea
                    v-model="localSystemPromptNormal"
                    :autosize="{ minRows: 10, maxRows: 20 }"
                    :placeholder="$t('conversationSettings.systemPrompt.placeholder')"
                    @blur="handleSystemPromptNormalChange"
                    style="width: 100%; font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 13px;"
                  />
                </div>
              </div>
            </div>

            <!-- Context Template (Normal Mode, Custom Switch) -->
            <div class="setting-row vertical">
              <div class="setting-info">
                <label>{{ $t('conversationSettings.contextTemplate.label') }}</label>
                <p class="desc">{{ $t('conversationSettings.contextTemplate.desc') }}</p>
              </div>
              <div class="setting-control full-width">
                <div class="prompt-header">
                  <div class="prompt-toggle">
                    <span class="prompt-toggle-label">{{ $t('conversationSettings.contextTemplate.custom') }}</span>
                    <t-switch
                      v-model="localUseCustomContextTemplate"
                      :label="[$t('common.off'), $t('common.on')]"
                      size="large"
                      @change="handleUseCustomContextTemplateToggle"
                    />
                  </div>
                </div>
                <p v-if="!localUseCustomContextTemplate" class="prompt-disabled-hint">
                  {{ $t('conversationSettings.contextTemplate.disabledHint') }}
                </p>
                <div v-if="localUseCustomContextTemplate" class="prompt-textarea-wrapper">
                  <t-textarea
                    v-model="localContextTemplate"
                    :autosize="{ minRows: 15, maxRows: 30 }"
                    :placeholder="$t('conversationSettings.contextTemplate.placeholder')"
                    @blur="handleContextTemplateChange"
                    style="width: 100%; font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 13px;"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </t-tab-panel>
    </t-tabs>
    </div>

    <div v-else-if="activeSection === 'models'" class="section-block" data-conversation-section="models">
      <div class="section-header">
        <h2>{{ $t('conversationSettings.menus.models') }}</h2>
        <p class="section-description">{{ $t('conversationSettings.models.description') }}</p>
      </div>

      <div class="settings-group">
        <!-- Default LLM (Chat/Summary Model) -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.models.chatGroupLabel') }}</label>
            <p class="desc">{{ $t('conversationSettings.models.chatGroupDesc') }}</p>
          </div>
          <div class="setting-control">
            <t-select
              v-model="localSummaryModelId"
              :loading="loadingModels"
              filterable
              :placeholder="$t('conversationSettings.models.chatModel.placeholder')"
              style="width: 320px;"
              @focus="loadAllModels"
              @change="handleConversationSummaryModelChange"
            >
              <t-option
                v-for="model in chatModels"
                :key="model.id"
                :value="model.id"
                :label="model.name"
              />
              <t-option value="__add_model__" class="add-model-option">
                <div class="model-option add">
                  <t-icon name="add" class="add-icon" />
                  <span class="model-name">{{ $t('agentSettings.model.addChat') }}</span>
                </div>
              </t-option>
            </t-select>
          </div>
        </div>

        <!-- Default ReRank Model -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.models.rerankGroupLabel') }}</label>
            <p class="desc">{{ $t('conversationSettings.models.rerankGroupDesc') }}</p>
          </div>
          <div class="setting-control">
            <t-select
              v-model="localConversationRerankModelId"
              :loading="loadingModels"
              filterable
              :placeholder="$t('conversationSettings.models.rerankModel.placeholder')"
              style="width: 320px;"
              @focus="loadAllModels"
              @change="handleConversationRerankModelChange"
            >
              <t-option
                v-for="model in rerankModels"
                :key="model.id"
                :value="model.id"
                :label="model.name"
              />
              <t-option value="__add_model__" class="add-model-option">
                <div class="model-option add">
                  <t-icon name="add" class="add-icon" />
                  <span class="model-name">{{ $t('agentSettings.model.addRerank') }}</span>
                </div>
              </t-option>
            </t-select>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="activeSection === 'thresholds'" class="section-block">
      <div class="section-header">
        <h2>{{ $t('conversationSettings.menus.thresholds') }}</h2>
        <p class="section-description">{{ $t('conversationSettings.thresholds.description') }}</p>
      </div>

      <div class="settings-group">
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.maxRounds.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.maxRounds.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localMaxRounds"
              :min="1"
              :max="50"
              @change="handleMaxRoundsChange"
            />
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.embeddingTopK.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.embeddingTopK.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localEmbeddingTopK"
              :min="1"
              :max="50"
              @change="handleEmbeddingTopKChange"
            />
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.keywordThreshold.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.keywordThreshold.desc') }}</p>
          </div>
          <div class="setting-control slider-with-value">
            <t-slider
              v-model="localKeywordThreshold"
              :min="0"
              :max="1"
              :step="0.05"
              style="width: 240px;"
              @change="handleKeywordThresholdChange"
            />
            <span class="value-display">{{ localKeywordThreshold.toFixed(2) }}</span>
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.vectorThreshold.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.vectorThreshold.desc') }}</p>
          </div>
          <div class="setting-control slider-with-value">
            <t-slider
              v-model="localVectorThreshold"
              :min="0"
              :max="1"
              :step="0.05"
              style="width: 240px;"
              @change="handleVectorThresholdChange"
            />
            <span class="value-display">{{ localVectorThreshold.toFixed(2) }}</span>
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.rerankTopK.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.rerankTopK.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localRerankTopK"
              :min="1"
              :max="20"
              @change="handleRerankTopKChange"
            />
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.rerankThreshold.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.rerankThreshold.desc') }}</p>
          </div>
          <div class="setting-control slider-with-value">
            <t-slider
              v-model="localRerankThreshold"
              :min="0"
              :max="1"
              :step="0.05"
              style="width: 240px;"
              @change="handleRerankThresholdChange"
            />
            <span class="value-display">{{ localRerankThreshold.toFixed(2) }}</span>
          </div>
        </div>

      </div>
    </div>

    <div v-else-if="activeSection === 'advanced'" class="section-block">
      <div class="section-header">
        <h2>{{ $t('conversationSettings.menus.advanced') }}</h2>
        <p class="section-description">{{ $t('conversationSettings.advanced.description') }}</p>
      </div>

      <div class="settings-group">
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.enableQueryExpansion.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.enableQueryExpansion.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-switch
              v-model="localEnableQueryExpansion"
              :label="[$t('common.off'), $t('common.on')]"
              @change="handleEnableQueryExpansionChange"
            />
          </div>
        </div>
        <!-- Enable Query Rewrite -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.enableRewrite.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.enableRewrite.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-switch
              v-model="localEnableRewrite"
              :label="[$t('common.off'), $t('common.on')]"
              @change="handleEnableRewriteChange"
            />
          </div>
        </div>

        <!-- Rewrite Prompt: Only displayed when rewrite is enabled -->
        <div v-if="localEnableRewrite" class="setting-row vertical">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.rewritePrompt.system') }}</label>
            <p class="desc">{{ $t('conversationSettings.rewritePrompt.desc') }}</p>
          </div>
          <div class="setting-control full-width">
            <t-textarea
              v-model="localRewritePromptSystem"
              :autosize="{ minRows: 8, maxRows: 16 }"
              @blur="handleRewritePromptSystemChange"
            />
          </div>
        </div>

        <div v-if="localEnableRewrite" class="setting-row vertical">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.rewritePrompt.user') }}</label>
            <p class="desc">{{ $t('conversationSettings.rewritePrompt.userDesc') }}</p>
          </div>
          <div class="setting-control full-width">
            <t-textarea
              v-model="localRewritePromptUser"
              :autosize="{ minRows: 8, maxRows: 16 }"
              @blur="handleRewritePromptUserChange"
            />
          </div>
        </div>

        <!-- Fallback Strategy -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.fallbackStrategy.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.fallbackStrategy.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-radio-group v-model="localFallbackStrategy" @change="handleFallbackStrategyChange">
              <t-radio value="fixed">{{ $t('conversationSettings.fallbackStrategy.fixed') }}</t-radio>
              <t-radio value="model">{{ $t('conversationSettings.fallbackStrategy.model') }}</t-radio>
            </t-radio-group>
          </div>
        </div>

        <!-- Fixed Fallback Response: Only displayed when fixed response is selected -->
        <div v-if="localFallbackStrategy === 'fixed'" class="setting-row vertical">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.fallbackResponse.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.fallbackResponse.desc') }}</p>
          </div>
          <div class="setting-control full-width">
            <t-textarea
              v-model="localFallbackResponse"
              :autosize="{ minRows: 3, maxRows: 6 }"
              @blur="handleFallbackResponseChange"
            />
          </div>
        </div>

        <!-- Fallback Prompt: Only displayed when "Leave it to the model to continue generating" is selected -->
        <div v-else-if="localFallbackStrategy === 'model'" class="setting-row vertical">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.fallbackPrompt.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.fallbackPrompt.desc') }}</p>
          </div>
          <div class="setting-control full-width">
            <t-textarea
              v-model="localFallbackPrompt"
              :autosize="{ minRows: 8, maxRows: 16 }"
              @blur="handleFallbackPromptChange"
            />
          </div>
        </div>

        <!-- Normal Mode Generation Params: Temperature -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.temperature.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.temperature.desc') }}</p>
          </div>
          <div class="setting-control">
            <div class="slider-with-value">
              <t-slider 
                v-model="localTemperatureNormal" 
                :min="0" 
                :max="1" 
                :step="0.1"
                :marks="{ 0: '0', 0.5: '0.5', 1: '1' }"
                @change="handleTemperatureNormalChange"
                style="width: 200px;"
              />
              <span class="value-display">{{ localTemperatureNormal.toFixed(1) }}</span>
            </div>
          </div>
        </div>

        <!-- Normal Mode Generation Params: Max Tokens -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('conversationSettings.maxTokens.label') }}</label>
            <p class="desc">{{ $t('conversationSettings.maxTokens.desc') }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localMaxCompletionTokens"
              :min="1"
              :max="100000"
              :step="100"
              @change="handleMaxCompletionTokensChange"
              style="width: 200px;"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed, nextTick } from 'vue'
import type { Ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { listModels, type ModelConfig } from '@/api/model'
import { getAgentConfig, updateAgentConfig, getConversationConfig, updateConversationConfig, type AgentConfig, type ConversationConfig, type ToolDefinition, type PlaceholderDefinition } from '@/api/system'

const props = defineProps<{
  // 来自外部设置弹窗的子菜单 key: 'modes' | 'models' | 'thresholds' | 'advanced'
  activeSubSection?: string
}>()

// Current sub-page (modes, models, thresholds, advanced)
const activeSection = computed(() => props.activeSubSection || 'modes')

const settingsStore = useSettingsStore()
const router = useRouter()
const { t } = useI18n()

// Tab state
const activeTab = ref('agent')

const getDefaultConversationConfig = (): ConversationConfig => ({
  prompt: '',
  context_template: '',
  temperature: 0.3,
  max_completion_tokens: 2048,
  use_custom_system_prompt: true,
  use_custom_context_template: true,
  max_rounds: 5,
  embedding_top_k: 10,
  keyword_threshold: 0.3,
  vector_threshold: 0.5,
  rerank_top_k: 5,
  rerank_threshold: 0.5,
  enable_rewrite: true,
  enable_query_expansion: true,
  fallback_strategy: 'fixed',
  fallback_response: '',
  fallback_prompt: '',
  summary_model_id: '',
  rerank_model_id: '',
  rewrite_prompt_system: '',
  rewrite_prompt_user: '',
})

const normalizeConversationConfig = (config?: Partial<ConversationConfig>): ConversationConfig => ({
  ...getDefaultConversationConfig(),
  ...config,
})

const conversationConfig = ref<ConversationConfig>(getDefaultConversationConfig())
const conversationConfigLoaded = ref(false)
const conversationSaving = ref(false)

// Agent mode local state
const localMaxIterations = ref(5)
const localTemperature = ref(0.7)
const localAllowedTools = ref<string[]>([])

type SystemPromptTab = 'web-enabled' | 'web-disabled'
const activeSystemPromptTab = ref<SystemPromptTab>('web-enabled')
const localSystemPromptWebEnabled = ref('')
const localSystemPromptWebDisabled = ref('')
const systemPromptRefs: Record<SystemPromptTab, Ref<string>> = {
  'web-enabled': localSystemPromptWebEnabled,
  'web-disabled': localSystemPromptWebDisabled,
}
const savedSystemPromptMap: Record<SystemPromptTab, string> = {
  'web-enabled': '',
  'web-disabled': '',
}
const getPromptRefByTab = (tab: SystemPromptTab) => systemPromptRefs[tab]
const getActivePromptRef = () => getPromptRefByTab(activeSystemPromptTab.value)
const localUseCustomSystemPrompt = ref(false)

// Normal mode local state
const localContextTemplate = ref('')
const localSystemPromptNormal = ref('')
const localTemperatureNormal = ref(0.3)
const localMaxCompletionTokens = ref(2048)
let savedContextTemplate = ''
let savedSystemPromptNormal = ''
let savedTemperatureNormal = 0.3
let savedMaxCompletionTokens = 2048

const localMaxRounds = ref(5)
const localEmbeddingTopK = ref(10)
const localKeywordThreshold = ref(0.3)
const localVectorThreshold = ref(0.5)
const localRerankTopK = ref(5)
const localRerankThreshold = ref(0.5)
const localEnableRewrite = ref(true)
const localEnableQueryExpansion = ref(true)
const localFallbackStrategy = ref<'fixed' | 'model'>('fixed')
const localFallbackResponse = ref('')
const localFallbackPrompt = ref('')
const localRewritePromptSystem = ref('')
const localRewritePromptUser = ref('')
const localSummaryModelId = ref('')
const localConversationRerankModelId = ref('')

// Whether Normal mode Prompt is customized (uses system default when off)
const localUseCustomSystemPromptNormal = ref(true)
const localUseCustomContextTemplate = ref(true)

const syncConversationLocals = () => {
  const cfg = conversationConfig.value
  localContextTemplate.value = cfg.context_template ?? ''
  savedContextTemplate = localContextTemplate.value
  localSystemPromptNormal.value = cfg.prompt ?? ''
  savedSystemPromptNormal = localSystemPromptNormal.value
  localTemperatureNormal.value = cfg.temperature ?? 0.3
  savedTemperatureNormal = localTemperatureNormal.value
  localMaxCompletionTokens.value = cfg.max_completion_tokens ?? 2048
  savedMaxCompletionTokens = localMaxCompletionTokens.value

  localMaxRounds.value = cfg.max_rounds ?? 5
  localEmbeddingTopK.value = cfg.embedding_top_k ?? 10
  localKeywordThreshold.value = cfg.keyword_threshold ?? 0.3
  localVectorThreshold.value = cfg.vector_threshold ?? 0.5
  localRerankTopK.value = cfg.rerank_top_k ?? 5
  localRerankThreshold.value = cfg.rerank_threshold ?? 0.5
  localEnableRewrite.value = cfg.enable_rewrite ?? true
  localEnableQueryExpansion.value = cfg.enable_query_expansion ?? true
  localFallbackStrategy.value = (cfg.fallback_strategy as 'fixed' | 'model') || 'fixed'
  localFallbackResponse.value = cfg.fallback_response ?? ''
  localFallbackPrompt.value = cfg.fallback_prompt ?? ''
  localRewritePromptSystem.value = cfg.rewrite_prompt_system ?? ''
  localRewritePromptUser.value = cfg.rewrite_prompt_user ?? ''
  localSummaryModelId.value = cfg.summary_model_id ?? ''
  localConversationRerankModelId.value = cfg.rerank_model_id ?? ''
  localUseCustomSystemPromptNormal.value = cfg.use_custom_system_prompt ?? true
  localUseCustomContextTemplate.value = cfg.use_custom_context_template ?? true

  settingsStore.updateConversationModels({
    summaryModelId: localSummaryModelId.value || '',
    rerankModelId: localConversationRerankModelId.value || '',
  })
}

const saveConversationConfig = async (partial: Partial<ConversationConfig>, toastMessage?: string) => {
  if (!conversationConfigLoaded.value) return

  const payload = normalizeConversationConfig({
    ...conversationConfig.value,
    ...partial,
  })

  try {
    conversationSaving.value = true
    const res = await updateConversationConfig(payload)
    conversationConfig.value = normalizeConversationConfig(res.data ?? payload)
    syncConversationLocals()
    if (toastMessage) {
      MessagePlugin.success(toastMessage)
    }
  } catch (error) {
    console.error('Failed to save conversation config:', error)
    MessagePlugin.error(getErrorMessage(error))
    throw error
  } finally {
    conversationSaving.value = false
  }
}

// Check if Agent is ready
const isAgentReady = computed(() => {
  return (
    localAllowedTools.value.length > 0 &&
    localSummaryModelId.value &&
    localSummaryModelId.value.trim() !== '' &&
    localConversationRerankModelId.value &&
    localConversationRerankModelId.value.trim() !== ''
  )
})

const buildAgentConfigPayload = (overrides: Partial<AgentConfig> = {}): AgentConfig => ({
  max_iterations: localMaxIterations.value,
  reflection_enabled: false,
  allowed_tools: localAllowedTools.value,
  temperature: localTemperature.value,
  system_prompt_web_enabled: localSystemPromptWebEnabled.value,
  system_prompt_web_disabled: localSystemPromptWebDisabled.value,
  use_custom_system_prompt: localUseCustomSystemPrompt.value,
  ...overrides,
})

// Check if model configuration is missing
const needsModelConfig = computed(() => {
  return (
    (!localSummaryModelId.value || localSummaryModelId.value.trim() === '') ||
    (!localConversationRerankModelId.value || localConversationRerankModelId.value.trim() === '')
  )
})

// Agent status hint message
const agentStatusMessage = computed(() => {
  const missing: string[] = []
  
  if (localAllowedTools.value.length === 0) {
    missing.push(t('agentSettings.status.missingAllowedTools'))
  }
  
  if (!localSummaryModelId.value || localSummaryModelId.value.trim() === '') {
    missing.push(t('agentSettings.status.missingSummaryModel'))
  }
  
  if (!localConversationRerankModelId.value || localConversationRerankModelId.value.trim() === '') {
    missing.push(t('agentSettings.status.missingRerankModel'))
  }
  
  if (missing.length === 0) {
    return ''
  }
  
  return t('agentSettings.status.pleaseConfigure', { items: missing.join('、') })
})

// Redirect to model settings
const handleGoToModelSettings = () => {
  router.push('/platform/settings')

  setTimeout(() => {
    const event = new CustomEvent('settings-nav', {
      detail: { section: 'agent', subsection: 'models' }
    })
    window.dispatchEvent(event)

    setTimeout(() => {
      const sectionEl = document.querySelector('[data-conversation-section="models"]')
      if (sectionEl) {
        sectionEl.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    }, 150)
  }, 100)
}

// Model list state
const chatModels = ref<ModelConfig[]>([])
const rerankModels = ref<ModelConfig[]>([])
const loadingModels = ref(false)

// Available tools list
const availableTools = ref<ToolDefinition[]>([])
// Available placeholders list
const availablePlaceholders = ref<PlaceholderDefinition[]>([])
const displayAllowedTools = computed(() => {
  return localAllowedTools.value.map(name => {
    const detail = availableTools.value.find(tool => tool.name === name)
    return {
      name,
      label: detail?.label || name,
      description: detail?.description || ''
    }
  })
})

// Config loading state
const loadingConfig = ref(false)
const configLoaded = ref(false) // Prevent duplicate loading
const isInitializing = ref(true) // Mark if initializing, to prevent triggering save during initialization

// Saved Prompt value, used to compare for changes
let savedUseCustomSystemPrompt = false

// Loading state for resetting to default Prompt
const isResettingPrompt = ref(false)

// Placeholder prompt related state
const promptTextareaRef = ref<any>(null)
const showPlaceholderPopup = ref(false)
const selectedPlaceholderIndex = ref(0)
let placeholderPopupTimer: any = null
const placeholderPrefix = ref('') // Current input prefix, used for filtering
const popupStyle = ref({ top: '0px', left: '0px' }) // Prompt box position

watch(activeSystemPromptTab, () => {
  showPlaceholderPopup.value = false
  placeholderPrefix.value = ''
  selectedPlaceholderIndex.value = 0
})

// Set up textarea native event listeners
const setupTextareaEventListeners = () => {
  nextTick(() => {
    const textarea = getTextareaElement()
    if (textarea) {
      // Add native keydown event listener (use capture phase to ensure priority processing)
      textarea.addEventListener('keydown', (e: KeyboardEvent) => {
        // If placeholder prompt is showing, prioritize placeholder-related keys
        if (showPlaceholderPopup.value && filteredPlaceholders.value.length > 0) {
          if (e.key === 'ArrowDown') {
            // ArrowDown selects the next item
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            if (selectedPlaceholderIndex.value < filteredPlaceholders.value.length - 1) {
              selectedPlaceholderIndex.value++
            } else {
              selectedPlaceholderIndex.value = 0 // Loop to the first one
            }
            return
          } else if (e.key === 'ArrowUp') {
            // ArrowUp selects the previous item
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            if (selectedPlaceholderIndex.value > 0) {
              selectedPlaceholderIndex.value--
            } else {
              selectedPlaceholderIndex.value = filteredPlaceholders.value.length - 1 // Loop to the last one
            }
            return
          } else if (e.key === 'Enter') {
            // Enter key inserts the selected placeholder
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            const selected = filteredPlaceholders.value[selectedPlaceholderIndex.value]
            if (selected) {
              insertPlaceholder(selected.name)
            }
            return
          } else if (e.key === 'Escape') {
            // ESC key closes the prompt
            e.preventDefault()
            e.stopPropagation()
            e.stopImmediatePropagation()
            showPlaceholderPopup.value = false
            placeholderPrefix.value = ''
            return
          }
        }
        
        // If the pressed key is {
        if (e.key === '{') {
          // Clear previous timer
          if (placeholderPopupTimer) {
            clearTimeout(placeholderPopupTimer)
          }
          
          // Delayed check, wait for input to complete (consecutive input of two {)
          placeholderPopupTimer = setTimeout(() => {
            checkAndShowPlaceholderPopup()
          }, 150)
        }
      }, true) // 使用 capture 阶段
      
      // Add native input event listener (as fallback)
      textarea.addEventListener('input', () => {
        if (placeholderPopupTimer) {
          clearTimeout(placeholderPopupTimer)
        }
        placeholderPopupTimer = setTimeout(() => {
          checkAndShowPlaceholderPopup()
        }, 50)
      })
    }
  })
}

// Helper function to get textarea element
const getTextareaElement = (): HTMLTextAreaElement | null => {
  if (promptTextareaRef.value) {
    if (promptTextareaRef.value.$el) {
      return promptTextareaRef.value.$el.querySelector('textarea')
    } else if (promptTextareaRef.value instanceof HTMLTextAreaElement) {
      return promptTextareaRef.value
    }
  }
  
  // If still not found, try finding via DOM
  const wrapper = document.querySelector('.setting-control.full-width')
  return wrapper?.querySelector('textarea') || null
}

// Initial load
onMounted(async () => {
  // Prevent duplicate loading
  if (configLoaded.value) return
  
  loadingConfig.value = true
  configLoaded.value = true
  isInitializing.value = true
  
  try {
    // Load config from backend
    const res = await getAgentConfig()
    const config = res.data
    
    // Update local state (save won't be triggered during initialization)
    localMaxIterations.value = config.max_iterations
    lastSavedValue = config.max_iterations // Record initial saved value
    localTemperature.value = config.temperature
    localAllowedTools.value = config.allowed_tools || []
    const promptWebEnabled = config.system_prompt_web_enabled || ''
    const promptWebDisabled = config.system_prompt_web_disabled || ''
    localSystemPromptWebEnabled.value = promptWebEnabled
    localSystemPromptWebDisabled.value = promptWebDisabled
    savedSystemPromptMap['web-enabled'] = promptWebEnabled
    savedSystemPromptMap['web-disabled'] = promptWebDisabled
    const useCustomPrompt = config.use_custom_system_prompt ?? false
    localUseCustomSystemPrompt.value = useCustomPrompt
    savedUseCustomSystemPrompt = useCustomPrompt
    availableTools.value = config.available_tools || []
    availablePlaceholders.value = config.available_placeholders || []
    
    // Debug info
    console.log('加载的占位符列表:', availablePlaceholders.value)
    
    // Batch load all models (single API call)
      await loadAllModels()
    
    // Sync to store (updates local storage only, no API save)
    // Note: isAgentEnabled is not set automatically, keep user choice
    // Enabled status should be manually controlled by the user
    settingsStore.updateAgentConfig({
      maxIterations: config.max_iterations,
      temperature: config.temperature,
      allowedTools: config.allowed_tools || [],
      system_prompt_web_enabled: promptWebEnabled,
      system_prompt_web_disabled: promptWebDisabled,
      use_custom_system_prompt: useCustomPrompt
    })

    // Load Normal mode config
    if (!conversationConfigLoaded.value) {
      try {
        const convRes = await getConversationConfig()
        conversationConfig.value = normalizeConversationConfig(convRes.data)
        conversationConfigLoaded.value = true
        syncConversationLocals()
      } catch (error) {
        console.error('加载普通模式配置失败:', error)
        // Use default values
        conversationConfigLoaded.value = true
      }
    }
    
    // Wait for next tick to ensure all reactive updates are complete
    await nextTick()
    // Wait one more frame to ensure event listeners are set up
    requestAnimationFrame(() => {
      // Initialization complete, saving is now allowed
      isInitializing.value = false
      
      // Set up native event listeners (as fallback)
      setupTextareaEventListeners()
    })
  } catch (error) {
    console.error('Failed to load Agent config:', error)
    MessagePlugin.error('Failed to load Agent config')
    configLoaded.value = false // Reset flag on failure to allow retry
    
    // Load from store on failure
    localMaxIterations.value = settingsStore.agentConfig.maxIterations
    localTemperature.value = settingsStore.agentConfig.temperature
  } finally {
    loadingConfig.value = false
    isInitializing.value = false    // Ensure initialization is marked complete even on failure
  }
})

// Map error codes to error messages
const getErrorMessage = (error: any): string => {
  const errorCode = error?.response?.data?.error?.code
  const errorMessage = error?.response?.data?.error?.message
  
  switch (errorCode) {
    case 2100:
      return t('agentSettings.errors.selectThinkingModel')
    case 2101:
      return t('agentSettings.errors.selectAtLeastOneTool')
    case 2102:
      return t('agentSettings.errors.iterationsRange')
    case 2103:
      return t('agentSettings.errors.temperatureRange')
    case 1010:
      return errorMessage || t('agentSettings.errors.validationFailed')
    default:
      return errorMessage || t('common.saveFailed')
  }
}

// Debounce timer
let maxIterationsDebounceTimer: any = null
// Last saved value, to avoid duplicate saves of the same value
let lastSavedValue: number | null = null

// Handle max iterations change (debounced version, for both clicks and drags)
const handleMaxIterationsChangeDebounced = (value: number) => {
  // Don't trigger save if initializing
  if (isInitializing.value) return
  
  // Ensure value is a number
  const numValue = typeof value === 'number' ? value : Number(value)
  if (isNaN(numValue)) {
    console.error('Invalid max_iterations value:', value)
    return
  }
  
  // Don't save if value hasn't changed
  if (lastSavedValue === numValue) {
    return
  }
  
  // Clear previous timer
  if (maxIterationsDebounceTimer) {
    clearTimeout(maxIterationsDebounceTimer)
}

  // Set new timer, save after 300ms (reduced delay for better responsiveness)
  maxIterationsDebounceTimer = setTimeout(async () => {
    // Re-check if value changed (might have changed during wait)
    if (lastSavedValue === numValue) {
      maxIterationsDebounceTimer = null
      return
    }
  
  try {
    const config = buildAgentConfigPayload({ max_iterations: numValue })
    await updateAgentConfig(config)
      settingsStore.updateAgentConfig({ maxIterations: numValue })
      lastSavedValue = numValue // Record saved value
    MessagePlugin.success(t('agentSettings.toasts.iterationsSaved'))
  } catch (error) {
    console.error('Save failed:', error)
    MessagePlugin.error(getErrorMessage(error))
    } finally {
      maxIterationsDebounceTimer = null
  }
  }, 300)
}

// Batch load all models (single API call)
const loadAllModels = async () => {
  if (chatModels.value.length > 0 && rerankModels.value.length > 0) return // Already loaded
  
  loadingModels.value = true
  try {
    const allModels = await listModels()
    // Filter by type to avoid redundant calls
    chatModels.value = allModels.filter(m => m.type === 'KnowledgeQA')
    rerankModels.value = allModels.filter(m => m.type === 'Rerank')
  } catch (error) {
    console.error('Failed to load model list:', error)
    MessagePlugin.error('Failed to load model list')
  } finally {
    loadingModels.value = false
  }
}

// Load chat models (deprecated, use loadAllModels)
const loadChatModels = async () => {
  await loadAllModels()
}

// Load ReRank models (deprecated, use loadAllModels)
const loadRerankModels = async () => {
  await loadAllModels()
}

// Handle temperature change
const handleTemperatureChange = async (value: number) => {
  // Don't trigger save if initializing
  if (isInitializing.value) return
  
  try {
    const config = buildAgentConfigPayload({ temperature: value })
    await updateAgentConfig(config)
    settingsStore.updateAgentConfig({ temperature: value })
    MessagePlugin.success(t('agentSettings.toasts.temperatureSaved'))
  } catch (error) {
    console.error('Save failed:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// Toggle custom Prompt enabled
const handleUseCustomPromptToggle = async (value: boolean) => {
  if (isInitializing.value) return
  if (value === savedUseCustomSystemPrompt) return

  try {
    const config = buildAgentConfigPayload({ use_custom_system_prompt: value })
    await updateAgentConfig(config)
    savedUseCustomSystemPrompt = value

    MessagePlugin.success(value ? t('agentSettings.toasts.customPromptEnabled') : t('agentSettings.toasts.defaultPromptEnabled'))
  } catch (error) {
    console.error('Failed to toggle custom Prompt:', error)
    MessagePlugin.error(getErrorMessage(error))
    // Rollback switch state
    localUseCustomSystemPrompt.value = savedUseCustomSystemPrompt
  }
}

// Handle system Prompt keydown (fallback, main logic in native listener)
const handlePromptKeydown = (e: KeyboardEvent) => {
  // If placeholder prompt is showing, update filter on alphanumeric or underscore input
  if (showPlaceholderPopup.value && /^[a-zA-Z0-9_]$/.test(e.key)) {
    // Delay check, wait for character input to complete
    if (placeholderPopupTimer) {
      clearTimeout(placeholderPopupTimer)
    }
    placeholderPopupTimer = setTimeout(() => {
      checkAndShowPlaceholderPopup()
    }, 50)
  }
}

// Filtered placeholders list (prefix match)
const filteredPlaceholders = computed(() => {
  if (!placeholderPrefix.value) {
    return availablePlaceholders.value
  }
  
  const prefix = placeholderPrefix.value.toLowerCase()
  return availablePlaceholders.value.filter(p => 
    p.name.toLowerCase().startsWith(prefix)
  )
})

// Calculate cursor pixel position in textarea
const calculateCursorPosition = (textarea: HTMLTextAreaElement) => {
  const cursorPos = textarea.selectionStart
  const activePromptValue = getActivePromptRef().value
  const textBeforeCursor = activePromptValue.substring(0, cursorPos)
  
  // Get textarea style and position
  const style = window.getComputedStyle(textarea)
  const textareaRect = textarea.getBoundingClientRect()
  
  // Calculate line count and current line text
  const lines = textBeforeCursor.split('\n')
  const currentLine = lines.length - 1
  const lineText = lines[currentLine] || ''
  
  // Get line height
  const lineHeight = parseFloat(style.lineHeight) || parseFloat(style.fontSize) * 1.2
  
  // Get padding
  const paddingTop = parseFloat(style.paddingTop) || 0
  const paddingLeft = parseFloat(style.paddingLeft) || 0
  
  // Use canvas to measure current line text width (more accurate)
  const canvas = document.createElement('canvas')
  const context = canvas.getContext('2d')
  let textWidth = 0
  
  if (context) {
    context.font = `${style.fontSize} ${style.fontFamily}`
    textWidth = context.measureText(lineText).width
  } else {
    // Fallback: estimate using monospace font width (Monaco/Menlo are monospace)
    const charWidth = parseFloat(style.fontSize) * 0.6 // 等宽字体字符宽度约为字体大小的 0.6 倍
    textWidth = lineText.length * charWidth
  }
  
  // Calculate cursor top position (considering scroll)
  const scrollTop = textarea.scrollTop
  const top = textareaRect.top + paddingTop + (currentLine * lineHeight) - scrollTop + lineHeight + 4
  
  // Calculate cursor left position (considering scroll)
  const scrollLeft = textarea.scrollLeft
  const left = textareaRect.left + paddingLeft + textWidth - scrollLeft
  
  return { top, left }
}

// Check and show placeholder popup
const checkAndShowPlaceholderPopup = () => {
  const textarea = getTextareaElement()
  
  if (!textarea) {
    return
  }
  
  const cursorPos = textarea.selectionStart
  const textBeforeCursor = getActivePromptRef().value.substring(0, cursorPos)
  
  // Check if {{ was input (look back from cursor for nearest {{)
  // Need to find nearest {{ before cursor, with no }} in between
  let lastOpenPos = -1
  for (let i = cursorPos - 1; i >= 0; i--) {
    if (i > 0 && textBeforeCursor[i - 1] === '{' && textBeforeCursor[i] === '{') {
      // Found {{
      const textAfterOpen = textBeforeCursor.substring(i + 1)
      // Check if it already contains }} (placeholder complete)
      if (!textAfterOpen.includes('}}')) {
        lastOpenPos = i - 1
        break
      }
    }
  }
  
  if (lastOpenPos === -1) {
    // No valid {{ found, hide prompt
    showPlaceholderPopup.value = false
    placeholderPrefix.value = ''
    return
  }
  
  // Get content from {{ to cursor position as prefix
  const textAfterOpen = textBeforeCursor.substring(lastOpenPos + 2)
  
  // Update prefix
  placeholderPrefix.value = textAfterOpen
  
  // Filter placeholders by prefix
  const filtered = filteredPlaceholders.value
  
  if (filtered.length > 0) {
    // Match found, show prompt
    // Calculate cursor position
    nextTick(() => {
      const position = calculateCursorPosition(textarea)
      popupStyle.value = {
        top: `${position.top}px`,
        left: `${position.left}px`
      }
      showPlaceholderPopup.value = true
      // Reset selected index to first (default)
      selectedPlaceholderIndex.value = 0
    })
  } else {
    // No match found, hide prompt
    showPlaceholderPopup.value = false
  }
}

// Handle system Prompt input
const handlePromptInput = () => {
  // Clear previous timer
  if (placeholderPopupTimer) {
    clearTimeout(placeholderPopupTimer)
  }
  
  // Delay check to avoid frequent triggers
  placeholderPopupTimer = setTimeout(() => {
    checkAndShowPlaceholderPopup()
  }, 50)
}

// Insert placeholder
const insertPlaceholder = (placeholderName: string) => {
  const textarea = getTextareaElement()
  if (!textarea) {
    return
  }
  
  // Close prompt first to avoid triggering blur event
  showPlaceholderPopup.value = false
  placeholderPrefix.value = ''
  selectedPlaceholderIndex.value = 0
  
  // Delay execution to ensure prompt box is closed
  nextTick(() => {
    const cursorPos = textarea.selectionStart
    const promptRef = getActivePromptRef()
    const currentValue = promptRef.value
    const textBeforeCursor = currentValue.substring(0, cursorPos)
    const textAfterCursor = currentValue.substring(cursorPos)
    
    // Find position of last {{
    const lastOpenPos = textBeforeCursor.lastIndexOf('{{')
    if (lastOpenPos === -1) {
      // If no {{ found, insert complete placeholder directly
      const placeholder = `{{${placeholderName}}}`
      promptRef.value = textBeforeCursor + placeholder + textAfterCursor
      // Set cursor position
      nextTick(() => {
        const newPos = cursorPos + placeholder.length
        textarea.setSelectionRange(newPos, newPos)
        textarea.focus()
      })
    } else {
      // Replace content from {{ to cursor with complete placeholder
      const beforePlaceholder = textBeforeCursor.substring(0, lastOpenPos)
      const placeholder = `{{${placeholderName}}}`
      promptRef.value = beforePlaceholder + placeholder + textAfterCursor
      // Set cursor position
      nextTick(() => {
        const newPos = lastOpenPos + placeholder.length
        textarea.setSelectionRange(newPos, newPos)
        textarea.focus()
      })
    }
  })
}

// Reset to default Prompt
const handleResetToDefault = async () => {
  const confirmDialog = DialogPlugin.confirm({
    header: t('agentSettings.reset.header'),
    body: t('agentSettings.reset.body'),
    confirmBtn: t('common.confirm'),
    cancelBtn: t('common.cancel'),
    onConfirm: async () => {
      try {
        isResettingPrompt.value = true
        
        // Get default values by setting system_prompt_web_* to empty strings
        // Backend returns defaults when fields are empty
        const tempConfig = buildAgentConfigPayload({
          system_prompt_web_enabled: '',
          system_prompt_web_disabled: '',
          use_custom_system_prompt: false,
        })
        
        await updateAgentConfig(tempConfig)
        
        // Reload config to get full content of default Prompt
        const res = await getAgentConfig()
        const defaultPromptWebEnabled = res.data.system_prompt_web_enabled || ''
        const defaultPromptWebDisabled = res.data.system_prompt_web_disabled || ''
        const useCustom = res.data.use_custom_system_prompt ?? false
        
        // Set to default Prompt content
        localSystemPromptWebEnabled.value = defaultPromptWebEnabled
        localSystemPromptWebDisabled.value = defaultPromptWebDisabled
        savedSystemPromptMap['web-enabled'] = defaultPromptWebEnabled
        savedSystemPromptMap['web-disabled'] = defaultPromptWebDisabled
        localUseCustomSystemPrompt.value = useCustom
        savedUseCustomSystemPrompt = useCustom
        
        MessagePlugin.success(t('agentSettings.toasts.resetToDefault'))
        confirmDialog.hide()
      } catch (error) {
        console.error('Failed to reset to default Prompt:', error)
        MessagePlugin.error(getErrorMessage(error))
      } finally {
        isResettingPrompt.value = false
      }
    }
  })
}

// Handle system Prompt change
const handleSystemPromptChange = async (tab: SystemPromptTab, e?: FocusEvent) => {
  // If placeholder prompt box was clicked, don't trigger save
  if (e?.relatedTarget) {
    const target = e.relatedTarget as HTMLElement
    if (target.closest('.placeholder-popup-wrapper')) {
      return
    }
  }
  
  // Delay check to avoid immediate trigger when clicking placeholder
  await nextTick()
  
  // If placeholder prompt box is still showing, it means user clicked placeholder; don't trigger save
  if (showPlaceholderPopup.value) {
    return
  }
  
  // Hide placeholder prompt
  placeholderPrefix.value = ''
  
  // Don't trigger save if initializing
  if (isInitializing.value) return
  
  const promptRef = getPromptRefByTab(tab)
  const savedValue = savedSystemPromptMap[tab]

  // Check if content changed
  if (promptRef.value === savedValue) {
    return // Content unchanged, don't call API
  }
  
  try {
    const config = buildAgentConfigPayload()
    await updateAgentConfig(config)
    savedSystemPromptMap[tab] = promptRef.value // 更新已保存的值
    MessagePlugin.success(t('agentSettings.toasts.systemPromptSaved'))
  } catch (error) {
    console.error('Failed to save system Prompt:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

// Listen to Agent ready status change, sync to store
watch(isAgentReady, (newValue, oldValue) => {
  if (!isInitializing.value) {
    // If config goes from "ready" to "not ready", and Agent is enabled, disable it automatically
    if (!newValue && oldValue && settingsStore.isAgentEnabled) {
      settingsStore.toggleAgent(false)
      MessagePlugin.warning(t('agentSettings.toasts.autoDisabled'))
    }
    // Note: Don't automatically enable when config goes from "not ready" to "ready" (let user decide)
  }
})

// Normal mode config handlers
const handleContextTemplateChange = async () => {
  if (!conversationConfigLoaded.value) return
  
  if (localContextTemplate.value === savedContextTemplate) {
    return
  }
  
  try {
    await saveConversationConfig(
      {
        context_template: localContextTemplate.value,
        use_custom_context_template: localUseCustomContextTemplate.value,
      },
      t('conversationSettings.toasts.contextTemplateSaved')
    )
    savedContextTemplate = localContextTemplate.value
  } catch (error) {
    console.error('Failed to save Context Template:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

const reloadConversationConfig = async () => {
  const convRes = await getConversationConfig()
  conversationConfig.value = normalizeConversationConfig(convRes.data)
  syncConversationLocals()
}

const handleUseCustomSystemPromptNormalToggle = async (value: boolean) => {
  if (!conversationConfigLoaded.value) return

  try {
    if (!value) {
      await saveConversationConfig(
        {
          prompt: '',
          use_custom_system_prompt: false,
        },
        t('conversationSettings.toasts.defaultPromptEnabled')
      )
      await reloadConversationConfig()
    } else {
      await saveConversationConfig(
        {
          prompt: localSystemPromptNormal.value,
          use_custom_system_prompt: true,
        },
        t('conversationSettings.toasts.customPromptEnabled')
      )
      savedSystemPromptNormal = localSystemPromptNormal.value
    }
  } catch (error) {
    console.error('Failed to toggle custom System Prompt in normal mode:', error)
    MessagePlugin.error(getErrorMessage(error))
    localUseCustomSystemPromptNormal.value = !value
  }
}

const handleUseCustomContextTemplateToggle = async (value: boolean) => {
  if (!conversationConfigLoaded.value) return

  try {
    if (!value) {
      await saveConversationConfig(
        {
          context_template: '',
          use_custom_context_template: false,
        },
        t('conversationSettings.toasts.defaultContextTemplateEnabled')
      )
      await reloadConversationConfig()
    } else {
      await saveConversationConfig(
        {
          context_template: localContextTemplate.value,
          use_custom_context_template: true,
        },
        t('conversationSettings.toasts.customContextTemplateEnabled')
      )
      savedContextTemplate = localContextTemplate.value
    }
  } catch (error) {
    console.error('Failed to toggle custom Context Template in normal mode:', error)
    MessagePlugin.error(getErrorMessage(error))
    localUseCustomContextTemplate.value = !value
  }
}

const handleSystemPromptNormalChange = async () => {
  if (!conversationConfigLoaded.value) return
  
  if (localSystemPromptNormal.value === savedSystemPromptNormal) {
    return
  }
  
  try {
    await saveConversationConfig(
      {
        prompt: localSystemPromptNormal.value,
        use_custom_system_prompt: localUseCustomSystemPromptNormal.value,
      },
      t('conversationSettings.toasts.systemPromptSaved')
    )
    savedSystemPromptNormal = localSystemPromptNormal.value
  } catch (error) {
    console.error('Failed to save System Prompt:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

const handleTemperatureNormalChange = async (value: number) => {
  if (!conversationConfigLoaded.value) return
  if (value === savedTemperatureNormal) return
  
  try {
    await saveConversationConfig(
      { temperature: value },
      t('conversationSettings.toasts.temperatureSaved')
    )
    savedTemperatureNormal = value
  } catch (error) {
    console.error('Failed to save Temperature:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

const handleMaxCompletionTokensChange = async (value: number) => {
  if (!conversationConfigLoaded.value) return
  
  try {
    await saveConversationConfig(
      { max_completion_tokens: value },
      t('conversationSettings.toasts.maxTokensSaved')
    )
    savedMaxCompletionTokens = value
  } catch (error) {
    console.error('Failed to save Max Tokens:', error)
    MessagePlugin.error(getErrorMessage(error))
  }
}

const handleMaxRoundsChange = async (value: number) => {
  try {
    await saveConversationConfig({ max_rounds: value }, t('conversationSettings.toasts.maxRoundsSaved'))
  } catch (error) {
    console.error('Failed to save max_rounds:', error)
    localMaxRounds.value = conversationConfig.value.max_rounds
  }
}

const handleEmbeddingTopKChange = async (value: number) => {
  try {
    await saveConversationConfig({ embedding_top_k: value }, t('conversationSettings.toasts.embeddingSaved'))
  } catch (error) {
    console.error('Failed to save embedding_top_k:', error)
    localEmbeddingTopK.value = conversationConfig.value.embedding_top_k
  }
}

const handleKeywordThresholdChange = async (value: number) => {
  try {
    await saveConversationConfig({ keyword_threshold: value }, t('conversationSettings.toasts.keywordThresholdSaved'))
  } catch (error) {
    console.error('Failed to save keyword_threshold:', error)
    localKeywordThreshold.value = conversationConfig.value.keyword_threshold
  }
}

const handleVectorThresholdChange = async (value: number) => {
  try {
    await saveConversationConfig({ vector_threshold: value }, t('conversationSettings.toasts.vectorThresholdSaved'))
  } catch (error) {
    console.error('Failed to save vector_threshold:', error)
    localVectorThreshold.value = conversationConfig.value.vector_threshold
  }
}

const handleRerankTopKChange = async (value: number) => {
  try {
    await saveConversationConfig({ rerank_top_k: value }, t('conversationSettings.toasts.rerankTopKSaved'))
  } catch (error) {
    console.error('Failed to save rerank_top_k:', error)
    localRerankTopK.value = conversationConfig.value.rerank_top_k
  }
}

const handleRerankThresholdChange = async (value: number) => {
  try {
    await saveConversationConfig({ rerank_threshold: value }, t('conversationSettings.toasts.rerankThresholdSaved'))
  } catch (error) {
    console.error('Failed to save rerank_threshold:', error)
    localRerankThreshold.value = conversationConfig.value.rerank_threshold
  }
}

const handleEnableRewriteChange = async (value: boolean) => {
  try {
    await saveConversationConfig({ enable_rewrite: value }, t('conversationSettings.toasts.enableRewriteSaved'))
  } catch (error) {
    console.error('Failed to save enable_rewrite:', error)
    localEnableRewrite.value = conversationConfig.value.enable_rewrite
  }
}

const handleEnableQueryExpansionChange = async (value: boolean) => {
  try {
    await saveConversationConfig(
      { enable_query_expansion: value },
      t('conversationSettings.toasts.enableQueryExpansionSaved')
    )
  } catch (error) {
    console.error('Failed to save enable_query_expansion:', error)
    localEnableQueryExpansion.value = conversationConfig.value.enable_query_expansion ?? true
  }
}

const handleFallbackStrategyChange = async (value: 'fixed' | 'model') => {
  try {
    await saveConversationConfig({ fallback_strategy: value }, t('conversationSettings.toasts.fallbackStrategySaved'))
  } catch (error) {
    console.error('Failed to save fallback_strategy:', error)
    localFallbackStrategy.value = (conversationConfig.value.fallback_strategy as 'fixed' | 'model') || 'fixed'
  }
}

const handleFallbackResponseChange = async () => {
  if (localFallbackResponse.value === (conversationConfig.value.fallback_response ?? '')) return
  try {
    await saveConversationConfig({ fallback_response: localFallbackResponse.value }, t('conversationSettings.toasts.fallbackResponseSaved'))
  } catch (error) {
    console.error('Failed to save fallback_response:', error)
    localFallbackResponse.value = conversationConfig.value.fallback_response ?? ''
  }
}

const handleRewritePromptSystemChange = async () => {
  if (localRewritePromptSystem.value === (conversationConfig.value.rewrite_prompt_system ?? '')) return
  try {
    await saveConversationConfig({ rewrite_prompt_system: localRewritePromptSystem.value }, t('conversationSettings.toasts.rewritePromptSystemSaved'))
  } catch (error) {
    console.error('Failed to save rewrite_prompt_system:', error)
    localRewritePromptSystem.value = conversationConfig.value.rewrite_prompt_system ?? ''
  }
}

const handleRewritePromptUserChange = async () => {
  if (localRewritePromptUser.value === (conversationConfig.value.rewrite_prompt_user ?? '')) return
  try {
    await saveConversationConfig({ rewrite_prompt_user: localRewritePromptUser.value }, t('conversationSettings.toasts.rewritePromptUserSaved'))
  } catch (error) {
    console.error('Failed to save rewrite_prompt_user:', error)
    localRewritePromptUser.value = conversationConfig.value.rewrite_prompt_user ?? ''
  }
}

const handleFallbackPromptChange = async () => {
  if (localFallbackPrompt.value === (conversationConfig.value.fallback_prompt ?? '')) return
  try {
    await saveConversationConfig({ fallback_prompt: localFallbackPrompt.value }, t('conversationSettings.toasts.fallbackPromptSaved'))
  } catch (error) {
    console.error('Failed to save fallback_prompt:', error)
    localFallbackPrompt.value = conversationConfig.value.fallback_prompt ?? ''
  }
}

const navigateToModelSettings = (subsection: 'chat' | 'rerank') => {
  router.push('/platform/settings')

  setTimeout(() => {
    const event = new CustomEvent('settings-nav', {
      detail: { section: 'models', subsection },
    })
    window.dispatchEvent(event)

    setTimeout(() => {
      const selector = subsection === 'rerank' ? '[data-model-type="rerank"]' : '[data-model-type="chat"]'
      const element = document.querySelector(selector)
      if (element) {
        element.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    }, 200)
  }, 100)
}

const handleConversationSummaryModelChange = async (value: string) => {
  if (value === '__add_model__') {
    localSummaryModelId.value = conversationConfig.value.summary_model_id ?? ''
    navigateToModelSettings('chat')
    return
  }

  try {
    await saveConversationConfig({ summary_model_id: value }, t('conversationSettings.toasts.chatModelSaved'))
  } catch (error) {
    console.error('Failed to save summary_model_id:', error)
    localSummaryModelId.value = conversationConfig.value.summary_model_id ?? ''
  }
}

const handleConversationRerankModelChange = async (value: string) => {
  if (value === '__add_model__') {
    localConversationRerankModelId.value = conversationConfig.value.rerank_model_id ?? ''
    navigateToModelSettings('rerank')
    return
  }

  try {
    await saveConversationConfig({ rerank_model_id: value }, t('conversationSettings.toasts.rerankModelSaved'))
  } catch (error) {
    console.error('Failed to save rerank_model_id:', error)
    localConversationRerankModelId.value = conversationConfig.value.rerank_model_id ?? ''
  }
}
</script>

<style lang="less" scoped>
.agent-settings {
  width: 100%;
}


.section-header {

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

.agent-status-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid #e5e7eb;
  margin-top: 8px;

  .status-label {
    flex: 1;
    max-width: 65%;
    padding-right: 24px;

    label {
      font-size: 15px;
      font-weight: 500;
      color: #333333;
      display: block;
      margin-bottom: 4px;
    }
  }

  .status-control {
    flex-shrink: 0;
    min-width: 280px;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 8px;

    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: 4px;
      font-size: 14px;
      font-weight: 500;

      &.ready {
        background: #f0fdf4;
        color: #16a34a;
        
        .status-icon {
          color: #16a34a;
          font-size: 16px;
        }
      }

      &:not(.ready) {
        background: #fff7ed;
        color: #ea580c;
        
        .status-icon {
          color: #ea580c;
          font-size: 16px;
        }
      }

      .status-text {
        line-height: 1.4;
      }
    }

    .status-hint {
      font-size: 13px;
      color: #666666;
      text-align: right;
      line-height: 1.5;
      max-width: 280px;
    }

    .status-tip {
      margin: 8px 0 0 0;
      font-size: 12px;
      color: #999999;
      text-align: right;
      line-height: 1.5;
      max-width: 280px;
      display: flex;
      align-items: flex-start;
      gap: 4px;
      justify-content: flex-end;

      .tip-icon {
        font-size: 14px;
        color: #999999;
        flex-shrink: 0;
        margin-top: 2px;
      }
    }
  }
}

.model-recommendation-box {
  margin: 20px 0;
  background: #f0fdf6;
  border: 1px solid #d1fae5;
  border-left: 3px solid #07C05F;
  border-radius: 6px;
  padding: 16px;

  .recommendation-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .recommendation-icon {
      font-size: 16px;
      color: #07C05F;
      flex-shrink: 0;
    }

    .recommendation-title {
      font-size: 14px;
      font-weight: 500;
      color: #059669;
    }
  }

  .recommendation-content {
    font-size: 13px;
    line-height: 1.6;
    color: #065f46;

    p {
      margin: 0;
    }
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid #e5e7eb;

  &:last-child {
    border-bottom: none;
  }

  &.vertical {
    flex-direction: column;
    align-items: flex-start;

    .setting-info {
      margin-bottom: 12px;
      max-width: 100%;
    }

    .setting-control.full-width {
      width: 100%;
    }
  }
}

.setting-info {
  flex: 1;
  max-width: 55%;
  word-break: keep-all;
  white-space: normal;

  label {
    font-size: 15px;
    font-weight: 500;
    color: #333333;
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: 13px;
    color: #666666;
    margin: 0;
    line-height: 1.5;
  }

  .hint-tip {
    margin: 8px 0 0 0;
    font-size: 12px;
    color: #999999;
    line-height: 1.5;
    display: flex;
    align-items: flex-start;
    gap: 4px;

    .tip-icon {
      font-size: 14px;
      color: #999999;
      flex-shrink: 0;
      margin-top: 2px;
    }
  }
}

.model-row {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
}

.model-column {
  min-width: 260px;
  flex: 1;
}

.model-column-label {
  font-size: 13px;
  font-weight: 500;
  color: #555;
  margin-bottom: 4px;
}

.model-column-desc {
  margin: 0 0 8px 0;
  font-size: 12px;
  color: #888;
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.slider-with-value {
  display: flex;
  align-items: center;
  gap: 16px;
  justify-content: flex-end;

  .value-display {
    font-size: 14px;
    font-weight: 500;
    color: #333333;
    min-width: 40px;
    text-align: right;
  }
}

// 模型选择器样式
.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .model-icon {
    font-size: 14px;
    color: #07C05F;
  }
  
  .add-icon {
    font-size: 14px;
    color: #07C05F;
  }
  
  .model-name {
    flex: 1;
    font-size: 13px;
  }
  
  &.add {
    .model-name {
      color: #07C05F;
      font-weight: 500;
    }
  }
}

.prompt-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  width: 100%;
}

.prompt-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

.prompt-toggle-label {
  font-size: 13px !important;
  color: #555;
}

.prompt-toggle :deep(.t-switch) {
  font-size: 0;
}

.prompt-toggle :deep(.t-switch__label),
.prompt-toggle :deep(.t-switch__content) {
  font-size: 12px !important;
  line-height: 18px;
  color: #666;
}

.prompt-toggle :deep(.t-switch__label--off),
.prompt-toggle :deep(.t-switch__content) {
  color: #fafafa !important;
}

.prompt-disabled-hint {
  margin: 0 0 8px;
  color: #666;
  font-size: 12px;
}

.prompt-tab-hint {
  margin: 0 0 12px;
  color: #666;
  font-size: 12px;
}

.system-prompt-tabs {
  width: 100%;
}

.allowed-tools-display {
  width: 100%;
}

.allowed-tool-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.allowed-tool-chip {
  background: #f5f7fa;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 12px;
  min-width: 180px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.allowed-tool-label {
  font-size: 13px;
  font-weight: 600;
  color: #1d2129;
}

.allowed-tool-desc {
  font-size: 12px;
  color: #666;
  line-height: 1.4;
}

.allowed-tools-empty {
  margin: 0;
  font-size: 12px;
  color: #999;
}

.prompt-textarea-readonly {
  background-color: #fafafa;
}

.prompt-textarea-wrapper {
  width: 100%;
}

.setting-control.full-width {
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.placeholder-hint {
  margin-top: 12px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.6;

  .hint-title {
    font-weight: 500;
    color: #333;
    margin: 0 0 8px 0;
  }

  .placeholder-list {
    margin: 8px 0;
    padding-left: 20px;
    color: #666;

    li {
      margin: 4px 0;

      code {
        background: #fff;
        padding: 2px 6px;
        border-radius: 3px;
        font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
        font-size: 11px;
        color: #e83e8c;
        border: 1px solid #e1e8ed;
      }
    }
  }

  .hint-tip {
    margin: 8px 0 0 0;
    color: #999;
    font-style: italic;
  }
}

.placeholder-popup-wrapper {
  position: fixed;
  z-index: 10001;
  pointer-events: auto;
}

.placeholder-popup {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  max-width: 400px;
  max-height: 300px;
  overflow-y: auto;
  padding: 4px 0;
}

.placeholder-item {
  padding: 8px 12px;
  cursor: pointer;
  transition: background-color 0.2s;

  &:hover,
  &.active {
    background-color: #f5f7fa;
  }

  .placeholder-name {
    font-weight: 500;
    margin-bottom: 4px;

    code {
      background: #f5f7fa;
      padding: 2px 6px;
      border-radius: 3px;
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
      font-size: 12px;
      color: #e83e8c;
    }
  }

  .placeholder-desc {
    font-size: 12px;
    color: #666;
    line-height: 1.4;
  }
}

</style>

