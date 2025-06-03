# Go Klotski

An interactive Go implementation of the classic Klotski sliding puzzle game, featuring multi-space movement, advanced state analysis, and an AI solver.

## Overview

This project implements a fully interactive Klotski sliding puzzle game where players must move pieces strategically to help piece 'b' escape through the exit. The board is represented as a 2D grid with different characters representing various game pieces and elements:

- `O` - Walls/boundaries
- `a-m` - Different game pieces (various shapes and sizes)
- `0` - Empty spaces
- `X` - Exit area (goal for piece 'b')

## 🎯 Game Objective

**GOAL: Move piece 'b' (the 3-cell horizontal piece) to exit through the XXX area at the bottom of the board!**

This is the classic Klotski puzzle challenge - strategically move the other pieces to clear a path for piece 'b' to reach the bottom exit. When piece 'b' occupies all three XXX positions, you win!

## Features

- **Classic Puzzle Goal**: Authentic Klotski escape puzzle mechanics.
- **Win Condition Detection**: Automatic victory detection when piece 'b' reaches the exit.
- **Interactive Gameplay**: Move pieces using simple commands.
- **Multi-Space Movement**: Move pieces multiple spaces at once up to available empty spaces.
- **Smart Distance Control**: Automatic maximum distance detection or specify exact distance.
- **Move Validation**: Only legal moves that don't cause collisions are allowed.
- **Piece Recognition**: Automatically detects multi-cell pieces and their shapes.
- **Real-time Board Updates**: See changes immediately after each move.
- **Play Again Option**: Reset and replay when puzzle is solved.
- **Shape-Based Hashing**: Fast FNV-1a hashing for board states based on piece shapes and relative positions, enabling O(1) lookup for visited states.
- **Advanced Heuristic Function**: Sophisticated heuristic analysis considering path blocking, size conflicts, positional traps, dead ends, and insufficient space.
- **Thread-Safe Priority Queue**: Min-heap based priority queue with O(log n) operations, protected by mutexes for concurrent access.
- **Parallel A* Solver**: Implements A* search algorithm leveraging multiple goroutines for parallel path exploration to find optimal solutions.
- **Game History Tracking**: Records all moves and board states, enabling cycle detection and solution path reconstruction.
- **Gameplay Statistics**: Provides insights into moves made, unique states, session duration, and cycle counts.

## Running the Game

### Build and run:
```bash
go build -o klotski
./klotski
```

### Direct execution:
```bash
go run . # (if all .go files are in the root)
# or specify files:
go run main.go priority_queue.go heuristic.go game_history.go astar_solver.go klotski_solver.go
```

## Game Commands

Once the game starts, you can use these commands:

- `move <piece> <direction> [distance]` - Move a specific piece.
  - **piece**: Any letter (a-m) representing a game piece.
  - **direction**: `up`, `down`, `left`, `right` (or `u`, `d`, `l`, `r`).
  - **distance**: Optional number of spaces to move (default: maximum possible).
  - Examples:
    - `move a right` - Move piece 'a' as far right as possible.
    - `move l up 2` - Move piece 'l' exactly 2 spaces up.

- `pieces` (or `p`) - List all available pieces and their sizes (highlights target piece 'b').
- `show` (or `s`) - Redisplay the current board.
- `shapes` - Analyze piece shapes, their normalized forms, and current board shape hash.
- `hash` (or `h`) - Show the current board's unique FNV-1a shape hash.
- `queue` - Demonstrate the thread-safe priority queue with sample data.
- `heuristic` - Perform and display a detailed heuristic analysis of the current board state.
- `solver` - Run a simplified A* solver demonstration (single-threaded, limited steps).
- `complexity` - Analyze the estimated search space complexity for the current board.
- `history` - Display the sequence of moves made and states visited in the current session.
- `stats` - Show detailed game statistics, including move counts, unique states, and cycle detection.
- `solve` - Run the complete parallel A* solver to find the optimal solution for the current board.
- `quit` (or `q`) - Exit the game.

## AI and Solver Features

This Klotski implementation includes several advanced components to enable efficient AI solving:

- **Shape Normalization & Hashing**:
  - Pieces are identified by their shape, normalized to a canonical form (e.g., starting at (0,0)).
  - The entire board state is hashed using FNV-1a based on the shapes of all pieces and their relative positions. This allows for very fast O(1) checking of previously visited states in the A* search, crucial for performance.

- **Advanced Heuristic Function (`heuristic.go`)**:
  - The A* solver is guided by a sophisticated heuristic function that estimates the distance to the goal.
  - It analyzes multiple types of constraints:
    1.  **Path Blocking**: Pieces directly blocking 'b's path to the exit.
    2.  **Size Conflicts**: Large pieces that can't maneuver around each other.
    3.  **Positional Traps**: Pieces that would be trapped if 'b' reaches the goal.
    4.  **Dead Ends**: Pieces stuck in corners with no movement options.
    5.  **Insufficient Space**: Lack of empty spaces for necessary maneuvers.
  - The heuristic formula combines Manhattan distance with penalties derived from these constraints.

- **Thread-Safe Priority Queue (`priority_queue.go`)**:
  - A min-heap based priority queue is used to manage states to be explored by the A* algorithm.
  - It is thread-safe, using mutexes and condition variables to allow concurrent additions and pops by multiple worker goroutines in the parallel solver. Operations are O(log n).

- **Parallel A* Solver (`astar_solver.go`)**:
  - The core AI solver uses the A* search algorithm to find the optimal (shortest) sequence of moves.
  - It is implemented to run in parallel, utilizing multiple worker goroutines (defaults to `runtime.NumCPU()`).
  - Each worker pops states from the shared priority queue, generates successor states, evaluates them using the heuristic, and adds new promising states back.
  - Shared data structures like the `closedSet` (visited states) and `allNodes` (for path reconstruction) are protected by mutexes.
  - Atomic operations are used for counters like nodes explored/generated.
  - This parallel approach can significantly speed up the search for solutions on complex boards.

- **Game History & Cycle Detection (`game_history.go`)**:
  - All moves and their resulting board states (identified by their hashes) are tracked.
  - This allows for real-time cycle detection: if a move results in a board state already seen, the system alerts the user.
  - The history is also used for reconstructing the solution path once the A* solver finds the goal.

## Movement Behavior

- **Default Movement**: Without specifying distance, pieces move as far as possible in the given direction.
- **Specified Distance**: When you specify a distance, the piece moves exactly that many spaces (if possible).
- **Automatic Limiting**: If you request more spaces than available, the piece moves the maximum possible distance.
- **Collision Detection**: Pieces cannot move through walls (`O`) or other pieces.
- **Multi-cell Support**: Complex piece shapes move together as complete units.
- **Exit Access**: Only piece 'b' can move into the XXX exit area; other pieces are blocked.

## Victory Condition

When piece 'b' successfully occupies all three XXX positions (Row 8, Columns 2-4) at the bottom of the board:

```
🎉 CONGRATULATIONS! 🎉
You solved the Klotski puzzle!
Piece 'b' has successfully escaped through the exit!
```

You'll then have the option to play again or quit.

## Example Gameplay

```
=== Interactive Klotski Game ===
GOAL: Move piece 'b' to exit through the XXX area!

Klotski Game Board:

O O O O O O O 
O a b b b c O 
O a a d c c O 
O e e d g g O 
O j j h f f O 
O i i h k k O 
O l 0 0 0 m O 
O O X X X O O 

Enter command: heuristic
=== Heuristic Analysis for Klotski Solver ===
Target Distance (Manhattan): 6 moves
...
Constraints Detected: 17
...
Total Constraint Penalty: 146
Estimated Moves to Solution: 54
🚨 VERY DIFFICULT! Consider alternative approaches.

Enter command: solve
🚀 Running Complete A* Solver...
This may take some time depending on puzzle complexity.

🔍 Starting Parallel A* Search for Klotski Solution...
Using 8 worker goroutines.
Initial heuristic: 54

⏱️ Explored: 10000, Generated: 25342, OpenSet: 12053 (approx)
...
=== A* Search Results ===
Solution Found: true
Message: Solution found! 81 moves
Time Taken: 1m25.3s
Nodes Explored: 250321
Nodes Generated: 680450
Max Open Set Size: 150230
Solution Length: 81 moves

Solution Path:
1. move e left 1
2. move j left 1
...
81. move b down 1

Efficiency: 3090.38 nodes explored per move

Would you like to see the solution executed step by step? (y/n):
```

## Project Structure

- `main.go` - Main program with game logic, piece detection, win condition, and interactive interface.
- `priority_queue.go` - Thread-safe priority queue implementation for A* search.
- `heuristic.go` - Constraint analysis and heuristic calculation logic.
- `game_history.go` - Game state tracking, cycle detection, and solution path reconstruction.
- `astar_solver.go` - Complete parallel A* solver implementation.
- `klotski_solver.go` - Simplified A* solver framework for demonstration purposes.
- `go.mod` - Go module configuration.
- `README.md` - This documentation file.
- `klotski` - Compiled executable (after `go build`). 