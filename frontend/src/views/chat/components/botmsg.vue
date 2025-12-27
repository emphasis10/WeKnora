<template>
    <div class="bot_msg">
        <div style="display: flex;flex-direction: column; gap:8px">
            <!-- Display @ mentioned knowledge bases and files (visible in non-Agent mode) -->
            <div v-if="!session.isAgentMode && mentionedItems && mentionedItems.length > 0" class="mentioned_items">
                <span 
                    v-for="item in mentionedItems" 
                    :key="item.id" 
                    class="mentioned_tag"
                    :class="[
                      item.type === 'kb' ? (item.kb_type === 'faq' ? 'faq-tag' : 'kb-tag') : 'file-tag'
                    ]"
                >
                    <span class="tag_icon">
                        <t-icon v-if="item.type === 'kb'" :name="item.kb_type === 'faq' ? 'chat-bubble-help' : 'folder'" />
                        <t-icon v-else name="file" />
                    </span>
                    <span class="tag_name">{{ item.name }}</span>
                </span>
            </div>
            <docInfo :session="session"></docInfo>
            <AgentStreamDisplay :session="session" :user-query="userQuery" v-if="session.isAgentMode"></AgentStreamDisplay>
            <deepThink :deepSession="session" v-if="session.showThink && !session.isAgentMode"></deepThink>
        </div>
        <!-- Traditional markdown rendering only displayed in non-Agent mode -->
        <div ref="parentMd" v-if="!session.hideContent && !session.isAgentMode">
            <!-- Render loading animation if message is summarizing -->
            <div v-if="session.thinking" class="thinking-loading">
                <div class="loading-typing">
                    <span></span>
                    <span></span>
                    <span></span>
                </div>
            </div>
            <!-- Render complete content directly to avoid issues caused by splitting, style consistent with thinking -->
            <div class="content-wrapper">
                <div class="ai-markdown-template markdown-content">
                    <div v-for="(token, index) in markdownTokens" :key="index" v-html="renderToken(token)"></div>
                </div>
            </div>
            <!-- Copy and Add to Knowledge Base buttons - visible in non-Agent mode -->
            <div v-if="session.is_completed && (content || session.content)" class="answer-toolbar">
                <t-button size="small" variant="outline" shape="round" @click.stop="handleCopyAnswer" :title="$t('agent.copy')">
                    <t-icon name="copy" />
                </t-button>
                <t-button size="small" variant="outline" shape="round" @click.stop="handleAddToKnowledge" :title="$t('agent.addToKnowledgeBase')">
                    <t-icon name="add" />
                </t-button>
            </div>
            <div v-if="isImgLoading" class="img_loading"><t-loading size="small"></t-loading><span>{{ $t('common.loading') }}</span></div>
        </div>
        <picturePreview :reviewImg="reviewImg" :reviewUrl="reviewUrl" @closePreImg="closePreImg"></picturePreview>
    </div>
</template>
<script setup>
import { onMounted, onBeforeUnmount, watch, computed, ref, reactive, defineProps, nextTick } from 'vue';
import { marked } from 'marked';
import docInfo from './docInfo.vue';
import deepThink from './deepThink.vue';
import AgentStreamDisplay from './AgentStreamDisplay.vue';
import picturePreview from '@/components/picture-preview.vue';
import { sanitizeHTML, safeMarkdownToHTML, createSafeImage, isValidImageURL } from '@/utils/security';
import { useI18n } from 'vue-i18n';
import { MessagePlugin } from 'tdesign-vue-next';
import { useUIStore } from '@/stores/ui';

marked.use({
    mangle: false,
    headerIds: false,
    breaks: true,  // 全局启用单个换行支持
});
const emit = defineEmits(['scroll-bottom'])
const { t } = useI18n()
const uiStore = useUIStore();
const renderer = new marked.Renderer();
let parentMd = ref()
let reviewUrl = ref('')
let reviewImg = ref(false)
let isImgLoading = ref(false);
const props = defineProps({
    // Required fields
    content: {
        type: String,
        required: false
    },
    session: {
        type: Object,
        required: false
    },
    userQuery: {
        type: String,
        required: false,
        default: ''
    },
    isFirstEnter: {
        type: Boolean,
        required: false
    }
});

const preview = (url) => {
    nextTick(() => {
        reviewUrl.value = url;
        reviewImg.value = true
    })
}

const closePreImg = () => {
    reviewImg.value = false
    reviewUrl.value = '';
}

// Create custom renderer instance
const customRenderer = new marked.Renderer();
// Override image rendering method
customRenderer.image = function(href, title, text) {
    // Validate image URL safety
    if (!isValidImageURL(href)) {
        return `<p>${t('error.invalidImageLink')}</p>`;
    }
    // Use safe image creation function
    return createSafeImage(href, text || '', title || '');
};

// Computed property: convert Markdown text to tokens
const markdownTokens = computed(() => {
    const text = props.content || props.session?.content || '';
    if (!text || typeof text !== 'string') {
        return [];
    }
    
    // First safely handle Markdown content
    const safeMarkdown = safeMarkdownToHTML(text);
    
    // Use marked.lexer for tokenization
    return marked.lexer(safeMarkdown);
});

// Render single token to HTML
const renderToken = (token) => {
    try {
        // Create temporary marked configuration
        const markedOptions = {
            renderer: customRenderer,
            breaks: true
        };
        
        // Parse single token
        // marked.parser accepts token array
        let html = marked.parser([token], markedOptions);
        
        // Use DOMPurify for final safety cleanup
        return sanitizeHTML(html);
    } catch (e) {
        console.error('Token rendering error:', e);
        return '';
    }
};

const myMarkdown = (res) => {
    return marked.parse(res, { renderer })
}

// Get actual content
const getActualContent = () => {
    return (props.content || props.session?.content || '').trim();
};

// Format title
const formatManualTitle = (question) => {
    if (!question) {
        return 'Session Excerpt';
    }
    const condensed = question.replace(/\s+/g, ' ').trim();
    if (!condensed) {
        return 'Session Excerpt';
    }
    return condensed.length > 40 ? `${condensed.slice(0, 40)}...` : condensed;
};

// Build manually added Markdown content
const buildManualMarkdown = (question, answer) => {
    const safeAnswer = answer?.trim() || '(No response content)';
    return `${safeAnswer}`;
};

// Copy response content
const handleCopyAnswer = async () => {
    const content = getActualContent();
    if (!content) {
        MessagePlugin.warning(t('chat.emptyContentWarning') || 'Current response is empty, cannot copy');
        return;
    }

    try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            await navigator.clipboard.writeText(content);
            MessagePlugin.success(t('chat.copySuccess') || 'Copied to clipboard');
        } else {
            const textArea = document.createElement('textarea');
            textArea.value = content;
            textArea.style.position = 'fixed';
            textArea.style.opacity = '0';
            document.body.appendChild(textArea);
            textArea.select();
            document.execCommand('copy');
            document.body.removeChild(textArea);
            MessagePlugin.success(t('chat.copySuccess') || 'Copied to clipboard');
        }
    } catch (err) {
        console.error('Copy failed:', err);
        MessagePlugin.error(t('chat.copyFailed') || 'Copy failed, please copy manually');
    }
};

// Add to knowledge base
const handleAddToKnowledge = () => {
    const content = getActualContent();
    if (!content) {
        MessagePlugin.warning(t('chat.emptyContentWarning') || 'Current response is empty, cannot save to knowledge base');
        return;
    }

    const question = (props.userQuery || '').trim();
    const manualContent = buildManualMarkdown(question, content);
    const manualTitle = formatManualTitle(question);

    uiStore.openManualEditor({
        mode: 'create',
        title: manualTitle,
        content: manualContent,
        status: 'draft',
    });

    MessagePlugin.info(t('chat.editorOpened') || 'Editor opened, please select a knowledge base and save');
};

// Handle click event on image in markdown-content
const handleMarkdownImageClick = (e) => {
    const target = e.target;
    if (target && target.tagName === 'IMG') {
        const src = target.getAttribute('src');
        if (src) {
            e.preventDefault();
            e.stopPropagation();
            preview(src);
        }
    }
};

onMounted(async () => {
    // Add click event to images in markdown-content
    nextTick(() => {
        if (parentMd.value) {
            parentMd.value.addEventListener('click', handleMarkdownImageClick, true);
        }
    });
});

onBeforeUnmount(() => {
    if (parentMd.value) {
        parentMd.value.removeEventListener('click', handleMarkdownImageClick, true);
    }
});
</script>
<style lang="less" scoped>
@import '../../../components/css/markdown.less';

// Content wrapper - consistent with answer style in Agent mode
.content-wrapper {
    background: #ffffff;
    border-radius: 6px;
    padding: 8px 12px;
    border: 1px solid #07c05f;
    box-shadow: 0 1px 3px rgba(7, 192, 95, 0.06);
    transition: all 0.2s ease;
}

@keyframes fadeInUp {
    from {
        opacity: 0;
        transform: translateY(8px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.ai-markdown-template {
    font-size: 13px;
    color: #374151;
    line-height: 1.6;
}

.markdown-content {
    :deep(p) {
        margin: 6px 0;
        line-height: 1.6;
    }

    :deep(code) {
        background: #f3f4f6;
        padding: 2px 5px;
        border-radius: 3px;
        font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
        font-size: 11px;
    }

    :deep(pre) {
        background: #f9fafb;
        padding: 10px;
        border-radius: 4px;
        overflow-x: auto;
        margin: 6px 0;

        code {
            background: none;
            padding: 0;
        }
    }

    :deep(ul), :deep(ol) {
        margin: 6px 0;
        padding-left: 20px;
    }

    :deep(li) {
        margin: 3px 0;
    }

    :deep(blockquote) {
        border-left: 2px solid #07c05f;
        padding-left: 10px;
        margin: 6px 0;
        color: #6b7280;
    }

    :deep(h1), :deep(h2), :deep(h3), :deep(h4), :deep(h5), :deep(h6) {
        margin: 10px 0 6px 0;
        font-weight: 600;
        color: #374151;
    }

    :deep(a) {
        color: #07c05f;
        text-decoration: none;

        &:hover {
            text-decoration: underline;
        }
    }

    :deep(table) {
        border-collapse: collapse;
        margin: 6px 0;
        font-size: 11px;
        width: 100%;

        th, td {
            border: 1px solid #e5e7eb;
            padding: 5px 8px;
            text-align: left;
        }

        th {
            background: #f9fafb;
            font-weight: 600;
        }

        tbody tr:nth-child(even) {
            background: #fafafa;
        }
    }

    :deep(img) {
        max-width: 80%;
        max-height: 300px;
        width: auto;
        height: auto;
        border-radius: 8px;
        display: block;
        margin: 8px 0;
        border: 0.5px solid #e5e7eb;
        object-fit: contain;
        cursor: pointer;
        transition: transform 0.2s ease;

        &:hover {
            transform: scale(1.02);
        }
    }
}

.ai-markdown-img {
    max-width: 80%;
    max-height: 300px;
    width: auto;
    height: auto;
    border-radius: 8px;
    display: block;
    cursor: pointer;
    object-fit: contain;
    margin: 8px 0 8px 16px;
    border: 0.5px solid #E7E7E7;
    transition: transform 0.2s ease;

    &:hover {
        transform: scale(1.02);
    }
}

.bot_msg {
    // background: #fff;
    border-radius: 4px;
    color: rgba(0, 0, 0, 0.9);
    font-size: 16px;
    // padding: 10px 12px;
    margin-right: auto;
    max-width: 100%;
    box-sizing: border-box;
}

.botanswer_laoding_gif {
    width: 24px;
    height: 18px;
    margin-left: 16px;
}

.thinking-loading {
    margin-left: 16px;
    margin-bottom: 8px;
}

.loading-typing {
    display: flex;
    align-items: center;
    gap: 4px;
    
    span {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: #07c05f;
        animation: typingBounce 1.4s ease-in-out infinite;
        
        &:nth-child(1) {
            animation-delay: 0s;
        }
        
        &:nth-child(2) {
            animation-delay: 0.2s;
        }
        
        &:nth-child(3) {
            animation-delay: 0.4s;
        }
    }
}

@keyframes typingBounce {
    0%, 60%, 100% {
        transform: translateY(0);
    }
    30% {
        transform: translateY(-8px);
    }
}

// Toolbar for copy and add to knowledge base buttons
.answer-toolbar {
    display: flex;
    justify-content: flex-start;
    gap: 6px;
    margin-top: 8px;
    min-height: 32px;

    :deep(.t-button) {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-width: auto;
        width: auto;
        border: 1px solid #e0e0e0;
        border-radius: 6px;
        background: #ffffff;
        color: #666;
        transition: all 0.2s ease;
        
        .t-button__content {
            display: inline-flex !important;
            align-items: center;
            justify-content: center;
            gap: 0;
        }
        
        .t-button__text {
            display: inline-flex !important;
            align-items: center;
            justify-content: center;
            gap: 0;
        }
        
        .t-icon {
            display: inline-flex !important;
            visibility: visible !important;
            opacity: 1 !important;
            align-items: center;
            justify-content: center;
            font-size: 16px;
            width: 16px;
            height: 16px;
            flex-shrink: 0;
            color: #666;
        }
        
        .t-icon svg {
            display: block !important;
            width: 16px;
            height: 16px;
        }
        
        .t-button__text > :not(.t-icon) {
            display: none;
        }
        
        &:hover:not(:disabled) {
            background: rgba(7, 192, 95, 0.08);
            border-color: rgba(7, 192, 95, 0.3);
            color: #07c05f;
            
            .t-icon {
                color: #07c05f;
            }
        }
        
        &:active:not(:disabled) {
            background: rgba(7, 192, 95, 0.12);
            border-color: rgba(7, 192, 95, 0.4);
            transform: translateY(0.5px);
        }
    }
}

.img_loading {
    background: #3032360f;
    height: 230px;
    width: 230px;
    color: #00000042;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
    font-size: 12px;
    gap: 4px;
    margin-left: 16px;
    border-radius: 8px;
}

:deep(.t-loading__gradient-conic) {
    background: conic-gradient(from 90deg at 50% 50%, #fff 0deg, #676767 360deg) !important;

}
</style>