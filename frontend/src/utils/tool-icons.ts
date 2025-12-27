/**
 * Tool Icons Utility
 * Maps tool names and match types to icons for better UI display
 */

// Tool name to icon mapping
export const toolIcons: Record<string, string> = {
    multi_kb_search: '🔍',
    knowledge_search: '📚',
    grep_chunks: '🔎',
    get_chunk_detail: '📄',
    list_knowledge_bases: '📂',
    list_knowledge_chunks: '🧩',
    get_document_info: 'ℹ️',
    query_knowledge_graph: '🕸️',
    think: '💭',
    todo_write: '📋',
};

// Match type to icon mapping
export const matchTypeIcons: Record<string, string> = {
    'Vector Match': '🎯',
    'Keyword Match': '🔤',
    'Neighbor Match': '📌',
    'History Match': '📜',
    'Parent Match': '⬆️',
    'Relation Match': '🔗',
    'Graph Match': '🕸️',
};

// Get icon for a tool name
export function getToolIcon(toolName: string): string {
    return toolIcons[toolName] || '🛠️';
}

// Get icon for a match type
export function getMatchTypeIcon(matchType: string): string {
    return matchTypeIcons[matchType] || '📍';
}

// Get tool display name (user-friendly)
export function getToolDisplayName(toolName: string): string {
    const displayNames: Record<string, string> = {
        multi_kb_search: 'Multi-KB Search',
        knowledge_search: 'Knowledge Search',
        grep_chunks: 'Text Search',
        get_chunk_detail: 'Get Chunk Detail',
        list_knowledge_chunks: 'List Knowledge Chunks',
        list_knowledge_bases: 'List Knowledge Bases',
        get_document_info: 'Get Document Info',
        query_knowledge_graph: 'Query Knowledge Graph',
        think: 'Deep Thinking',
        todo_write: 'Make Plan',
    };
    return displayNames[toolName] || toolName;
}

