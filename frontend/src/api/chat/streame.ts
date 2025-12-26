import { fetchEventSource } from '@microsoft/fetch-event-source'
import { ref, type Ref, onUnmounted, nextTick } from 'vue'
import { generateRandomString } from '@/utils/index';



interface StreamOptions {
  // Request method (default POST)
  method?: 'GET' | 'POST'
  // Request headers
  headers?: Record<string, string>
  // Request body automatically serialized
  body?: Record<string, any>
  // Stream rendering interval (ms)
  chunkInterval?: number
}

export function useStream() {
  // Reactive state
  const output = ref('')              // Displayed content
  const isStreaming = ref(false)      // Streaming status
  const isLoading = ref(false)        // Initial loading status
  const error = ref<string | null>(null)// Error message
  let controller = new AbortController()

  // Stream rendering buffer
  let buffer: string[] = []
  let renderTimer: number | null = null

  // Start stream request
  const startStream = async (params: { session_id: any; query: any; knowledge_base_ids?: string[]; knowledge_ids?: string[]; agent_enabled?: boolean; web_search_enabled?: boolean; summary_model_id?: string; mcp_service_ids?: string[]; mentioned_items?: Array<{ id: string; name: string; type: string; kb_type?: string }>; method: string; url: string }) => {
    // Reset state
    output.value = '';
    error.value = null;
    isStreaming.value = true;
    isLoading.value = true;

    // Get API configuration
    const apiUrl = import.meta.env.VITE_IS_DOCKER ? "" : "http://localhost:8080";

    // Get JWT Token
    const token = localStorage.getItem('weknora_token');
    if (!token) {
      error.value = "Login token not found, please log in again";
      stopStream();
      return;
    }

    // Get cross-tenant access request headers
    const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id');
    const defaultTenantId = localStorage.getItem('weknora_tenant');
    let tenantIdHeader: string | null = null;
    if (selectedTenantId) {
      try {
        const defaultTenant = defaultTenantId ? JSON.parse(defaultTenantId) : null;
        const defaultId = defaultTenant?.id ? String(defaultTenant.id) : null;
        if (selectedTenantId !== defaultId) {
          tenantIdHeader = selectedTenantId;
        }
      } catch (e) {
        console.error('Failed to parse tenant info', e);
      }
    }

    // Validate knowledge_base_ids for agent-chat requests
    // Note: knowledge_base_ids can be empty if user hasn't selected any, but we allow it
    // The backend will handle the case when no knowledge bases are selected
    const isAgentChat = params.url === '/api/v1/agent-chat';
    // Removed validation - allow empty knowledge_base_ids array
    // The backend should handle this case appropriately

    try {
      let url =
        params.method == "POST"
          ? `${apiUrl}${params.url}/${params.session_id}`
          : `${apiUrl}${params.url}/${params.session_id}?message_id=${params.query}`;

      // Prepare POST body with required fields for agent-chat
      // knowledge_base_ids array and agent_enabled can update Session's SessionAgentConfig
      const postBody: any = {
        query: params.query,
        agent_enabled: params.agent_enabled !== undefined ? params.agent_enabled : true
      };
      // Always include knowledge_base_ids for agent-chat (already validated above)
      if (params.knowledge_base_ids !== undefined && params.knowledge_base_ids.length > 0) {
        postBody.knowledge_base_ids = params.knowledge_base_ids;
      }
      // Include knowledge_ids if provided
      if (params.knowledge_ids !== undefined && params.knowledge_ids.length > 0) {
        postBody.knowledge_ids = params.knowledge_ids;
      }
      // Include web_search_enabled if provided
      if (params.web_search_enabled !== undefined) {
        postBody.web_search_enabled = params.web_search_enabled;
      }
      // Include summary_model_id if provided (for non-Agent mode)
      if (params.summary_model_id) {
        postBody.summary_model_id = params.summary_model_id;
      }
      // Include mcp_service_ids if provided (for Agent mode)
      if (params.mcp_service_ids !== undefined && params.mcp_service_ids.length > 0) {
        postBody.mcp_service_ids = params.mcp_service_ids;
      }
      // Include mentioned_items if provided (for displaying @mentions in chat)
      if (params.mentioned_items !== undefined && params.mentioned_items.length > 0) {
        postBody.mentioned_items = params.mentioned_items;
      }

      await fetchEventSource(url, {
        method: params.method,
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
          "X-Request-ID": `${generateRandomString(12)}`,
          ...(tenantIdHeader ? { "X-Tenant-ID": tenantIdHeader } : {}),
        },
        body:
          params.method == "POST"
            ? JSON.stringify(postBody)
            : null,
        signal: controller.signal,
        openWhenHidden: true,

        onopen: async (res) => {
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          isLoading.value = false;
        },

        onmessage: (ev) => {
          buffer.push(JSON.parse(ev.data)); // Push data into buffer
          // Execute custom processing
          if (chunkHandler) {
            chunkHandler(JSON.parse(ev.data));
          }
        },

        onerror: (err) => {
          throw new Error(`Stream connection failed: ${err}`);
        },

        onclose: () => {
          stopStream();
        },
      });
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      stopStream()
    }
  }

  let chunkHandler: ((data: any) => void) | null = null
  // Register chunk handler
  const onChunk = (handler: () => void) => {
    chunkHandler = handler
  }


  // Stop stream
  const stopStream = () => {
    controller.abort();
    controller = new AbortController(); // Reset controller (for potential restart)
    isStreaming.value = false;
    isLoading.value = false;
  }

  // Auto cleanup on component unmount
  onUnmounted(stopStream)

  return {
    output,          // Displayed content
    isStreaming,     // Whether streaming is in progress
    isLoading,       // Initial connection status
    error,
    onChunk,
    startStream,     // Start stream
    stopStream       // Manual stop
  }
}