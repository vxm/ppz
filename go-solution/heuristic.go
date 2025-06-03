package main

import (
	"fmt"
	"math"
)

// Constraint represents a constraint violation in the board state
type Constraint struct {
	Type           string // Type of constraint
	Severity       int    // How much this constraint affects the solution (1-10)
	Description    string // Human-readable description
	PiecesInvolved []rune // Which pieces are involved
}

// HeuristicAnalysis contains the complete analysis of a board state
type HeuristicAnalysis struct {
	TargetDistance int          // Direct distance for 'b' to reach exit
	Constraints    []Constraint // All constraint violations found
	TotalPenalty   int          // Sum of all constraint penalties
	EstimatedMoves int          // Estimated minimum moves to solve
	IsImpossible   bool         // True if position is unsolvable
}

// analyzeConstraints finds all constraint violations in the current board state
func (g *Game) analyzeConstraints() []Constraint {
	constraints := []Constraint{}

	// Get piece 'b' (target piece)
	targetPiece := g.Pieces['b']
	if targetPiece == nil {
		return constraints
	}

	// Analyze different types of constraints
	constraints = append(constraints, g.analyzePathBlockingConstraints(targetPiece)...)
	constraints = append(constraints, g.analyzeSizeConflictConstraints(targetPiece)...)
	constraints = append(constraints, g.analyzePositionalConstraints(targetPiece)...)
	constraints = append(constraints, g.analyzeSpaceConstraints()...)

	return constraints
}

// analyzePathBlockingConstraints finds pieces directly blocking 'b's path to exit
func (g *Game) analyzePathBlockingConstraints(targetPiece *Piece) []Constraint {
	constraints := []Constraint{}

	// Calculate the column range that 'b' occupies
	minCol, maxCol := 7, 0
	targetRow := 7 // Row where 'b' needs to end up

	for _, pos := range targetPiece.Positions {
		if pos.Col < minCol {
			minCol = pos.Col
		}
		if pos.Col > maxCol {
			maxCol = pos.Col
		}
	}

	// Check each row between current position and target for blocking pieces
	for row := 0; row < targetRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			cell := g.Board[row][col]
			if cell != '0' && cell != 'b' && cell != 'O' {
				// This piece is blocking the path
				severity := calculatePathBlockSeverity(row, targetRow, cell)
				constraints = append(constraints, Constraint{
					Type:           "PATH_BLOCKING",
					Severity:       severity,
					Description:    fmt.Sprintf("Piece '%c' blocks direct path at (%d,%d)", cell, row, col),
					PiecesInvolved: []rune{cell, 'b'},
				})
			}
		}
	}

	return constraints
}

// analyzeSizeConflictConstraints finds pieces that are too large to move around each other
func (g *Game) analyzeSizeConflictConstraints(targetPiece *Piece) []Constraint {
	constraints := []Constraint{}

	// Pieces that are problematic due to size when 'b' needs to move
	problematicPieces := []rune{'a', 'c'} // Large L-shaped pieces

	for _, pieceID := range problematicPieces {
		piece := g.Pieces[pieceID]
		if piece == nil {
			continue
		}

		// Check if this piece creates size conflicts
		if g.createsSizeConflict(targetPiece, piece) {
			severity := calculateSizeConflictSeverity(targetPiece, piece)
			constraints = append(constraints, Constraint{
				Type:           "SIZE_CONFLICT",
				Severity:       severity,
				Description:    fmt.Sprintf("Piece '%c' too large to maneuver around 'b'", pieceID),
				PiecesInvolved: []rune{pieceID, 'b'},
			})
		}
	}

	return constraints
}

// analyzePositionalConstraints finds impossible piece arrangements
func (g *Game) analyzePositionalConstraints(targetPiece *Piece) []Constraint {
	constraints := []Constraint{}

	// Check for pieces that would be "trapped" when 'b' reaches exit
	exitCols := []int{2, 3, 4} // Exit columns

	for pieceID, piece := range g.Pieces {
		if pieceID == 'b' {
			continue
		}

		// Check if piece would be trapped below the exit when 'b' is there
		if g.wouldBeTrappedBelowExit(piece, exitCols) {
			constraints = append(constraints, Constraint{
				Type:           "POSITIONAL_TRAP",
				Severity:       7,
				Description:    fmt.Sprintf("Piece '%c' would be trapped below exit", pieceID),
				PiecesInvolved: []rune{pieceID, 'b'},
			})
		}

		// Check for corner traps and dead-end positions
		if g.isInDeadEndPosition(piece) {
			constraints = append(constraints, Constraint{
				Type:           "DEAD_END",
				Severity:       5,
				Description:    fmt.Sprintf("Piece '%c' in dead-end position", pieceID),
				PiecesInvolved: []rune{pieceID},
			})
		}
	}

	return constraints
}

// analyzeSpaceConstraints finds insufficient space for required movements
func (g *Game) analyzeSpaceConstraints() []Constraint {
	constraints := []Constraint{}

	// Count available empty spaces
	emptySpaces := 0
	for row := 0; row < len(g.Board); row++ {
		for col := 0; col < len(g.Board[row]); col++ {
			if g.Board[row][col] == '0' {
				emptySpaces++
			}
		}
	}

	// Check if there's insufficient maneuvering space
	requiredSpaces := g.calculateRequiredManeuveringSpace()
	if emptySpaces < requiredSpaces {
		constraints = append(constraints, Constraint{
			Type:           "INSUFFICIENT_SPACE",
			Severity:       8,
			Description:    fmt.Sprintf("Only %d empty spaces, need %d for maneuvering", emptySpaces, requiredSpaces),
			PiecesInvolved: []rune{},
		})
	}

	return constraints
}

// Helper functions for constraint analysis

func calculatePathBlockSeverity(blockRow, targetRow int, blockingPiece rune) int {
	// Closer to target = higher severity
	distance := targetRow - blockRow
	baseSeverity := 10 - distance

	// Larger pieces are harder to move
	pieceSize := getPieceTypeSize(blockingPiece)
	return baseSeverity + pieceSize
}

func calculateSizeConflictSeverity(piece1, piece2 *Piece) int {
	// Base severity for size conflicts
	return 6 + len(piece1.Positions) + len(piece2.Positions)
}

func (g *Game) createsSizeConflict(piece1, piece2 *Piece) bool {
	// Check if pieces are both large and in conflicting positions
	if len(piece1.Positions) >= 3 && len(piece2.Positions) >= 3 {
		// Check if they're in the same general area (simplified)
		return g.piecesInSameRegion(piece1, piece2)
	}
	return false
}

func (g *Game) piecesInSameRegion(piece1, piece2 *Piece) bool {
	// Simple region check - if pieces overlap in row/column ranges
	for _, pos1 := range piece1.Positions {
		for _, pos2 := range piece2.Positions {
			if abs(pos1.Row-pos2.Row) <= 2 && abs(pos1.Col-pos2.Col) <= 2 {
				return true
			}
		}
	}
	return false
}

func (g *Game) wouldBeTrappedBelowExit(piece *Piece, exitCols []int) bool {
	// Check if piece is in the exit columns and below row 7
	// Note: Only piece 'b' should ever be in X positions
	for _, pos := range piece.Positions {
		if pos.Row >= 7 {
			for _, col := range exitCols {
				if pos.Col == col {
					// Any piece other than 'b' in exit area is a violation
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) isInDeadEndPosition(piece *Piece) bool {
	// Check if piece is stuck in a corner or against walls with no movement options
	for _, pos := range piece.Positions {
		// Corner positions are often dead ends
		if (pos.Row <= 1 || pos.Row >= 6) && (pos.Col <= 1 || pos.Col >= 5) {
			// Check if piece can actually move from this position
			if !g.pieceCanMove(piece) {
				return true
			}
		}
	}
	return false
}

func (g *Game) pieceCanMove(piece *Piece) bool {
	// Check if piece can move in any direction
	directions := [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} // up, down, left, right

	for _, dir := range directions {
		if g.canMovePiece(piece, dir[0], dir[1]) {
			return true
		}
	}
	return false
}

func (g *Game) calculateRequiredManeuveringSpace() int {
	// Estimate minimum empty spaces needed for complex piece movements
	// This is a heuristic based on piece sizes and board complexity
	totalPieceSize := 0
	for _, piece := range g.Pieces {
		totalPieceSize += len(piece.Positions)
	}

	// Rule of thumb: need at least 20% of piece space as maneuvering room
	return totalPieceSize / 5
}

func getPieceTypeSize(pieceID rune) int {
	// Return typical size penalty for different piece types
	switch pieceID {
	case 'a', 'b', 'c':
		return 3 // Large pieces
	case 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k':
		return 2 // Medium pieces
	case 'l', 'm':
		return 1 // Small pieces
	default:
		return 2
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// calculateHeuristic computes the admissible heuristic value for the current board state
func (g *Game) calculateHeuristic() HeuristicAnalysis {
	analysis := HeuristicAnalysis{}

	// Get target piece
	targetPiece := g.Pieces['b']
	if targetPiece == nil {
		analysis.IsImpossible = true
		analysis.EstimatedMoves = math.MaxInt32
		return analysis
	}

	// Calculate direct distance to goal
	analysis.TargetDistance = g.calculateTargetDistance(targetPiece)

	// Analyze all constraints
	analysis.Constraints = g.analyzeConstraints()

	// Calculate total penalty from constraints
	totalPenalty := 0
	for _, constraint := range analysis.Constraints {
		totalPenalty += constraint.Severity

		// Check for impossible positions
		if constraint.Type == "POSITIONAL_TRAP" && constraint.Severity >= 9 {
			analysis.IsImpossible = true
		}
	}
	analysis.TotalPenalty = totalPenalty

	// Calculate estimated moves (admissible heuristic)
	// Base distance + penalty factor (scaled down to remain admissible)
	analysis.EstimatedMoves = analysis.TargetDistance + (totalPenalty / 3)

	if analysis.IsImpossible {
		analysis.EstimatedMoves = math.MaxInt32
	}

	return analysis
}

func (g *Game) calculateTargetDistance(targetPiece *Piece) int {
	// Calculate Manhattan distance from current position to exit
	// Target positions: (7,2), (7,3), (7,4)

	// Find current center of piece 'b'
	sumRow, sumCol := 0, 0
	for _, pos := range targetPiece.Positions {
		sumRow += pos.Row
		sumCol += pos.Col
	}
	currentRow := sumRow / len(targetPiece.Positions)
	currentCol := sumCol / len(targetPiece.Positions)

	// Target center: row 7, col 3
	targetRow, targetCol := 7, 3

	return abs(targetRow-currentRow) + abs(targetCol-currentCol)
}

// GetHeuristicValue returns just the heuristic value for A* algorithm
func (g *Game) GetHeuristicValue() int {
	analysis := g.calculateHeuristic()
	return analysis.EstimatedMoves
}
