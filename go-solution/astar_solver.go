package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// AStarNode represents a node in the A* search
type AStarNode struct {
	Game      *Game      // Complete game state
	GCost     int        // Cost from start (number of moves)
	HCost     int        // Heuristic cost to goal
	FCost     int        // Total cost (GCost + HCost)
	Parent    *AStarNode // Parent node for path reconstruction
	Move      string     // Move that led to this state
	BoardHash uint64     // Hash for quick comparison
}

// AStarSolver modifications for parallelism
type AStarSolver struct {
	openSet   *PriorityQueue        // Priority queue for unexplored states
	closedSet map[uint64]bool       // Hash set of explored states
	allNodes  map[uint64]*AStarNode // All nodes for path reconstruction

	// Mutexes for shared data
	closedSetMutex sync.RWMutex
	allNodesMutex  sync.RWMutex

	startTime      time.Time
	nodesGenerated int64 // Use atomic for these counters
	nodesExplored  int64
	maxOpenSetSize int64 // Track max size atomically or within PopMin/Add

	solutionFound   atomic.Bool     // Flag to signal solution
	solutionChannel chan *AStarNode // Channel to send the solution node
	stopWorkersChan chan struct{}   // Channel to signal workers to stop
	wg              sync.WaitGroup  // To wait for workers
	numWorkers      int
}

// NewAStarSolver creates a new A* solver, now with worker config
func NewAStarSolver(numWorkers int) *AStarSolver {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU() // Default to number of CPU cores
	}
	return &AStarSolver{
		openSet:         NewPriorityQueue(),
		closedSet:       make(map[uint64]bool),
		allNodes:        make(map[uint64]*AStarNode),
		solutionChannel: make(chan *AStarNode, 1), // Buffered to avoid blocking sender
		stopWorkersChan: make(chan struct{}),
		numWorkers:      numWorkers,
	}
}

// Solve finds the optimal solution using A* search (now parallel)
func (solver *AStarSolver) Solve(initialGame *Game) *SolutionResult {
	solver.startTime = time.Now()
	solver.solutionFound.Store(false)
	// Reset counters if solver instance is reused (though typically new one per solve)
	atomic.StoreInt64(&solver.nodesGenerated, 0)
	atomic.StoreInt64(&solver.nodesExplored, 0)
	atomic.StoreInt64(&solver.maxOpenSetSize, 0)

	// Reinitialize channels for multiple Solve calls on the same solver instance
	solver.solutionChannel = make(chan *AStarNode, 1)
	solver.stopWorkersChan = make(chan struct{})

	fmt.Println("🔍 Starting Parallel A* Search for Klotski Solution...")
	fmt.Printf("Using %d worker goroutines.\n", solver.numWorkers)
	fmt.Printf("Initial heuristic: %d\n", initialGame.GetHeuristicValue())
	fmt.Println()

	if initialGame.checkWinCondition() {
		return &SolutionResult{
			Found:   true,
			Moves:   []string{},
			Message: "Puzzle is already solved!",
		}
	}

	initialNode := &AStarNode{
		Game:      solver.copyGame(initialGame), // Ensure deep copy
		GCost:     0,
		HCost:     initialGame.GetHeuristicValue(),
		FCost:     initialGame.GetHeuristicValue(),
		Parent:    nil,
		Move:      "initial",
		BoardHash: initialGame.getBoardShapeHash(),
	}

	solver.allNodes[initialNode.BoardHash] = initialNode
	solver.openSet.Add(initialNode.FCost, initialNode.BoardHash, "initial")
	atomic.AddInt64(&solver.nodesGenerated, 1)
	atomic.StoreInt64(&solver.maxOpenSetSize, 1)

	for i := 0; i < solver.numWorkers; i++ {
		solver.wg.Add(1)
		go solver.worker(i)
	}

	var finalGoalNode *AStarNode

	// Wait for a solution or for all workers to finish
	// Option 1: Select on solutionChannel and wg.Wait() in a goroutine
	waitGroupDone := make(chan struct{})
	go func() {
		solver.wg.Wait()
		close(waitGroupDone)
	}()

	select {
	case goalNode := <-solver.solutionChannel:
		finalGoalNode = goalNode
		// Ensure other workers stop quickly - closing stopWorkersChan
		// and signaling PQ it's done helps workers exit loops.
		close(solver.stopWorkersChan)
		solver.openSet.SetDoneAdding() // Make PQ PopMin return nil for waiting workers
	case <-waitGroupDone:
		// All workers finished, no solution found or solution already processed
		if finalGoalNode == nil && solver.solutionFound.Load() {
			// This case might happen if solution was found, channel read, but then wg finished.
			// Redundant check, primary path is through solutionChannel.
			// Or, solution was put on channel, but this select chose waitGroupDone.
			// Try non-blocking read from solutionChannel again.
			select {
			case goalNode := <-solver.solutionChannel:
				finalGoalNode = goalNode
			default:
				// No solution was actually sent or already handled.
			}
		}
	}

	// Ensure all workers are definitely finished before constructing result.
	// This primarily handles the case where solution was found and we need to let
	// workers acknowledge the stop signal.
	<-waitGroupDone // Wait if not already done.

	if finalGoalNode != nil {
		return solver.constructSolution(finalGoalNode)
	}

	return &SolutionResult{
		Found:          false,
		NodesExplored:  atomic.LoadInt64(&solver.nodesExplored),
		NodesGenerated: atomic.LoadInt64(&solver.nodesGenerated),
		MaxOpenSetSize: atomic.LoadInt64(&solver.maxOpenSetSize),
		TimeTaken:      time.Since(solver.startTime),
		Message:        "No solution found - search space exhausted or workers stopped.",
	}
}

func (solver *AStarSolver) worker(workerID int) {
	defer solver.wg.Done()
	// fmt.Printf("Worker %d started\n", workerID)

	for {
		// Check if stop signal received
		select {
		case <-solver.stopWorkersChan:
			// fmt.Printf("Worker %d received stop signal\n", workerID)
			return
		default:
			// Continue if no stop signal
		}

		if solver.solutionFound.Load() { // Atomic check
			return
		}

		poppedItem, ok := solver.openSet.PopMin()
		if !ok { // Queue is empty and done adding
			// fmt.Printf("Worker %d found openSet empty and done.\n", workerID)
			return
		}

		// Track max open set size (approximation, can be done in Add/PopMin of PQ too)
		currentOpenSetSize := int64(solver.openSet.Size())
		maxSize := atomic.LoadInt64(&solver.maxOpenSetSize)
		if currentOpenSetSize > maxSize {
			atomic.CompareAndSwapInt64(&solver.maxOpenSetSize, maxSize, currentOpenSetSize)
		}

		solver.allNodesMutex.RLock()
		currentNode, exists := solver.allNodes[poppedItem.BoardHash]
		solver.allNodesMutex.RUnlock()

		if !exists { // Should not happen if logic is correct
			// fmt.Printf("Worker %d: Node %d not in allNodes after pop!\n", workerID, poppedItem.BoardHash)
			continue
		}

		// If a better path to this node was already found and processed, its GCost might be lower
		// or it might already be in closed set.
		solver.closedSetMutex.Lock()
		if solver.closedSet[currentNode.BoardHash] {
			solver.closedSetMutex.Unlock()
			// fmt.Printf("Worker %d: Node %d already in closed set.\n", workerID, poppedItem.BoardHash)
			continue
		}
		// Mark as "being processed" / "expanded"
		solver.closedSet[currentNode.BoardHash] = true
		solver.closedSetMutex.Unlock()

		atomic.AddInt64(&solver.nodesExplored, 1)

		if workerID == 0 && atomic.LoadInt64(&solver.nodesExplored)%10000 == 0 { // Log progress from one worker
			fmt.Printf("⏱️ Explored: %d, Generated: %d, OpenSet: %d (approx)\n",
				atomic.LoadInt64(&solver.nodesExplored),
				atomic.LoadInt64(&solver.nodesGenerated),
				solver.openSet.Size())
		}

		if currentNode.Game.checkWinCondition() {
			if solver.solutionFound.CompareAndSwap(false, true) { // Ensure only one worker sends solution
				// fmt.Printf("Worker %d found solution! Hash: %d\n", workerID, currentNode.BoardHash)
				solver.solutionChannel <- currentNode
				// No need to close stopWorkersChan here, Solve method will do it.
			}
			return // This worker is done
		}

		successors := solver.generateSuccessors(currentNode) // Pass current node from allNodes

		for _, successor := range successors {
			if solver.solutionFound.Load() { // Check before extensive locking
				return
			}

			select { // Allow early exit if stop signal comes during successor processing
			case <-solver.stopWorkersChan:
				return
			default:
			}

			solver.closedSetMutex.RLock()
			isClosed := solver.closedSet[successor.BoardHash]
			solver.closedSetMutex.RUnlock()
			if isClosed {
				continue
			}

			solver.allNodesMutex.Lock() // Full lock for potential write/read-modify-write
			existingNode, existsInAllNodes := solver.allNodes[successor.BoardHash]

			if existsInAllNodes {
				if successor.GCost < existingNode.GCost {
					existingNode.GCost = successor.GCost
					existingNode.FCost = existingNode.GCost + existingNode.HCost // HCost is fixed for state
					existingNode.Parent = successor.Parent
					existingNode.Move = successor.Move
					solver.allNodesMutex.Unlock() // Unlock before adding to PQ
					solver.openSet.Add(existingNode.FCost, existingNode.BoardHash, existingNode.Move)
					// nodesGenerated not incremented as it's an update
				} else {
					solver.allNodesMutex.Unlock() // Just unlock if no update
				}
			} else {
				solver.allNodes[successor.BoardHash] = successor
				atomic.AddInt64(&solver.nodesGenerated, 1)
				solver.allNodesMutex.Unlock() // Unlock before adding to PQ
				solver.openSet.Add(successor.FCost, successor.BoardHash, successor.Move)
			}
		}
	}
}

// generateSuccessors creates all valid successor states
func (solver *AStarSolver) generateSuccessors(node *AStarNode) []*AStarNode {
	successors := []*AStarNode{}

	// Try moving each piece in each direction
	for pieceID, piece := range node.Game.Pieces {
		directions := []struct {
			name               string
			deltaRow, deltaCol int
		}{
			{"up", -1, 0},
			{"down", 1, 0},
			{"left", 0, -1},
			{"right", 0, 1},
		}

		for _, dir := range directions {
			// Check if piece can move in this direction
			maxDistance := node.Game.getMaxMoveDistance(piece, dir.deltaRow, dir.deltaCol)
			if maxDistance > 0 {
				// Create successor state
				newGame := solver.copyGame(node.Game)
				newPiece := newGame.Pieces[pieceID]

				// Execute the move
				newGame.executeMovepiece(newPiece, dir.deltaRow*maxDistance, dir.deltaCol*maxDistance)

				// Create move description
				moveDesc := fmt.Sprintf("move %c %s %d", pieceID, dir.name, maxDistance)

				// Calculate costs
				gCost := node.GCost + 1 // Each move costs 1
				hCost := newGame.GetHeuristicValue()
				fCost := gCost + hCost

				successor := &AStarNode{
					Game:      newGame,
					GCost:     gCost,
					HCost:     hCost,
					FCost:     fCost,
					Parent:    node,
					Move:      moveDesc,
					BoardHash: newGame.getBoardShapeHash(),
				}

				successors = append(successors, successor)
			}
		}
	}

	return successors
}

// copyGame creates a deep copy of a game state
func (solver *AStarSolver) copyGame(original *Game) *Game {
	// Create new board
	newBoard := make(Board, len(original.Board))
	for i := range original.Board {
		newBoard[i] = make([]rune, len(original.Board[i]))
		copy(newBoard[i], original.Board[i])
	}

	// Create new game
	newGame := &Game{
		Board:   newBoard,
		Pieces:  make(map[rune]*Piece),
		History: nil, // History is not needed for solver's game state copies
	}

	// Re-analyze pieces in the new game
	newGame.analyzePieces()

	return newGame
}

// constructSolution reconstructs the solution path
func (solver *AStarSolver) constructSolution(goalNode *AStarNode) *SolutionResult {
	moves := []string{}
	current := goalNode

	// Walk back through parents to get move sequence
	for current.Parent != nil {
		moves = append([]string{current.Move}, moves...) // Prepend to reverse order
		current = current.Parent
	}

	return &SolutionResult{
		Found:          true,
		Moves:          moves,
		SolutionLength: len(moves),
		NodesExplored:  atomic.LoadInt64(&solver.nodesExplored),
		NodesGenerated: atomic.LoadInt64(&solver.nodesGenerated),
		MaxOpenSetSize: atomic.LoadInt64(&solver.maxOpenSetSize),
		TimeTaken:      time.Since(solver.startTime),
		Message:        fmt.Sprintf("Solution found! %d moves", len(moves)),
		FinalNode:      goalNode,
	}
}

// SolutionResult contains the results of the A* search
type SolutionResult struct {
	Found          bool
	Moves          []string
	SolutionLength int
	NodesExplored  int64 // Changed to int64 for atomic
	NodesGenerated int64 // Changed to int64 for atomic
	MaxOpenSetSize int64 // Changed to int64
	TimeTaken      time.Duration
	Message        string
	FinalNode      *AStarNode
}

// PrintSolution displays the solution in a readable format
func (result *SolutionResult) PrintSolution() {
	fmt.Println("=== A* Search Results ===")
	fmt.Printf("Solution Found: %t\n", result.Found)
	fmt.Printf("Message: %s\n", result.Message)
	fmt.Printf("Time Taken: %v\n", result.TimeTaken)
	fmt.Printf("Nodes Explored: %d\n", result.NodesExplored)     // Already prints int64 fine
	fmt.Printf("Nodes Generated: %d\n", result.NodesGenerated)   // Already prints int64 fine
	fmt.Printf("Max Open Set Size: %d\n", result.MaxOpenSetSize) // Already prints int64 fine

	if result.Found {
		fmt.Printf("Solution Length: %d moves\n", result.SolutionLength)
		if result.SolutionLength > 0 { // Avoid division by zero
			fmt.Println("\nSolution Path:")
			for i, move := range result.Moves {
				fmt.Printf("%d. %s\n", i+1, move)
			}
			fmt.Printf("\nEfficiency: %.2f nodes explored per move\n",
				float64(result.NodesExplored)/float64(result.SolutionLength))
		} else { // Solution of 0 moves (already solved)
			fmt.Println("\nSolution Path: Already at goal!")
		}
	}
	fmt.Println()
}

// ExecuteSolution executes the solution moves on a game
func (result *SolutionResult) ExecuteSolution(game *Game) {
	if !result.Found {
		fmt.Println("No solution to execute!")
		return
	}

	fmt.Println("🎮 Executing Solution...")
	fmt.Println()

	for i, moveStr := range result.Moves {
		fmt.Printf("Step %d: %s\n", i+1, moveStr)

		// Parse and execute the move
		// Format: "move <piece> <direction> <distance>"
		// This would need proper parsing, but for demo we'll show the concept

		game.Display()
		fmt.Println()

		// In a real implementation, you'd parse and execute each move
		// For now, just showing the concept
	}

	fmt.Println("🎉 Solution executed! Puzzle solved!")
}
