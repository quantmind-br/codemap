package graph

import (
	"strings"
)

// QueryResult represents a search result with context.
type QueryResult struct {
	Node  *Node   `json:"node"`
	Score float64 `json:"score,omitempty"`
	Path  []*Node `json:"path,omitempty"` // For path queries
}

// PathResult represents a path between two nodes.
type PathResult struct {
	From   *Node   `json:"from"`
	To     *Node   `json:"to"`
	Path   []*Node `json:"path"`
	Edges  []*Edge `json:"edges"`
	Length int     `json:"length"`
}

// FindNodesByPattern searches for nodes matching a pattern in name or path.
func (g *CodeGraph) FindNodesByPattern(pattern string, kinds []NodeKind) []*Node {
	pattern = strings.ToLower(pattern)
	var results []*Node

	kindSet := make(map[NodeKind]bool)
	for _, k := range kinds {
		kindSet[k] = true
	}

	for _, node := range g.Nodes {
		// Filter by kind if specified
		if len(kinds) > 0 && !kindSet[node.Kind] {
			continue
		}

		// Match name or path
		nameLower := strings.ToLower(node.Name)
		pathLower := strings.ToLower(node.Path)

		if strings.Contains(nameLower, pattern) || strings.Contains(pathLower, pattern) {
			results = append(results, node)
		}
	}

	return results
}

// FindPath finds the shortest path between two nodes using BFS.
// Returns nil if no path exists.
func (g *CodeGraph) FindPath(fromID, toID NodeID, maxDepth int) *PathResult {
	if maxDepth <= 0 {
		maxDepth = 10 // Default max depth
	}

	from := g.GetNode(fromID)
	to := g.GetNode(toID)
	if from == nil || to == nil {
		return nil
	}

	// BFS with path tracking
	type queueItem struct {
		nodeID NodeID
		path   []NodeID
		edges  []*Edge
	}

	visited := make(map[NodeID]bool)
	queue := []queueItem{{nodeID: fromID, path: []NodeID{fromID}, edges: nil}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if len(current.path) > maxDepth {
			continue
		}

		if current.nodeID == toID {
			// Build result
			path := make([]*Node, len(current.path))
			for i, id := range current.path {
				path[i] = g.GetNode(id)
			}
			return &PathResult{
				From:   from,
				To:     to,
				Path:   path,
				Edges:  current.edges,
				Length: len(path) - 1,
			}
		}

		if visited[current.nodeID] {
			continue
		}
		visited[current.nodeID] = true

		// Explore outgoing edges
		for _, edge := range g.GetOutgoingEdges(current.nodeID) {
			if !visited[edge.To] {
				newPath := make([]NodeID, len(current.path)+1)
				copy(newPath, current.path)
				newPath[len(current.path)] = edge.To

				newEdges := make([]*Edge, len(current.edges)+1)
				copy(newEdges, current.edges)
				newEdges[len(current.edges)] = edge

				queue = append(queue, queueItem{
					nodeID: edge.To,
					path:   newPath,
					edges:  newEdges,
				})
			}
		}
	}

	return nil // No path found
}

// FindAllPaths finds all paths between two nodes up to maxDepth.
func (g *CodeGraph) FindAllPaths(fromID, toID NodeID, maxDepth int) []*PathResult {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	from := g.GetNode(fromID)
	to := g.GetNode(toID)
	if from == nil || to == nil {
		return nil
	}

	var results []*PathResult
	visited := make(map[NodeID]bool)

	var dfs func(current NodeID, path []NodeID, edges []*Edge)
	dfs = func(current NodeID, path []NodeID, edges []*Edge) {
		if len(path) > maxDepth {
			return
		}

		if current == toID {
			pathNodes := make([]*Node, len(path))
			for i, id := range path {
				pathNodes[i] = g.GetNode(id)
			}
			edgesCopy := make([]*Edge, len(edges))
			copy(edgesCopy, edges)

			results = append(results, &PathResult{
				From:   from,
				To:     to,
				Path:   pathNodes,
				Edges:  edgesCopy,
				Length: len(path) - 1,
			})
			return
		}

		visited[current] = true
		defer func() { visited[current] = false }()

		for _, edge := range g.GetOutgoingEdges(current) {
			if !visited[edge.To] {
				dfs(edge.To, append(path, edge.To), append(edges, edge))
			}
		}
	}

	dfs(fromID, []NodeID{fromID}, nil)
	return results
}

// GetDependencyTree returns all nodes reachable from a starting node.
func (g *CodeGraph) GetDependencyTree(startID NodeID, maxDepth int) map[int][]*Node {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	levels := make(map[int][]*Node)
	visited := make(map[NodeID]bool)

	var bfs func(ids []NodeID, depth int)
	bfs = func(ids []NodeID, depth int) {
		if depth > maxDepth || len(ids) == 0 {
			return
		}

		var nextLevel []NodeID
		for _, id := range ids {
			if visited[id] {
				continue
			}
			visited[id] = true

			if node := g.GetNode(id); node != nil {
				levels[depth] = append(levels[depth], node)
			}

			for _, edge := range g.GetOutgoingEdges(id) {
				if !visited[edge.To] {
					nextLevel = append(nextLevel, edge.To)
				}
			}
		}

		if len(nextLevel) > 0 {
			bfs(nextLevel, depth+1)
		}
	}

	bfs([]NodeID{startID}, 0)
	return levels
}

// GetReverseTree returns all nodes that transitively depend on the starting node.
func (g *CodeGraph) GetReverseTree(startID NodeID, maxDepth int) map[int][]*Node {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	levels := make(map[int][]*Node)
	visited := make(map[NodeID]bool)

	var bfs func(ids []NodeID, depth int)
	bfs = func(ids []NodeID, depth int) {
		if depth > maxDepth || len(ids) == 0 {
			return
		}

		var nextLevel []NodeID
		for _, id := range ids {
			if visited[id] {
				continue
			}
			visited[id] = true

			if node := g.GetNode(id); node != nil {
				levels[depth] = append(levels[depth], node)
			}

			for _, edge := range g.GetIncomingEdges(id) {
				if !visited[edge.From] {
					nextLevel = append(nextLevel, edge.From)
				}
			}
		}

		if len(nextLevel) > 0 {
			bfs(nextLevel, depth+1)
		}
	}

	bfs([]NodeID{startID}, 0)
	return levels
}

// Stats returns statistics about the graph.
type Stats struct {
	TotalNodes      int            `json:"total_nodes"`
	TotalEdges      int            `json:"total_edges"`
	NodesByKind     map[string]int `json:"nodes_by_kind"`
	EdgesByKind     map[string]int `json:"edges_by_kind"`
	FileCount       int            `json:"file_count"`
	FunctionCount   int            `json:"function_count"`
	AvgEdgesPerNode float64        `json:"avg_edges_per_node"`
}

// GetStats computes statistics about the graph.
func (g *CodeGraph) GetStats() *Stats {
	stats := &Stats{
		TotalNodes:  len(g.Nodes),
		TotalEdges:  len(g.Edges),
		NodesByKind: make(map[string]int),
		EdgesByKind: make(map[string]int),
	}

	for _, node := range g.Nodes {
		stats.NodesByKind[node.Kind.String()]++
		if node.Kind == KindFile {
			stats.FileCount++
		} else if node.Kind == KindFunction || node.Kind == KindMethod {
			stats.FunctionCount++
		}
	}

	for _, edge := range g.Edges {
		stats.EdgesByKind[edge.Kind.String()]++
	}

	if stats.TotalNodes > 0 {
		stats.AvgEdgesPerNode = float64(stats.TotalEdges) / float64(stats.TotalNodes)
	}

	return stats
}
