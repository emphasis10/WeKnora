package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// QueryKnowledgeGraphTool queries the knowledge graph for entities and relationships
type QueryKnowledgeGraphTool struct {
	BaseTool
	knowledgeService interfaces.KnowledgeBaseService
}

// NewQueryKnowledgeGraphTool creates a new query knowledge graph tool
func NewQueryKnowledgeGraphTool(knowledgeService interfaces.KnowledgeBaseService) *QueryKnowledgeGraphTool {
	description := `Query knowledge graph to explore entity relationships and knowledge networks.

## Core Function
Explores relationships between entities in knowledge bases that have graph extraction configured.

## When to Use
✅ **Use for**:
- Understanding relationships between entities (e.g., "relationship between Docker and Kubernetes")
- Exploring knowledge networks and concept associations
- Finding related information about specific entities
- Understanding technical architecture and system relationships

❌ **Don't use for**:
- General text search → use knowledge_search
- Knowledge base without graph extraction configured
- Need exact document content → use knowledge_search

## Parameters
- **knowledge_base_ids** (required): Array of knowledge base IDs (1-10). Only KBs with graph extraction configured will be effective.
- **query** (required): Query content - can be entity name, relationship query, or concept search.

## Graph Configuration
Knowledge graph must be pre-configured in knowledge bases:
- **Entity types** (Nodes): e.g., "Technology", "Tool", "Concept"
- **Relationship types** (Relations): e.g., "depends_on", "uses", "contains"

If KB is not configured with graph, tool will return regular search results.

## Workflow
1. **Relationship exploration**: query_knowledge_graph → list_knowledge_chunks (for detailed content)
2. **Network analysis**: query_knowledge_graph → knowledge_search (for comprehensive understanding)
3. **Topic research**: knowledge_search → query_knowledge_graph (for deep entity relationships)

## Notes
- Results indicate graph configuration status
- Cross-KB results are automatically deduplicated
- Results are sorted by relevance`

	return &QueryKnowledgeGraphTool{
		BaseTool:         NewBaseTool("query_knowledge_graph", description),
		knowledgeService: knowledgeService,
	}
}

// Parameters returns the JSON schema for the tool's parameters
func (t *QueryKnowledgeGraphTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"knowledge_base_ids": map[string]interface{}{
				"type":        "array",
				"description": "Array of knowledge base IDs to query",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 10,
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Query content (entity name or query text)",
			},
		},
		"required": []string{"knowledge_base_ids", "query"},
	}
}

// Execute performs the knowledge graph query with concurrent KB processing
func (t *QueryKnowledgeGraphTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	// Extract knowledge_base_ids array
	kbIDsRaw, ok := args["knowledge_base_ids"].([]interface{})
	if !ok || len(kbIDsRaw) == 0 {
		return &types.ToolResult{
			Success: false,
			Error:   "knowledge_base_ids is required and must be a non-empty array",
		}, fmt.Errorf("knowledge_base_ids is required")
	}

	// Convert to string slice
	var kbIDs []string
	for _, id := range kbIDsRaw {
		if idStr, ok := id.(string); ok && idStr != "" {
			kbIDs = append(kbIDs, idStr)
		}
	}

	if len(kbIDs) == 0 {
		return &types.ToolResult{
			Success: false,
			Error:   "knowledge_base_ids must contain at least one valid KB ID",
		}, fmt.Errorf("no valid KB IDs provided")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "query is required",
		}, fmt.Errorf("invalid query")
	}

	// Concurrently query all knowledge bases
	type graphQueryResult struct {
		kbID    string
		kb      *types.KnowledgeBase
		results []*types.SearchResult
		err     error
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	kbResults := make(map[string]*graphQueryResult)

	searchParams := types.SearchParams{
		QueryText:  query,
		MatchCount: 10,
	}

	for _, kbID := range kbIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Get knowledge base to check graph configuration
			kb, err := t.knowledgeService.GetKnowledgeBaseByID(ctx, id)
			if err != nil {
				mu.Lock()
				kbResults[id] = &graphQueryResult{kbID: id, err: fmt.Errorf("failed to fetch knowledge base: %v", err)}
				mu.Unlock()
				return
			}

			// Check if graph extraction is enabled
			if kb.ExtractConfig == nil || (len(kb.ExtractConfig.Nodes) == 0 && len(kb.ExtractConfig.Relations) == 0) {
				mu.Lock()
				kbResults[id] = &graphQueryResult{kbID: id, err: fmt.Errorf("knowledge graph extraction not configured")}
				mu.Unlock()
				return
			}

			// Query graph
			results, err := t.knowledgeService.HybridSearch(ctx, id, searchParams)
			if err != nil {
				mu.Lock()
				kbResults[id] = &graphQueryResult{kbID: id, kb: kb, err: fmt.Errorf("query failed: %v", err)}
				mu.Unlock()
				return
			}

			mu.Lock()
			kbResults[id] = &graphQueryResult{kbID: id, kb: kb, results: results}
			mu.Unlock()
		}(kbID)
	}

	wg.Wait()

	// Collect and deduplicate results
	seenChunks := make(map[string]*types.SearchResult)
	var errors []string
	graphConfigs := make(map[string]map[string]interface{})
	kbCounts := make(map[string]int)

	for _, kbID := range kbIDs {
		result := kbResults[kbID]
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("KB %s: %v", kbID, result.err))
			continue
		}

		if result.kb != nil && result.kb.ExtractConfig != nil {
			graphConfigs[kbID] = map[string]interface{}{
				"nodes":     result.kb.ExtractConfig.Nodes,
				"relations": result.kb.ExtractConfig.Relations,
			}
		}

		kbCounts[kbID] = len(result.results)
		for _, r := range result.results {
			if _, seen := seenChunks[r.ID]; !seen {
				seenChunks[r.ID] = r
			}
		}
	}

	// Convert map to slice and sort by score
	allResults := make([]*types.SearchResult, 0, len(seenChunks))
	for _, result := range seenChunks {
		allResults = append(allResults, result)
	}

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	if len(allResults) == 0 {
		return &types.ToolResult{
			Success: true,
			Output:  "No relevant graph information found.",
			Data: map[string]interface{}{
				"knowledge_base_ids": kbIDs,
				"query":              query,
				"results":            []interface{}{},
				"graph_configs":      graphConfigs,
				"errors":             errors,
			},
		}, nil
	}

	// Format output with enhanced graph information
	output := "=== Knowledge Graph Query ===\n\n"
	output += fmt.Sprintf("📊 Query: %s\n", query)
	output += fmt.Sprintf("🎯 Target Knowledge Bases: %v\n", kbIDs)
	output += fmt.Sprintf("✓ Found %d unique relevant results\n\n", len(allResults))

	if len(errors) > 0 {
		output += "=== ⚠️ Partial Failures ===\n"
		for _, errMsg := range errors {
			output += fmt.Sprintf("  - %s\n", errMsg)
		}
		output += "\n"
	}

	// Display graph configuration status
	hasGraphConfig := false
	output += "=== 📈 Graph Configuration Status ===\n\n"
	for kbID, config := range graphConfigs {
		hasGraphConfig = true
		output += fmt.Sprintf("Knowledge Base [%s]:\n", kbID)

		nodes, _ := config["nodes"].([]interface{})
		relations, _ := config["relations"].([]interface{})

		if len(nodes) > 0 {
			output += fmt.Sprintf("  ✓ Entity Types (%d): ", len(nodes))
			nodeNames := make([]string, 0, len(nodes))
			for _, n := range nodes {
				if nodeMap, ok := n.(map[string]interface{}); ok {
					if name, ok := nodeMap["name"].(string); ok {
						nodeNames = append(nodeNames, name)
					}
				}
			}
			output += fmt.Sprintf("%v\n", nodeNames)
		} else {
			output += "  ⚠️ No entity types configured\n"
		}

		if len(relations) > 0 {
			output += fmt.Sprintf("  ✓ Relationship Types (%d): ", len(relations))
			relNames := make([]string, 0, len(relations))
			for _, r := range relations {
				if relMap, ok := r.(map[string]interface{}); ok {
					if name, ok := relMap["name"].(string); ok {
						relNames = append(relNames, name)
					}
				}
			}
			output += fmt.Sprintf("%v\n", relNames)
		} else {
			output += "  ⚠️ No relationship types configured\n"
		}
		output += "\n"
	}

	if !hasGraphConfig {
		output += "⚠️ None of the queried knowledge bases have graph extraction configured\n"
		output += "💡 Tip: Entity and relationship types need to be configured in the knowledge base settings\n\n"
	}

	// Display result counts by KB
	if len(kbCounts) > 0 {
		output += "=== 📚 Knowledge Base Coverage ===\n"
		for kbID, count := range kbCounts {
			output += fmt.Sprintf("  - %s: %d results\n", kbID, count)
		}
		output += "\n"
	}

	// Display search results
	output += "=== 🔍 Query Results ===\n\n"
	if !hasGraphConfig {
		output += "💡 Currently returning related document snippets (knowledge base not configured for graph)\n\n"
	} else {
		output += "💡 Content retrieval based on graph configuration\n\n"
	}

	formattedResults := make([]map[string]interface{}, 0, len(allResults))
	currentKB := ""

	for i, result := range allResults {
		// Group by knowledge base
		if result.KnowledgeID != currentKB {
			currentKB = result.KnowledgeID
			if i > 0 {
				output += "\n"
			}
			output += fmt.Sprintf("[Source Document: %s]\n\n", result.KnowledgeTitle)
		}

		relevanceLevel := GetRelevanceLevel(result.Score)

		output += fmt.Sprintf("Result #%d:\n", i+1)
		output += fmt.Sprintf("  📍 Relevance: %.2f (%s)\n", result.Score, relevanceLevel)
		output += fmt.Sprintf("  🔗 Match Type: %s\n", FormatMatchType(result.MatchType))
		output += fmt.Sprintf("  📄 Content: %s\n", result.Content)
		output += fmt.Sprintf("  🆔 chunk_id: %s\n\n", result.ID)

		formattedResults = append(formattedResults, map[string]interface{}{
			"result_index":    i + 1,
			"chunk_id":        result.ID,
			"content":         result.Content,
			"score":           result.Score,
			"relevance_level": relevanceLevel,
			"knowledge_id":    result.KnowledgeID,
			"knowledge_title": result.KnowledgeTitle,
			"match_type":      FormatMatchType(result.MatchType),
		})
	}

	output += "=== 💡 Usage Tips ===\n"
	output += "- ✓ Results have been cross-KB deduplicated and sorted by relevance\n"
	output += "- ✓ Use get_chunk_detail for full content\n"
	output += "- ✓ Use list_knowledge_chunks to explore context\n"
	if !hasGraphConfig {
		output += "- ⚠️ Configure graph extraction for more accurate entity relationship results\n"
	}
	output += "- ⏳ Full graph query language (Cypher) support is under development\n"

	// Build structured graph data for frontend visualization
	graphData := buildGraphVisualizationData(allResults, graphConfigs)

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"knowledge_base_ids": kbIDs,
			"query":              query,
			"results":            formattedResults,
			"count":              len(allResults),
			"kb_counts":          kbCounts,
			"graph_configs":      graphConfigs,
			"graph_data":         graphData,
			"has_graph_config":   hasGraphConfig,
			"errors":             errors,
			"display_type":       "graph_query_results",
		},
	}, nil
}

// buildGraphVisualizationData builds structured data for graph visualization
func buildGraphVisualizationData(
	results []*types.SearchResult,
	graphConfigs map[string]map[string]interface{},
) map[string]interface{} {
	// Build a simple graph structure for frontend visualization
	nodes := make([]map[string]interface{}, 0)
	edges := make([]map[string]interface{}, 0)

	// Create nodes from results
	seenEntities := make(map[string]bool)
	for i, result := range results {
		if !seenEntities[result.ID] {
			nodes = append(nodes, map[string]interface{}{
				"id":       result.ID,
				"label":    fmt.Sprintf("Chunk %d", i+1),
				"content":  result.Content,
				"kb_id":    result.KnowledgeID,
				"kb_title": result.KnowledgeTitle,
				"score":    result.Score,
				"type":     "chunk",
			})
			seenEntities[result.ID] = true
		}
	}

	return map[string]interface{}{
		"nodes":       nodes,
		"edges":       edges,
		"total_nodes": len(nodes),
		"total_edges": len(edges),
	}
}
