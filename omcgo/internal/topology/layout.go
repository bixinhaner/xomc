package topology

import (
	"context"
	"math"
	"sort"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LayoutAlgorithm defines the interface for topology layout algorithms.
type LayoutAlgorithm interface {
	Apply(ctx context.Context, nodes []TopoNode, edges []TopoEdge) ([]TopoNode, error)
	Name() string
}

// LayoutConfig holds configuration for layout algorithms.
type LayoutConfig struct {
	Type          string  `json:"type"`           // force, hierarchy, circular
	LevelSpacing  float64 `json:"level_spacing"`  // For hierarchy: vertical space between levels
	NodeSpacing   float64 `json:"node_spacing"`   // For hierarchy: horizontal space between nodes
	Iterations    int     `json:"iterations"`     // For force: number of iterations
	Repulsion     float64 `json:"repulsion"`      // For force: repulsive force
	Attraction    float64 `json:"attraction"`     // For force: attractive force
	CenterX       float64 `json:"center_x"`       // Canvas center X
	CenterY       float64 `json:"center_y"`       // Canvas center Y
}

// DefaultLayoutConfig returns default layout configuration.
// Optimized for performance with minimal iterations while maintaining acceptable layout quality.
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		Type:         "hierarchy",
		LevelSpacing: 150,
		NodeSpacing:  120,
		Iterations:   50,   // Reduced from 100 for better performance
		Repulsion:    8000, // Strong repulsion for faster convergence
		Attraction:   0.1,
		CenterX:      1000,
		CenterY:      500,
	}
}

// FastLayoutConfig returns a fast layout configuration for large datasets (>200 nodes).
// Uses minimal iterations for sub-second response times.
func FastLayoutConfig() LayoutConfig {
	return LayoutConfig{
		Type:         "force",
		LevelSpacing: 150,
		NodeSpacing:  120,
		Iterations:   20,    // Minimal iterations for large datasets
		Repulsion:    12000, // Very strong repulsion for quick spread
		Attraction:   0.2,   // Stronger attraction to counter repulsion
		CenterX:      1000,
		CenterY:      500,
	}
}

// InstantLayoutConfig returns an instant configuration for very large datasets (>400 nodes).
// Uses hierarchy layout (O(n)) instead of force (O(n²)) for sub-200ms response.
func InstantLayoutConfig() LayoutConfig {
	return LayoutConfig{
		Type:         "hierarchy", // Use O(n) hierarchy instead of O(n²) force
		LevelSpacing: 150,
		NodeSpacing:  120,
		Iterations:   0, // Not used for hierarchy
		Repulsion:    0,
		Attraction:   0,
		CenterX:      1000,
		CenterY:      500,
	}
}

// NewLayoutAlgorithm creates a layout algorithm based on type.
func NewLayoutAlgorithm(cfg LayoutConfig, logger *zap.Logger) LayoutAlgorithm {
	switch cfg.Type {
	case "force":
		return &ForceLayout{
			Iterations: cfg.Iterations,
			Repulsion:  cfg.Repulsion,
			Attraction: cfg.Attraction,
			CenterX:    cfg.CenterX,
			CenterY:    cfg.CenterY,
			logger:     logger,
		}
	case "circular":
		return &CircularLayout{
			Radius: 400,
			CenterX: cfg.CenterX,
			CenterY: cfg.CenterY,
		}
	case "hierarchy", "tree":
		return &HierarchicalLayout{
			LevelSpacing: cfg.LevelSpacing,
			NodeSpacing:  cfg.NodeSpacing,
			CenterX:      cfg.CenterX,
			CenterY:      50,
		}
	default:
		return &HierarchicalLayout{
			LevelSpacing: cfg.LevelSpacing,
			NodeSpacing:  cfg.NodeSpacing,
			CenterX:      cfg.CenterX,
			CenterY:      50,
		}
	}
}

// HierarchicalLayout implements a hierarchical tree layout.
// This is the default for network topologies as they naturally form a hierarchy.
type HierarchicalLayout struct {
	LevelSpacing float64
	NodeSpacing  float64
	CenterX      float64
	CenterY      float64
}

func (l *HierarchicalLayout) Name() string {
	return "hierarchy"
}

// Apply applies the hierarchical layout algorithm.
func (l *HierarchicalLayout) Apply(ctx context.Context, nodes []TopoNode, edges []TopoEdge) ([]TopoNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	// Build adjacency list for edge traversal
	adj := buildAdjacencyList(edges)

	// Build level structure using BFS from root nodes (nodes with no incoming edges)
	levels := l.buildLevels(nodes, adj)

	// Assign positions based on levels
	result := make([]TopoNode, len(nodes))
	nodeMap := make(map[uuid.UUID]int)
	for i, n := range nodes {
		nodeMap[n.ID] = i
		result[i] = n
	}

	// Calculate canvas dimensions for centering
	maxLevelWidth := 0
	for _, levelNodes := range levels {
		if len(levelNodes) > maxLevelWidth {
			maxLevelWidth = len(levelNodes)
		}
	}

	// Assign positions
	for levelIdx, levelNodes := range levels {
		y := l.CenterY + float64(levelIdx)*l.LevelSpacing
		levelWidth := float64(len(levelNodes))
		startXLevel := l.CenterX - levelWidth*l.NodeSpacing/2

		for i, nodeID := range levelNodes {
			x := startXLevel + float64(i)*l.NodeSpacing
			if idx, ok := nodeMap[nodeID]; ok {
				result[idx].X = x
				result[idx].Y = y
			}
		}
	}

	return result, nil
}

// buildLevels builds the level structure using BFS.
func (l *HierarchicalLayout) buildLevels(nodes []TopoNode, adj map[uuid.UUID][]uuid.UUID) [][]uuid.UUID {
	// Find root nodes (nodes with no incoming edges)
	hasIncoming := make(map[uuid.UUID]bool)
	for _, targets := range adj {
		for _, target := range targets {
			hasIncoming[target] = true
		}
	}

	var roots []uuid.UUID
	nodeMap := make(map[uuid.UUID]TopoNode)
	for _, n := range nodes {
		nodeMap[n.ID] = n
		if !hasIncoming[n.ID] {
			roots = append(roots, n.ID)
		}
	}

	// If no roots found (cycle), use all nodes as roots
	if len(roots) == 0 {
		for _, n := range nodes {
			roots = append(roots, n.ID)
		}
	}

	// BFS to build levels
	levels := [][]uuid.UUID{}
	visited := make(map[uuid.UUID]bool)
	currentLevel := roots

	for len(currentLevel) > 0 {
		levels = append(levels, currentLevel)
		nextLevel := []uuid.UUID{}

		for _, nodeID := range currentLevel {
			if visited[nodeID] {
				continue
			}
			visited[nodeID] = true

			for _, childID := range adj[nodeID] {
				if !visited[childID] {
					nextLevel = append(nextLevel, childID)
				}
			}
		}

		// Add unvisited nodes to next level (handle disconnected components)
		if len(nextLevel) == 0 {
			for _, n := range nodes {
				if !visited[n.ID] {
					nextLevel = append(nextLevel, n.ID)
				}
			}
		}

		currentLevel = nextLevel
	}

	// Add any remaining unvisited nodes
	for _, n := range nodes {
		if !visited[n.ID] {
			levels = append(levels, []uuid.UUID{n.ID})
		}
	}

	return levels
}

// ForceLayout implements a force-directed layout algorithm.
// Useful for exploring complex network relationships.
type ForceLayout struct {
	Iterations int
	Repulsion  float64
	Attraction float64
	CenterX    float64
	CenterY    float64
	logger     *zap.Logger
}

func (l *ForceLayout) Name() string {
	return "force"
}

// Apply applies the force-directed layout algorithm.
func (l *ForceLayout) Apply(ctx context.Context, nodes []TopoNode, edges []TopoEdge) ([]TopoNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	result := make([]TopoNode, len(nodes))
	copy(result, nodes)

	// Initialize positions with a spread pattern to avoid overlap
	// This ensures nodes start in different positions for the force algorithm to work
	hasAnyPosition := false
	for _, n := range result {
		if n.X != 0 || n.Y != 0 {
			hasAnyPosition = true
			break
		}
	}

	// If all nodes are at origin, initialize them in a grid pattern
	if !hasAnyPosition {
		gridSize := int(math.Ceil(math.Sqrt(float64(len(result)))))
		cellSize := 100.0
		startX := l.CenterX - float64(gridSize)*cellSize/2
		startY := l.CenterY - float64(gridSize)*cellSize/2

		for i := range result {
			row := i / gridSize
			col := i % gridSize
			result[i].X = startX + float64(col)*cellSize
			result[i].Y = startY + float64(row)*cellSize
		}
	}

	// Build adjacency list

	// Force-directed iteration
	for iter := 0; iter < l.Iterations; iter++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Calculate displacements
		displacements := make(map[uuid.UUID]struct{ x, y float64 })
		for _, n := range result {
			displacements[n.ID] = struct{ x, y float64 }{}
		}

		// Repulsive forces (all pairs)
		for i, ni := range result {
			dx, dy := 0.0, 0.0
			for j, nj := range result {
				if i == j {
					continue
				}
				vx := ni.X - nj.X
				vy := ni.Y - nj.Y
				dist := math.Sqrt(vx*vx + vy*vy)
				if dist < 1 {
					dist = 1
				}
				force := l.Repulsion / (dist * dist)
				dx += vx / dist * force
				dy += vy / dist * force
			}
			d := displacements[ni.ID]
			d.x += dx
			d.y += dy
			displacements[ni.ID] = d
		}

		// Attractive forces (connected nodes)
		for _, edge := range edges {
			srcIdx := -1
			tgtIdx := -1
			for i, n := range result {
				if n.ID == edge.SourceID {
					srcIdx = i
				}
				if n.ID == edge.TargetID {
					tgtIdx = i
				}
			}
			if srcIdx >= 0 && tgtIdx >= 0 {
				vx := result[tgtIdx].X - result[srcIdx].X
				vy := result[tgtIdx].Y - result[srcIdx].Y
				dist := math.Sqrt(vx*vx + vy*vy)
				if dist > 0.1 {
					fx := vx / dist * l.Attraction * dist
					fy := vy / dist * l.Attraction * dist

					d := displacements[result[srcIdx].ID]
					d.x += fx
					d.y += fy
					displacements[result[srcIdx].ID] = d

					d = displacements[result[tgtIdx].ID]
					d.x -= fx
					d.y -= fy
					displacements[result[tgtIdx].ID] = d
				}
			}
		}

		// Apply displacements with temperature cooling
		temperature := 1.0 - float64(iter)/float64(l.Iterations)
		for i := range result {
			d := displacements[result[i].ID]
			result[i].X += d.x * temperature
			result[i].Y += d.y * temperature

			// Limit to canvas bounds
			result[i].X = clamp(result[i].X, 50, 1950)
			result[i].Y = clamp(result[i].Y, 50, 950)
		}
	}

	return result, nil
}

// CircularLayout arranges nodes in a circle.
type CircularLayout struct {
	Radius float64
	CenterX float64
	CenterY float64
}

func (l *CircularLayout) Name() string {
	return "circular"
}

func (l *CircularLayout) Apply(ctx context.Context, nodes []TopoNode, edges []TopoEdge) ([]TopoNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	result := make([]TopoNode, len(nodes))
	copy(result, nodes)

	// Sort nodes by label for consistent layout
	sort.Slice(result, func(i, j int) bool {
		return result[i].Label < result[j].Label
	})

	// Arrange in circle
	for i := range result {
		angle := 2 * math.Pi * float64(i) / float64(len(result))
		result[i].X = l.CenterX + l.Radius*math.Cos(angle)
		result[i].Y = l.CenterY + l.Radius*math.Sin(angle)
	}

	return result, nil
}

// buildAdjacencyList builds an adjacency list from edges.
func buildAdjacencyList(edges []TopoEdge) map[uuid.UUID][]uuid.UUID {
	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, edge := range edges {
		adj[edge.SourceID] = append(adj[edge.SourceID], edge.TargetID)
		// Also add reverse for undirected traversal
		adj[edge.TargetID] = append(adj[edge.TargetID], edge.SourceID)
	}
	return adj
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
