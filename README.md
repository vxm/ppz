# ppz
Python Puzzle.

This project creates a game, which can be played once the python script is executed and solves it at any stage.

To understand the puzzle; have a look to the image added at root level and shown in my Wiki page.

https://github.com/vxm/ppz/wiki

How it works:

To solve the puzzle, the black piece needs to leave by red door.


The rules are:

  • A piece may only move vertically or horizontally into empty
  space next to it. 

  • If there are 2 empty spaces in given direction, the given piece
  may move 1 or 2 spaces (counting as 1 move)

  • Only the black piece can go through the red door.
  
  
Solution:
  

 To execute the solution, open menu and enter key "a", as per menu instructions this will execute the A* algorithm which solves the problem. The heuristic value have two stages, the first phase and clue for the algorithm is, that the black piece needs to be under the double size horizontal white piece. And the second stage or objective is found when the black piece is placed to in the only possible position for it to leave the board.
 
 Plese note that the python solution, once executed, it will include options to play with the board. The A* algorithm and Dijkstra algorithm will solve from there.
 
 The key for the performance of the algorithm was allowing the board hash clashes.

 
