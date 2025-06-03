package main

import (
	"fmt"
	"time"
)

// SearchNode represents a node in the simplified A* search demo
type SearchNode struct {
	Game   *Game       // Simplified game state (could be just hash or full board)
	GCost  int         // Cost from start
	HCost  int         // Heuristic cost
	FCost  int         // GCost + HCost
	Parent *SearchNode // Parent node
	Move   string      // Move that led to this state
	// No BoardHash here directly, it's part of the QueueItem
}

// KlotskiSolver provides a simplified A* solving framework demonstration.
// This is separate from the main parallel AStarSolver.
type KlotskiSolver struct {
	openSet       *PriorityQueue         // Min-priority queue of states to visit
	closedSet     map[uint64]bool        // Set of visited states (by board hash)
	allNodes      map[uint64]*SearchNode // Maps board hash to SearchNode for path reconstruction
	startTime     time.Time
	nodesExplored int
}

// NewKlotskiSolver creates a new solver instance for demo.
func NewKlotskiSolver() *KlotskiSolver {
	return &KlotskiSolver{
		openSet:   NewPriorityQueue(),
		closedSet: make(map[uint64]bool),
		allNodes:  make(map[uint64]*SearchNode),
	}
}

// SolveDemo demonstrates the A* components with a simplified loop.
func (ks *KlotskiSolver) SolveDemo(initialGame *Game) {
	fmt.Println("=== A* Solver Framework Demonstration (Simplified) ===")
	ks.startTime = time.Now()
	ks.nodesExplored = 0

	initialHash := initialGame.getBoardShapeHash()
	initialHeuristic := initialGame.GetHeuristicValue()

	startNode := &SearchNode{
		Game:  initialGame, // In real A*, this would be a copy
		GCost: 0,
		HCost: initialHeuristic,
		FCost: initialHeuristic,
		Move:  "initial",
	}

	ks.openSet.Add(startNode.FCost, initialHash, startNode.Move) // Store move string as BoardData for demo
	ks.allNodes[initialHash] = startNode

	fmt.Printf("Initial state added to open set (F: %d, H: %d, Hash: %d)\n",
		startNode.FCost, startNode.HCost, initialHash)

	maxIterations := 10 // Limit for demo purposes
	iteration := 0

	for iteration < maxIterations && !ks.openSet.IsEmpty() {
		iteration++

		currentItem, ok := ks.openSet.PopMin()
		if !ok {
			fmt.Println("Open set empty or closed, stopping demo.")
			break
		}

		// Retrieve the full node using the hash from the popped QueueItem
		currentNode, nodeExists := ks.allNodes[currentItem.BoardHash]
		if !nodeExists {
			fmt.Printf("Warning: Popped item with hash %d not found in allNodes. Skipping.\n", currentItem.BoardHash)
			continue
		}

		ks.nodesExplored++

		fmt.Printf("\nIteration %d: Popped '%s' (F: %d, G: %d, H: %d, Hash: %d)\n",
			iteration, currentNode.Move, currentNode.FCost, currentNode.GCost, currentNode.HCost, currentItem.BoardHash)

		if ks.closedSet[currentItem.BoardHash] {
			fmt.Printf("State %d already visited. Skipping.\n", currentItem.BoardHash)
			continue
		}
		ks.closedSet[currentItem.BoardHash] = true

		// Check for win condition (simplified for demo)
		if currentNode.HCost == 0 { // Assuming HCost = 0 means goal
			fmt.Println("🎉 Goal state reached (heuristic is 0)! Path reconstruction would start here.")
			// Path reconstruction would trace Parent pointers from currentNode
			break
		}

		// Simulate generating successor states
		successorStates := ks.generateSimulatedSuccessors(currentNode, currentItem.BoardHash, initialGame)

		for _, successor := range successorStates {
			successorHash := successor.Game.getBoardShapeHash() // Assuming Game has method to get hash

			if ks.closedSet[successorHash] {
				continue
			}

			// Add to open set and allNodes
			if existingNode, exists := ks.allNodes[successorHash]; !exists || successor.GCost < existingNode.GCost {
				ks.allNodes[successorHash] = successor
				ks.openSet.Add(successor.FCost, successorHash, successor.Move)
				fmt.Printf("  Added successor '%s' to open set (F: %d, G: %d, H: %d, Hash: %d)\n",
					successor.Move, successor.FCost, successor.GCost, successor.HCost, successorHash)
			}
		}
	}

	fmt.Printf("\nDemo finished after %d iterations. Nodes explored: %d. Time: %v\n",
		iteration, ks.nodesExplored, time.Since(ks.startTime))
	fmt.Println("This simplified demo illustrates A* component interactions (PriorityQueue, Heuristic, State Tracking). The main 'solve' command uses the full parallel A* solver.")
}

// generateSimulatedSuccessors creates a few dummy successor states for demonstration.
func (ks *KlotskiSolver) generateSimulatedSuccessors(parent *SearchNode, parentHash uint64, game *Game) []*SearchNode {
	successors := []*SearchNode{}

	// Simulate a few moves with arbitrary costs
	moves := []struct {
		desc          string
		dGCost        int    // Delta GCost from parent
		dHCost        int    // Change in Heuristic relative to parent
		newHashOffset uint64 // To simulate different states
	}{
		{"move_A_right", 1, -1, parentHash + 101},
		{"move_B_down", 1, 2, parentHash + 202},
		{"move_C_left", 1, 0, parentHash + 303},
	}

	for _, m := range moves {
		// In a real A*, game state would be copied and move applied
		// For demo, we just adjust costs and use a new hash
		newGCost := parent.GCost + m.dGCost
		newHCost := parent.HCost + m.dHCost // Simplified heuristic change
		if newHCost < 0 {
			newHCost = 0
		}

		// Create a placeholder game state for the successor for demo purposes
		// In a real A*, this would be a result of a move.
		successorGame := game // For demo, just point to the initial game
		// If we needed unique hashes based on actual moves, we'd need to create distinct Game objects.
		// For this demo, newHashOffset simulates unique states.

		successors = append(successors, &SearchNode{
			Game:   successorGame, // Placeholder
			GCost:  newGCost,
			HCost:  newHCost,
			FCost:  newGCost + newHCost,
			Parent: parent,
			Move:   m.desc,
			// Successor's hash is simulated by newHashOffset for demo purposes
			// In real A*, it'd be newGame.getBoardShapeHash()
		})
	}
	return successors
}

// EstimateComplexity analyzes the search space complexity
func (g *Game) EstimateComplexity() {
	fmt.Println("=== Search Space Complexity Analysis ===")
	fmt.Println()

	// Count moveable pieces
	moveablePieces := 0
	for _, piece := range g.Pieces {
		if g.pieceCanMove(piece) {
			moveablePieces++
		}
	}

	// Estimate branching factor
	avgMovesPerPiece := 2.5 // Average directions a piece can move
	branchingFactor := float64(moveablePieces) * avgMovesPerPiece

	// Current heuristic gives us depth estimate
	heuristic := g.GetHeuristicValue()

	// Estimate search space size
	searchSpaceEstimate := 1.0
	for i := 0; i < heuristic && i < 20; i++ { // Cap at depth 20 for calculation
		searchSpaceEstimate *= branchingFactor
	}

	fmt.Printf("Moveable Pieces: %d\n", moveablePieces)
	fmt.Printf("Estimated Branching Factor: %.1f\n", branchingFactor)
	fmt.Printf("Heuristic Depth Estimate: %d\n", heuristic)

	if searchSpaceEstimate > 1e12 {
		fmt.Printf("Search Space Estimate: %.2e states (EXTREMELY LARGE)\n", searchSpaceEstimate)
		fmt.Println("🚨 This puzzle may be computationally intractable with brute force!")
	} else if searchSpaceEstimate > 1e9 {
		fmt.Printf("Search Space Estimate: %.2e states (VERY LARGE)\n", searchSpaceEstimate)
		fmt.Println("⚠️  This puzzle will require significant computation time.")
	} else if searchSpaceEstimate > 1e6 {
		fmt.Printf("Search Space Estimate: %.2e states (LARGE)\n", searchSpaceEstimate)
		fmt.Println("👍 This puzzle is computationally feasible with A*.")
	} else {
		fmt.Printf("Search Space Estimate: %.0f states (MANAGEABLE)\n", searchSpaceEstimate)
		fmt.Println("🎯 This puzzle should solve quickly with A*!")
	}

	fmt.Println()
	fmt.Println("The heuristic function is crucial for pruning this search space.")
	fmt.Println("Good heuristics can reduce search by orders of magnitude!")
}
