package main

import (
	"fmt"
	"time"
)

// GameState represents a single state in the game history
type GameState struct {
	BoardHash  uint64    // Fast hash for quick comparison
	BoardData  Board     // Complete board state
	Move       string    // The move that led to this state
	MoveNumber int       // Sequential move number
	Timestamp  time.Time // When this state was reached
	ParentHash uint64    // Hash of the previous state (for path reconstruction)
}

// GameHistory tracks all visited states during a game session
type GameHistory struct {
	States       []GameState    // Chronological list of all states
	VisitedSet   map[uint64]int // Hash -> index in States slice for O(1) lookup
	CurrentIndex int            // Current position in history
	StartTime    time.Time      // When the game session started
}

// NewGameHistory creates a new game history tracker
func NewGameHistory() *GameHistory {
	return &GameHistory{
		States:       make([]GameState, 0),
		VisitedSet:   make(map[uint64]int),
		CurrentIndex: -1,
		StartTime:    time.Now(),
	}
}

// AddState records a new board state in the history
func (gh *GameHistory) AddState(game *Game, move string) {
	boardHash := game.getBoardShapeHash()

	// Create deep copy of board
	boardCopy := make(Board, len(game.Board))
	for i := range game.Board {
		boardCopy[i] = make([]rune, len(game.Board[i]))
		copy(boardCopy[i], game.Board[i])
	}

	// Get parent hash (previous state)
	var parentHash uint64
	if gh.CurrentIndex >= 0 {
		parentHash = gh.States[gh.CurrentIndex].BoardHash
	}

	// Create new state
	newState := GameState{
		BoardHash:  boardHash,
		BoardData:  boardCopy,
		Move:       move,
		MoveNumber: len(gh.States) + 1,
		Timestamp:  time.Now(),
		ParentHash: parentHash,
	}

	// Add to history
	gh.States = append(gh.States, newState)
	gh.VisitedSet[boardHash] = len(gh.States) - 1
	gh.CurrentIndex = len(gh.States) - 1
}

// HasVisited checks if a board state has been visited before
func (gh *GameHistory) HasVisited(boardHash uint64) bool {
	_, exists := gh.VisitedSet[boardHash]
	return exists
}

// GetVisitedIndex returns the index of a visited state, or -1 if not found
func (gh *GameHistory) GetVisitedIndex(boardHash uint64) int {
	if index, exists := gh.VisitedSet[boardHash]; exists {
		return index
	}
	return -1
}

// DetectCycle checks if the current move creates a cycle
func (gh *GameHistory) DetectCycle(game *Game) *CycleInfo {
	currentHash := game.getBoardShapeHash()

	if prevIndex := gh.GetVisitedIndex(currentHash); prevIndex != -1 {
		// Found a cycle!
		cycleLength := gh.CurrentIndex - prevIndex + 1
		return &CycleInfo{
			StartIndex:  prevIndex,
			EndIndex:    gh.CurrentIndex,
			CycleLength: cycleLength,
			FirstVisit:  gh.States[prevIndex].Timestamp,
			SecondVisit: time.Now(),
		}
	}

	return nil
}

// GetSolutionPath reconstructs the path from start to current state
func (gh *GameHistory) GetSolutionPath() []GameState {
	if gh.CurrentIndex < 0 {
		return []GameState{}
	}

	// Return all states up to current
	path := make([]GameState, gh.CurrentIndex+1)
	copy(path, gh.States[:gh.CurrentIndex+1])
	return path
}

// GetMoveSequence returns just the sequence of moves made
func (gh *GameHistory) GetMoveSequence() []string {
	moves := make([]string, 0, len(gh.States))
	for _, state := range gh.States[:gh.CurrentIndex+1] {
		if state.Move != "initial" && state.Move != "" {
			moves = append(moves, state.Move)
		}
	}
	return moves
}

// CycleInfo contains information about a detected cycle
type CycleInfo struct {
	StartIndex  int       // Index where cycle starts
	EndIndex    int       // Index where cycle ends
	CycleLength int       // Number of moves in the cycle
	FirstVisit  time.Time // When this state was first reached
	SecondVisit time.Time // When this state was reached again
}

// ShowHistory displays the game history
func (gh *GameHistory) ShowHistory() {
	fmt.Println("=== Game History ===")
	fmt.Printf("Session Duration: %v\n", time.Since(gh.StartTime))
	fmt.Printf("Total States: %d\n", len(gh.States))
	fmt.Printf("Unique States: %d\n", len(gh.VisitedSet))
	fmt.Printf("Current Position: %d\n", gh.CurrentIndex+1)
	fmt.Println()

	if len(gh.States) == 0 {
		fmt.Println("No moves made yet.")
		return
	}

	fmt.Println("Move History:")
	for i, state := range gh.States[:gh.CurrentIndex+1] {
		marker := "  "
		if i == gh.CurrentIndex {
			marker = "→ "
		}

		if state.Move == "initial" {
			fmt.Printf("%s%d. Initial state (hash: %d)\n", marker, state.MoveNumber, state.BoardHash)
		} else {
			fmt.Printf("%s%d. %s (hash: %d)\n", marker, state.MoveNumber, state.Move, state.BoardHash)
		}
	}
	fmt.Println()
}

// ShowStatistics displays detailed statistics about the game session
func (gh *GameHistory) ShowStatistics() {
	fmt.Println("=== Game Statistics ===")

	// Basic stats
	fmt.Printf("Total Moves: %d\n", len(gh.States)-1) // -1 for initial state
	fmt.Printf("Unique Board States: %d\n", len(gh.VisitedSet))
	fmt.Printf("Session Duration: %v\n", time.Since(gh.StartTime))

	if len(gh.States) > 1 {
		fmt.Printf("Average Time per Move: %v\n", time.Since(gh.StartTime)/time.Duration(len(gh.States)-1))
	}

	// Cycle detection
	cycles := gh.findAllCycles()
	if len(cycles) > 0 {
		fmt.Printf("Cycles Detected: %d\n", len(cycles))
		for i, cycle := range cycles {
			fmt.Printf("  Cycle %d: moves %d-%d (%d steps)\n",
				i+1, cycle.StartIndex+1, cycle.EndIndex+1, cycle.CycleLength)
		}
	} else {
		fmt.Println("No cycles detected")
	}

	// Move analysis
	moveTypes := make(map[string]int)
	for _, state := range gh.States {
		if state.Move != "initial" && state.Move != "" {
			moveTypes[state.Move]++
		}
	}

	if len(moveTypes) > 0 {
		fmt.Println("\nMove Frequency:")
		for move, count := range moveTypes {
			fmt.Printf("  %s: %d times\n", move, count)
		}
	}

	fmt.Println()
}

// findAllCycles finds all cycles in the game history
func (gh *GameHistory) findAllCycles() []CycleInfo {
	cycles := []CycleInfo{}
	visited := make(map[uint64]int)

	for i, state := range gh.States {
		if prevIndex, exists := visited[state.BoardHash]; exists {
			// Found a cycle
			cycle := CycleInfo{
				StartIndex:  prevIndex,
				EndIndex:    i,
				CycleLength: i - prevIndex,
				FirstVisit:  gh.States[prevIndex].Timestamp,
				SecondVisit: state.Timestamp,
			}
			cycles = append(cycles, cycle)
		}
		visited[state.BoardHash] = i
	}

	return cycles
}

// IsCurrentStateRevisit checks if current state is a revisit
func (gh *GameHistory) IsCurrentStateRevisit() bool {
	if gh.CurrentIndex < 0 {
		return false
	}

	currentHash := gh.States[gh.CurrentIndex].BoardHash

	// Check if this hash appears earlier in history
	for i := 0; i < gh.CurrentIndex; i++ {
		if gh.States[i].BoardHash == currentHash {
			return true
		}
	}

	return false
}

// GetStateAtIndex returns the state at a specific index
func (gh *GameHistory) GetStateAtIndex(index int) *GameState {
	if index < 0 || index >= len(gh.States) {
		return nil
	}
	return &gh.States[index]
}

// Clear resets the game history
func (gh *GameHistory) Clear() {
	gh.States = gh.States[:0]
	gh.VisitedSet = make(map[uint64]int)
	gh.CurrentIndex = -1
	gh.StartTime = time.Now()
}
