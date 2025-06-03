package main

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Board [][]rune

type Position struct {
	Row, Col int
}

type Piece struct {
	ID        rune
	Positions []Position
}

type Game struct {
	Board   Board
	Pieces  map[rune]*Piece
	History *GameHistory
}

// Shape represents the canonical form of a piece shape
type Shape struct {
	Cells []Position // Normalized to start at (0,0)
	Hash  uint64     // Fast hash of the shape
}

// ShapeGroup represents pieces with the same shape
type ShapeGroup struct {
	Shape     Shape
	PieceIDs  []rune
	Positions [][]Position // Positions for each piece of this shape
}

func main() {
	board := Board{
		{'O', 'O', 'O', 'O', 'O', 'O', 'O'},
		{'O', 'a', 'b', 'b', 'b', 'c', 'O'},
		{'O', 'a', 'a', 'd', 'c', 'c', 'O'},
		{'O', 'e', 'e', 'd', 'f', 'f', 'O'},
		{'O', 'e', 'e', 'g', 'f', 'f', 'O'},
		{'O', 'h', 'h', 'g', 'i', 'i', 'O'},
		{'O', 'j', 'j', 'm', 'k', 'k', 'O'},
		{'O', 'l', '0', '0', '0', 'n', 'O'},
		{'O', 'O', 'X', 'X', 'X', 'O', 'O'},
	}

	game := NewGame(board)
	game.Play()
}

func NewGame(board Board) *Game {
	game := &Game{
		Board:   board,
		Pieces:  make(map[rune]*Piece),
		History: NewGameHistory(),
	}
	game.analyzePieces()

	// Record initial state
	game.History.AddState(game, "initial")

	return game
}

// getNormalizedShape returns the canonical shape starting at (0,0)
func getNormalizedShape(positions []Position) Shape {
	if len(positions) == 0 {
		return Shape{}
	}

	// Find the minimum row and column
	minRow, minCol := positions[0].Row, positions[0].Col
	for _, pos := range positions {
		if pos.Row < minRow {
			minRow = pos.Row
		}
		if pos.Col < minCol {
			minCol = pos.Col
		}
	}

	// Normalize positions to start at (0,0)
	normalized := make([]Position, len(positions))
	for i, pos := range positions {
		normalized[i] = Position{
			Row: pos.Row - minRow,
			Col: pos.Col - minCol,
		}
	}

	// Sort positions for consistent ordering
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Row != normalized[j].Row {
			return normalized[i].Row < normalized[j].Row
		}
		return normalized[i].Col < normalized[j].Col
	})

	return Shape{
		Cells: normalized,
		Hash:  hashShape(normalized),
	}
}

// hashShape creates a fast hash of a normalized shape
func hashShape(positions []Position) uint64 {
	h := fnv.New64a()
	for _, pos := range positions {
		h.Write([]byte(fmt.Sprintf("%d,%d;", pos.Row, pos.Col)))
	}
	return h.Sum64()
}

// getBoardShapeGroups groups pieces by their shapes
func (g *Game) getBoardShapeGroups() map[uint64]*ShapeGroup {
	shapeGroups := make(map[uint64]*ShapeGroup)

	for pieceID, piece := range g.Pieces {
		shape := getNormalizedShape(piece.Positions)

		if group, exists := shapeGroups[shape.Hash]; exists {
			group.PieceIDs = append(group.PieceIDs, pieceID)
			group.Positions = append(group.Positions, piece.Positions)
		} else {
			shapeGroups[shape.Hash] = &ShapeGroup{
				Shape:     shape,
				PieceIDs:  []rune{pieceID},
				Positions: [][]Position{piece.Positions},
			}
		}
	}

	return shapeGroups
}

// getBoardShapeHash returns a fast hash of the board state based on piece shapes
func (g *Game) getBoardShapeHash() uint64 {
	shapeGroups := g.getBoardShapeGroups()

	// Create a slice of shape group data for consistent ordering
	type shapeData struct {
		shapeHash uint64
		positions [][]Position
	}

	var shapes []shapeData
	for shapeHash, group := range shapeGroups {
		// Sort positions within each shape group for consistency
		sortedPositions := make([][]Position, len(group.Positions))
		copy(sortedPositions, group.Positions)

		sort.Slice(sortedPositions, func(i, j int) bool {
			return comparePositionSets(sortedPositions[i], sortedPositions[j])
		})

		shapes = append(shapes, shapeData{
			shapeHash: shapeHash,
			positions: sortedPositions,
		})
	}

	// Sort shapes by hash for consistent ordering
	sort.Slice(shapes, func(i, j int) bool {
		return shapes[i].shapeHash < shapes[j].shapeHash
	})

	// Create final hash
	h := fnv.New64a()
	for _, shape := range shapes {
		h.Write([]byte(fmt.Sprintf("S%d:", shape.shapeHash)))
		for _, positions := range shape.positions {
			h.Write([]byte("P"))
			for _, pos := range positions {
				h.Write([]byte(fmt.Sprintf("%d,%d;", pos.Row, pos.Col)))
			}
		}
	}

	return h.Sum64()
}

// getBoardShapeState returns a detailed representation of board state by shapes
func (g *Game) getBoardShapeState() string {
	shapeGroups := g.getBoardShapeGroups()

	var result strings.Builder
	result.WriteString("Board Shape State:\n")

	// Sort shape groups by hash for consistent output
	var sortedHashes []uint64
	for hash := range shapeGroups {
		sortedHashes = append(sortedHashes, hash)
	}
	sort.Slice(sortedHashes, func(i, j int) bool {
		return sortedHashes[i] < sortedHashes[j]
	})

	for _, shapeHash := range sortedHashes {
		group := shapeGroups[shapeHash]
		result.WriteString(fmt.Sprintf("Shape %d (%d cells): pieces %v\n",
			shapeHash, len(group.Shape.Cells), group.PieceIDs))

		for i, positions := range group.Positions {
			result.WriteString(fmt.Sprintf("  %c: %v\n", group.PieceIDs[i], positions))
		}
	}

	result.WriteString(fmt.Sprintf("Board Hash: %d\n", g.getBoardShapeHash()))
	return result.String()
}

// comparePositionSets compares two sets of positions for sorting
func comparePositionSets(a, b []Position) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}

	// Sort both sets and compare lexicographically
	sortedA := make([]Position, len(a))
	sortedB := make([]Position, len(b))
	copy(sortedA, a)
	copy(sortedB, b)

	sort.Slice(sortedA, func(i, j int) bool {
		if sortedA[i].Row != sortedA[j].Row {
			return sortedA[i].Row < sortedA[j].Row
		}
		return sortedA[i].Col < sortedA[j].Col
	})

	sort.Slice(sortedB, func(i, j int) bool {
		if sortedB[i].Row != sortedB[j].Row {
			return sortedB[i].Row < sortedB[j].Row
		}
		return sortedB[i].Col < sortedB[j].Col
	})

	for i := 0; i < len(sortedA); i++ {
		if sortedA[i].Row != sortedB[i].Row {
			return sortedA[i].Row < sortedB[i].Row
		}
		if sortedA[i].Col != sortedB[i].Col {
			return sortedA[i].Col < sortedB[i].Col
		}
	}

	return false // They are equal
}

func (g *Game) analyzePieces() {
	visited := make(map[Position]bool)

	for row := 0; row < len(g.Board); row++ {
		for col := 0; col < len(g.Board[row]); col++ {
			pos := Position{row, col}
			cell := g.Board[row][col]

			if visited[pos] || cell == 'O' || cell == '0' || cell == 'X' {
				continue
			}

			piece := &Piece{
				ID:        cell,
				Positions: []Position{},
			}

			g.floodFill(row, col, cell, piece, visited)
			g.Pieces[cell] = piece
		}
	}
}

func (g *Game) floodFill(row, col int, target rune, piece *Piece,
	visited map[Position]bool) {
	if row < 0 || row >= len(g.Board) || col < 0 ||
		col >= len(g.Board[0]) {
		return
	}

	pos := Position{row, col}
	if visited[pos] || g.Board[row][col] != target {
		return
	}

	visited[pos] = true
	piece.Positions = append(piece.Positions, pos)

	g.floodFill(row+1, col, target, piece, visited)
	g.floodFill(row-1, col, target, piece, visited)
	g.floodFill(row, col+1, target, piece, visited)
	g.floodFill(row, col-1, target, piece, visited)
}

func (g *Game) Play() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== Interactive Klotski Game ===")
	fmt.Println("GOAL: Move piece 'b' to exit through the XXX area!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  move <piece> <direction> [distance]")
	fmt.Println("  show - display board")
	fmt.Println("  pieces - list all pieces")
	fmt.Println("  shapes - analyze piece shapes and board hash")
	fmt.Println("  hash - show current board hash")
	fmt.Println("  queue - demonstrate priority queue")
	fmt.Println("  heuristic - analyze constraints and calculate heuristic")
	fmt.Println("  solver - demonstrate A* solver framework")
	fmt.Println("  complexity - analyze search space complexity")
	fmt.Println("  history - show game history and moves")
	fmt.Println("  stats - show game statistics and cycles")
	fmt.Println("  solve - run complete A* solver to find solution")
	fmt.Println("  quit - exit game")
	fmt.Println()
	fmt.Println("Note: Distance is optional. If not specified, moves as far as possible.")
	fmt.Println()

	for {
		g.Display()

		if g.checkWinCondition() {
			fmt.Println("🎉 CONGRATULATIONS! 🎉")
			fmt.Println("You solved the Klotski puzzle!")
			fmt.Println("Piece 'b' has successfully escaped through the exit!")
			fmt.Println()
			fmt.Print("Play again? (y/n): ")
			if scanner.Scan() {
				response := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if response == "y" || response == "yes" {
					g.resetGame()
					continue
				}
			}
			fmt.Println("Thanks for playing!")
			return
		}

		fmt.Print("Enter command: ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := strings.ToLower(parts[0])

		switch command {
		case "quit", "exit", "q":
			fmt.Println("Thanks for playing!")
			return
		case "show", "s":
			continue
		case "pieces", "p":
			g.showPieces()
		case "shapes":
			fmt.Print(g.getBoardShapeState())
		case "hash", "h":
			fmt.Printf("Current board hash: %d\n\n", g.getBoardShapeHash())
		case "queue":
			g.demonstratePriorityQueue()
		case "heuristic":
			g.analyzeConstraintsAndCalculateHeuristic()
		case "solver":
			g.demonstrateAStarSolver()
		case "complexity":
			g.analyzeSearchSpaceComplexity()
		case "history":
			g.showHistory()
		case "stats":
			g.showStatistics()
		case "solve":
			g.runCompleteAStarSolver()
		case "move", "m":
			if len(parts) < 3 || len(parts) > 4 {
				fmt.Println("Usage: move <piece> <direction> [distance]")
				fmt.Println("Directions: up, down, left, right")
				fmt.Println("Distance: optional number (default: max possible)")
				continue
			}
			pieceID := rune(parts[1][0])
			direction := strings.ToLower(parts[2])

			distance := -1 // -1 means move as far as possible
			if len(parts) == 4 {
				if d, err := strconv.Atoi(parts[3]); err == nil && d > 0 {
					distance = d
				} else {
					fmt.Printf("Invalid distance '%s'. Must be a positive number.\n", parts[3])
					continue
				}
			}

			g.movePiece(pieceID, direction, distance)
		default:
			fmt.Println("Unknown command. Try 'move', 'show', 'pieces', 'shapes', 'hash', 'queue', 'heuristic', 'solver', 'complexity', 'history', 'stats', 'solve', or 'quit'")
		}
	}
}

func (g *Game) checkWinCondition() bool {
	piece, exists := g.Pieces['b']
	if !exists {
		return false
	}

	// Check if piece 'b' occupies the XXX exit area (row 8, cols 2,3,4)
	exitPositions := []Position{
		{8, 2}, {8, 3}, {8, 4},
	}

	if len(piece.Positions) != 3 {
		return false
	}

	// Create a map of piece positions for quick lookup
	piecePositions := make(map[Position]bool)
	for _, pos := range piece.Positions {
		piecePositions[pos] = true
	}

	// Check if all exit positions are occupied by piece 'b'
	for _, exitPos := range exitPositions {
		if !piecePositions[exitPos] {
			return false
		}
	}

	return true
}

func (g *Game) resetGame() {
	// Reset to initial board state
	initialBoard := Board{
		{'O', 'O', 'O', 'O', 'O', 'O', 'O'},
		{'O', 'a', 'b', 'b', 'b', 'c', 'O'},
		{'O', 'a', 'a', 'd', 'c', 'c', 'O'},
		{'O', 'e', 'e', 'd', 'g', 'g', 'O'},
		{'O', 'j', 'j', 'h', 'f', 'f', 'O'},
		{'O', 'i', 'i', 'h', 'k', 'k', 'O'},
		{'O', 'l', '0', '0', '0', 'm', 'O'},
		{'O', 'O', 'X', 'X', 'X', 'O', 'O'},
	}

	g.Board = initialBoard
	g.Pieces = make(map[rune]*Piece)
	g.analyzePieces()

	// Reset game history
	g.History.Clear()
	g.History.AddState(g, "initial")

	fmt.Println("Game reset! Good luck!")
	fmt.Println()
}

func (g *Game) Display() {
	fmt.Println("\nKlotski Game Board:")
	fmt.Println()
	for _, row := range g.Board {
		for _, cell := range row {
			fmt.Printf("%c ", cell)
		}
		fmt.Println()
	}
	fmt.Println()
}

func (g *Game) showPieces() {
	fmt.Println("Available pieces:")
	for id, piece := range g.Pieces {
		if id == 'b' {
			fmt.Printf("  %c: %d cells (TARGET PIECE - must reach XXX exit!)\n",
				id, len(piece.Positions))
		} else {
			fmt.Printf("  %c: %d cells\n", id, len(piece.Positions))
		}
	}
	fmt.Println()
}

func (g *Game) movePiece(pieceID rune, direction string, requestedDistance int) {
	piece, exists := g.Pieces[pieceID]
	if !exists {
		fmt.Printf("Piece '%c' not found\n", pieceID)
		return
	}

	var deltaRow, deltaCol int
	switch direction {
	case "up", "u":
		deltaRow, deltaCol = -1, 0
	case "down", "d":
		deltaRow, deltaCol = 1, 0
	case "left", "l":
		deltaRow, deltaCol = 0, -1
	case "right", "r":
		deltaRow, deltaCol = 0, 1
	default:
		fmt.Printf("Invalid direction '%s'. Use: up, down, left, right\n",
			direction)
		return
	}

	maxDistance := g.getMaxMoveDistance(piece, deltaRow, deltaCol)
	if maxDistance == 0 {
		fmt.Printf("Cannot move piece '%c' %s - blocked!\n",
			pieceID, direction)
		return
	}

	actualDistance := maxDistance
	if requestedDistance > 0 && requestedDistance < maxDistance {
		actualDistance = requestedDistance
	}

	g.executeMovepiece(piece, deltaRow*actualDistance, deltaCol*actualDistance)

	// Create move description for history
	moveDesc := fmt.Sprintf("move %c %s %d", pieceID, direction, actualDistance)

	// Record the move in history
	g.History.AddState(g, moveDesc)

	// Check for cycles
	if cycle := g.History.DetectCycle(g); cycle != nil {
		fmt.Printf("⚠️  Cycle detected! You've returned to a previous state from move %d\n",
			cycle.StartIndex+1)
	}

	if requestedDistance == -1 {
		fmt.Printf("Moved piece '%c' %s %d spaces (max possible)\n",
			pieceID, direction, actualDistance)
	} else if actualDistance == requestedDistance {
		fmt.Printf("Moved piece '%c' %s %d spaces\n",
			pieceID, direction, actualDistance)
	} else {
		fmt.Printf("Moved piece '%c' %s %d spaces (max possible, requested %d)\n",
			pieceID, direction, actualDistance, requestedDistance)
	}
}

func (g *Game) getMaxMoveDistance(piece *Piece, deltaRow, deltaCol int) int {
	distance := 0

	for {
		distance++
		canMove := true

		for _, pos := range piece.Positions {
			newRow := pos.Row + deltaRow*distance
			newCol := pos.Col + deltaCol*distance

			if newRow < 0 || newRow >= len(g.Board) ||
				newCol < 0 || newCol >= len(g.Board[0]) {
				canMove = false
				break
			}

			targetCell := g.Board[newRow][newCol]
			if targetCell == 'O' {
				canMove = false
				break
			}

			// Only piece 'b' can move into 'X' cells (exit area)
			if targetCell == 'X' && piece.ID != 'b' {
				canMove = false
				break
			}

			if targetCell != '0' && targetCell != 'X' && targetCell != piece.ID {
				canMove = false
				break
			}
		}

		if !canMove {
			return distance - 1
		}
	}
}

func (g *Game) canMovePiece(piece *Piece, deltaRow, deltaCol int) bool {
	for _, pos := range piece.Positions {
		newRow := pos.Row + deltaRow
		newCol := pos.Col + deltaCol

		if newRow < 0 || newRow >= len(g.Board) ||
			newCol < 0 || newCol >= len(g.Board[0]) {
			return false
		}

		targetCell := g.Board[newRow][newCol]
		if targetCell == 'O' {
			return false
		}

		// Only piece 'b' can move into 'X' cells (exit area)
		if targetCell == 'X' && piece.ID != 'b' {
			return false
		}

		if targetCell != '0' && targetCell != 'X' && targetCell != piece.ID {
			return false
		}
	}
	return true
}

func (g *Game) executeMovepiece(piece *Piece, deltaRow, deltaCol int) {
	for _, pos := range piece.Positions {
		g.Board[pos.Row][pos.Col] = '0'
	}

	newPositions := make([]Position, len(piece.Positions))
	for i, pos := range piece.Positions {
		newPos := Position{pos.Row + deltaRow, pos.Col + deltaCol}
		newPositions[i] = newPos

		// If moving into exit area, keep the 'X' visual but track piece position
		if g.Board[newPos.Row][newPos.Col] == 'X' {
			g.Board[newPos.Row][newPos.Col] = piece.ID
		} else {
			g.Board[newPos.Row][newPos.Col] = piece.ID
		}
	}

	piece.Positions = newPositions
}

func (g *Game) demonstratePriorityQueue() {
	fmt.Println("=== Priority Queue Demonstration for Klotski Solver ===")
	fmt.Println()

	// Create a priority queue
	pq := NewPriorityQueue()

	// Get current board hash
	currentHash := g.getBoardShapeHash()

	// Add current state as initial state (priority 0 - highest)
	pq.Add(0, currentHash, "initial_state")
	fmt.Printf("Added initial state with priority 0, hash: %d\n", currentHash)

	// Simulate some possible moves and their estimated costs
	// In a real A* implementation, priority = g(cost) + h(heuristic)
	moves := []struct {
		piece     rune
		direction string
		priority  int
		desc      string
	}{
		{'l', "right", 5, "move piece 'l' right - low cost"},
		{'m', "left", 8, "move piece 'm' left - medium cost"},
		{'a', "down", 12, "move piece 'a' down - higher cost"},
		{'b', "down", 3, "move piece 'b' down - very promising!"},
		{'c', "left", 10, "move piece 'c' left - medium-high cost"},
	}

	// Add simulated states to queue
	for i, move := range moves {
		// Generate a fake hash for demonstration
		fakeHash := currentHash + uint64(i*1000+move.priority)
		pq.Add(move.priority, fakeHash, fmt.Sprintf("move_%c_%s", move.piece, move.direction))
		fmt.Printf("Added %s (priority %d, hash: %d)\n",
			move.desc, move.priority, fakeHash)
	}

	fmt.Printf("\nQueue size: %d items\n", pq.Size())
	fmt.Println("\nProcessing states in priority order (A* algorithm):")
	fmt.Println("(Lower priority = higher importance)")

	// Process all states in order
	step := 1
	for !pq.IsEmpty() {
		item, ok := pq.PopMin()
		if !ok {
			break // Queue is empty and closed
		}
		fmt.Printf("%d. Processing state: %s (priority %d, hash %d)\n",
			step, item.BoardData, item.Priority, item.BoardHash)
		step++

		// In a real solver, you would:
		// 1. Check if this is the goal state (piece 'b' in exit)
		// 2. Generate all possible moves from this state
		// 3. Calculate costs and add promising states to queue
		// 4. Use visited set to avoid cycles
	}

	fmt.Println("\nThis demonstrates how a Klotski solver would use the priority queue:")
	fmt.Println("- States with lower priority values are processed first")
	fmt.Println("- Thread-safe operations allow multiple solver threads")
	fmt.Println("- Board hashes enable fast state comparison and cycle detection")
	fmt.Println("- Efficient O(log n) insertion and extraction")
	fmt.Println()
}

func (g *Game) analyzeConstraintsAndCalculateHeuristic() {
	fmt.Println("=== Heuristic Analysis for Klotski Solver ===")
	fmt.Println()

	// Perform complete heuristic analysis
	analysis := g.calculateHeuristic()

	// Display target distance
	fmt.Printf("Target Distance (Manhattan): %d moves\n", analysis.TargetDistance)
	fmt.Printf("Target piece 'b' needs to reach exit at row 7, columns 2-4\n")
	fmt.Println()

	// Display constraints found
	if len(analysis.Constraints) == 0 {
		fmt.Println("✅ No constraints detected - this is a very promising position!")
	} else {
		fmt.Printf("Constraints Detected: %d\n", len(analysis.Constraints))
		fmt.Println("----------------------------------------")

		// Group constraints by type
		constraintTypes := make(map[string][]Constraint)
		for _, constraint := range analysis.Constraints {
			constraintTypes[constraint.Type] = append(constraintTypes[constraint.Type], constraint)
		}

		// Display each type
		for constraintType, constraints := range constraintTypes {
			fmt.Printf("\n🔍 %s (%d found):\n", formatConstraintType(constraintType), len(constraints))
			for _, constraint := range constraints {
				fmt.Printf("  • %s (severity: %d)\n", constraint.Description, constraint.Severity)
				if len(constraint.PiecesInvolved) > 0 {
					fmt.Printf("    Pieces involved: %v\n", constraint.PiecesInvolved)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Heuristic Summary ===")
	fmt.Printf("Total Constraint Penalty: %d\n", analysis.TotalPenalty)
	fmt.Printf("Estimated Moves to Solution: %d\n", analysis.EstimatedMoves)

	if analysis.IsImpossible {
		fmt.Println("❌ IMPOSSIBLE POSITION DETECTED!")
		fmt.Println("This board state may be unsolvable.")
	} else if analysis.EstimatedMoves <= 10 {
		fmt.Println("🎯 VERY PROMISING! Close to solution.")
	} else if analysis.EstimatedMoves <= 20 {
		fmt.Println("👍 GOOD POSITION! Reasonable path to solution.")
	} else if analysis.EstimatedMoves <= 40 {
		fmt.Println("⚠️  CHALLENGING! Will require many moves.")
	} else {
		fmt.Println("🚨 VERY DIFFICULT! Consider alternative approaches.")
	}

	fmt.Println()
	fmt.Println("This heuristic guides the A* solver to prioritize more promising states.")
	fmt.Println("Lower heuristic values mean higher priority in the search queue.")
	fmt.Println()
}

func formatConstraintType(constraintType string) string {
	switch constraintType {
	case "PATH_BLOCKING":
		return "Path Blocking Constraints"
	case "SIZE_CONFLICT":
		return "Size Conflict Constraints"
	case "POSITIONAL_TRAP":
		return "Positional Trap Constraints"
	case "DEAD_END":
		return "Dead End Constraints"
	case "INSUFFICIENT_SPACE":
		return "Insufficient Space Constraints"
	default:
		return constraintType
	}
}

func (g *Game) demonstrateAStarSolver() {
	solver := NewKlotskiSolver()
	solver.SolveDemo(g)
}

func (g *Game) analyzeSearchSpaceComplexity() {
	g.EstimateComplexity()
}

func (g *Game) showHistory() {
	g.History.ShowHistory()
}

func (g *Game) showStatistics() {
	g.History.ShowStatistics()
}

// runCompleteAStarSolver runs the complete A* solver
func (g *Game) runCompleteAStarSolver() {
	fmt.Println("🚀 Running Complete A* Solver...")
	fmt.Println("This may take some time depending on puzzle complexity.")
	fmt.Println()

	// Create solver and run (use 0 for default number of workers)
	solver := NewAStarSolver(0)
	result := solver.Solve(g)

	// Display results
	result.PrintSolution()

	if result.Found {
		fmt.Print("Would you like to see the solution executed step by step? (y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response == "y" || response == "yes" {
				// Create a deep copy of the board g had when 'solve' was invoked
				boardCopyForSolution := make(Board, len(g.Board))
				for i := range g.Board {
					boardCopyForSolution[i] = make([]rune, len(g.Board[i]))
					copy(boardCopyForSolution[i], g.Board[i])
				}
				solutionGame := NewGame(boardCopyForSolution) // Pass the deep copy

				g.executeSolutionStepByStep(solutionGame, result.Moves)
			}
		}
	}
}

// executeSolutionStepByStep executes the solution moves with user interaction
func (g *Game) executeSolutionStepByStep(game *Game, moves []string) {
	fmt.Println("🎬 Step-by-step Solution Execution")
	fmt.Println("Press Enter after each step to continue...")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Initial State:")
	game.Display()
	fmt.Print("Press Enter to start...")
	scanner.Scan()

	for i, moveStr := range moves {
		fmt.Printf("\n=== Step %d/%d ===\n", i+1, len(moves))
		fmt.Printf("Move: %s\n", moveStr)

		// Parse the move string: "move <piece> <direction> <distance>"
		parts := strings.Fields(moveStr)
		if len(parts) >= 4 {
			pieceID := rune(parts[1][0])
			direction := parts[2]
			distance := 1
			if len(parts) > 3 {
				if d, err := strconv.Atoi(parts[3]); err == nil {
					distance = d
				}
			}

			// Execute the move
			game.movePiece(pieceID, direction, distance)
		}

		game.Display()

		if game.checkWinCondition() {
			fmt.Println("🎉 PUZZLE SOLVED! 🎉")
			break
		}

		if i < len(moves)-1 {
			fmt.Print("Press Enter for next step...")
			scanner.Scan()
		}
	}

	fmt.Println("\n✅ Solution execution complete!")
	fmt.Printf("Total moves executed: %d\n", len(moves))
	fmt.Println()
}
