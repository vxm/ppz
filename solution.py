import bisect
import copy
import functools
import math
import random
import time

# global variable to define the amount of random
# movements when the option is chosen.
g_random_moves = 1000
PENALTY_DIVISION = 100000

class Board:
    """
    A class representing a Board filled with blocks.
    - 'board' contains the data of the board.
    """
    
    def __init__(self):
        self.board = [['O', 'O', 'O', 'O', 'O', 'O', 'O'],
                    ['O', 'a', 'b', 'b', 'b', 'c', 'O'],
                    ['O', 'a', 'a', 'd', 'c', 'c', 'O'],
                    ['O', 'e', 'e', 'd', 'g', 'g', 'O'],
                    ['O', 'e', 'e', 'h', 'g', 'g', 'O'],
                    ['O', 'j', 'j', 'h', 'f', 'f', 'O'],
                    ['O', 'i', 'i', 'n', 'k', 'k', 'O'],
                    ['O', 'l', '0', '0', '0', 'm', 'O'],
                    ['O', 'O', 'O', 'O', 'O', 'O', 'O']]
        self.objetive_position = [2, len(self.board) - 2]
        self.resetCache()
        self.pieces = {}
        self.hashes = {}
        self.computePieces()
        
        Board.oppositeDirection = {'u':'d','d':'u','l':'r','r':'l'}

    def resetCache(self):
        self._defective = 10000000

    def computePieces(self):
        """
        computes the pieces and where they can be found.
        """
        # initialising the pieces dictionary with
        # empty arrays
        for line in self.board:
            for p in line:
                if p.islower() and p not in self.pieces:
                    self.pieces[p] = []

        # adding coordinates to those pieces.
        for y, line in enumerate(self.board):
            for x, element in enumerate(line):
                if element in self.pieces:
                    self.pieces[element].append([x, y])

        for k in self.pieces.keys():
            self.hashes[k] = self.pieceHash(k)

    def pieceHash(self, piece):
        if piece in self.pieces.keys():
            pieceCoordinates = self.pieces[piece][0]
            return hash((pieceCoordinates[0]*9757157, pieceCoordinates[1]))
        return 1234567

    @property
    def hash(self):
        """
        Returns the hash from the tuple of first coordinates of each piece,
        given they cannot rotate, that defines their unique value.
        The hash for each piece coordinate is computed multiplying the first
        coordinate by length of the board adding the second coordinate,
        to ensure there won't be a clash.
        """
        # unique ones
        l = self.pieceHash('l')
        m = self.pieceHash('m')
        n = self.pieceHash('n')
    
        i = self.pieceHash('i')
        k = self.pieceHash('k')
        g = self.pieceHash('g')
        e = self.pieceHash('e')
        f = self.pieceHash('f')
        j = self.pieceHash('j')
    
        h = self.pieceHash('h')
        d = self.pieceHash('d')
    
        # two vertical
        a = self.pieceHash('a')
        c = self.pieceHash('c')

        b = self.pieceHash('b')
        return hash((l*m*n, i*k*g*e*f*j, h*d, a*c,b))

    @property
    def b_defective(self):
        """
        returns how far is b from final position
        """
        b_first_corner = self.pieces['b'][0]
        b_distance_to_objective = [0,0]
        b_distance_to_objective[0] = self.objetive_position[0] - b_first_corner[0]
        b_distance_to_objective[1] = self.objetive_position[1] - b_first_corner[1]
        if 2 + b_first_corner[1] < self.objetive_position[1]:
            return  b_distance_to_objective[1] * 2
        
        result = math.sqrt(b_distance_to_objective[0]**2 + b_distance_to_objective[1]**2) * 2
        return result
    
    @property
    def defective(self):
        """
        returns the value accumulating two factors representing
        how far is this board from the final solution:
            - how far is b from final position
            - how much e up relative to b
        """
        if self._defective != 10000000:
            return self._defective

        defective = self.b_defective
    
        b_first_corner = self.pieces['b'][0]

        incompatibles = ['a','c','i','g','k','e','f','j']
        for incompatible in incompatibles:
            bad_first_corner = self.pieces[incompatible][0]
            if bad_first_corner >= b_first_corner:
                defective += 2 / len(incompatibles)
        
        self._defective = defective
        return defective

    @property
    def done(self):
        """
        returns True if this board got to its objective.
        """
        return self.b_defective == 0.0

    def printState(self):
        """
        Prints the board on it's current state.
        """
        for line in self.board:
            print(line)
        print("defectiveness:", self.defective)

    def e(self, x, y):
        """
        Returns the element at the given coordinates.
        """
        if x < 0 or y < 0:
            return 'O'
    
        if x >= len(self.board[0]) or y >= len(self.board):
            return 'O'
        
        return self.board[y][x]

    def setE(self, x, y, v):
        """
        Sets the element at the given coordinates.
        """
        self.board[y][x] = v

    def empty(self, x, y):
        """
        Returns if the element at the given coordinates represents
        an empty space.
        """
        return self.e(x, y) == '0'
    

    def piecePossibleMoves(self, piece, coordinates):
        """
        returns whether or not the is empty spaces in all directions
        relative to coordinates (coordinates)
        example within standard board, of input coordinates.
        [[2, 1], [3, 1], [2, 2], [3, 2]]
        example of returned value:
            {'u': True, 'd': False, 'l': False, 'r': False}, True
        for a piece that can only move up, can be moved (last ret)
        """
        moves = {}

        l = 1
        clear = True
        while clear:
            key = 'l' + str(l)
            candidates = [self.empty(c[0] - l, c[1]) or (self.e(c[0] - l, c[1]) == piece) for c in coordinates]
            moves[key] = all(candidates)
            l+=1
            clear = moves[key]

        r = 1
        clear = True
        while clear:
            key = 'r' + str(r)
            moves[key] = all([self.empty(c[0] + r, c[1])
                or self.e(c[0] + r, c[1]) == piece for c in coordinates])
            r+=1
            clear = moves[key]

        u = 1
        clear = True
        while clear:
            key = 'u' + str(u)
            moves[key] = all([self.empty(c[0], c[1] - u)
                or self.e(c[0], c[1] - u) == piece for c in coordinates])
            u+=1
            clear = moves[key]

        d = 1
        clear = True
        while clear:
            key = 'd' + str(d)
            moves[key] = all([self.empty(c[0], c[1] + d)
                or self.e(c[0], c[1] + d) == piece for c in coordinates])
            d+=1
            clear = moves[key]

        can_move = any([m[1] for m in moves.items()])
        return moves, can_move

    def possibleMoves(self):
        """
        returns only the pieces that can move and their movable direction.
        a possible return value would be like this:
            {'g': ['d'], 'h': ['d'], 'i': ['r'], 'j': ['l']}
        for piece 'g' can move down, 'h' can move down, 'i' can move right
        and 'j' can move left.
        """
        moves = {}
        for p, c in self.pieces.items():
            allMoves, canMove = self.piecePossibleMoves(p, c)
            if canMove:
                moves[p] = [k for k, v in allMoves.items() if v]
        return moves

    def move(self, pieceName, moves):
        """
        Updates two elements on each call.
        - the board itself.
        - each coordinate of the piece
        """
        for c in self.pieces[pieceName]:
            self.setE(c[0], c[1], '0')

        (direction, steps) = moves

        if direction == 'u':
            for coor in self.pieces[pieceName]:
                coor[1] -= int(steps)

        if direction == 'd':
            for coor in self.pieces[pieceName]:
                coor[1] += int(steps)

        if direction == 'l':
            for coor in self.pieces[pieceName]:
                coor[0] -= int(steps)

        if direction == 'r':
            for coor in self.pieces[pieceName]:
                coor[0] += int(steps)

        for c in self.pieces[pieceName]:
            self.setE(c[0], c[1], pieceName)

        self.hashes[pieceName] = self.pieceHash(pieceName)

    def simulateMove(self, pieceName, direction):
        """
        emulates the move requested to identify
        properties of the possible table.
        """
        # do
        self.move(pieceName, direction)
        # store temp data
        h = self.hash
        d = self.done
        # undo

        direction = Board.oppositeDirection[direction[0]] + direction[1]
        self.move(pieceName, direction)
        return h, d

class moveNode:
    """
    A class representing all possible moves in a board
    at a given state.
    {'g': ['d'], 'h': ['d'], 'i': ['r'], 'j': ['l']}
    """
    # a set of hashes for all the seen boards
    seen = set()
    names = {'d': 'down', 'u': 'up', 'l': 'left', 'r': 'right'}

    def __init__(self, board, parent=None, moves=None):
        self.board = board
        self.parent = parent
        self.moves = moves
        moveNode.seen.add(board.hash)
        self.playableMoves = []
        self.flattenMoves()
        if parent is not None:
            self._deep = parent.deep + 1
        else:
            self._deep = 0

    @property
    def deep(self):
        return self._deep

    @property
    def penalty(self):
        return (self.deep/PENALTY_DIVISION) + self.board.defective

    def flattenMoves(self):
        """
        Obtains the possible moves on the board
        and stripes them on unit possible moves.
        """
        for pieceName, directions in self.board.possibleMoves().items():
            for direction in directions:
                self.playableMoves.append([None, [pieceName, direction]])

    def nodeMoves(self):
        """
        runs through the sequence of possible moves, and
        if there is novelty, creates this object for the move.
        finally returns the movements that could happen at
        this moment for the current board.
        """
        nodes = []
        for i, (_, (piece, direction)) in enumerate(self.playableMoves):
            # we simulate the move in place
            hashr, done = self.board.simulateMove(piece, direction)
            # if the result of the similation was
            # previously visited we skip this step
            if hashr in moveNode.seen:
                continue

            # print("\tnovel move: "+ piece +" "+direction)
            if done:
                # a bit of a celebration here!.
                print('\n\n-----------*****************************-----------')
                print('-----------* This solves the problem! **-----------')
                print('-----------*****************************-----------\n\n')
                self.board.move(piece, direction)
                moveInstructions = [[piece,direction]]
                parentIt = self
                while parentIt:
                    # while still don't reach the root
                    if parentIt.moves:
                        moveInstructions.insert(0, parentIt.moves)
                    parentIt = parentIt.parent

                for step, m in enumerate(moveInstructions):
                    name = moveNode.names[m[1][0]]
                    print (f"Step:{(step + 1)}, piece:{m[0]}, moves:{name}, nSteps:{m[1][1]}, moveNode_penalty:{moveNode.penalty}, defective:{self.board.defective}")

                return None

            newBoard = copy.deepcopy(self.board)
            newBoard.resetCache()
            newBoard.move(piece, direction)
            newMove = moveNode(newBoard, self, [piece, direction])
            nodes.append(newMove)

        return nodes


@functools.total_ordering
class Node:
    def __init__(self, node):
        self.node = node
        self.penalty = node.penalty
    def __lt__(self, other):
        return self.penalty < other.penalty
    def __str__(self):
        return '{} {}'.format(self.node, self.penalty)

def playBoard():
    """
    This function allows you to play with a board and visualize the results
    """
    inputOption = '-'
    myboard = Board()
    print("\n\n\t------------- START -------------")
    myboard.printState()
    while(inputOption != 'e' and inputOption != 'q'):
        print("---------------------------")
        print("Choose your option:")
        print("---------------------------")
        print("\n\t(q/e) Quit\
            \n\t(m) Manual moves\
            \n\t(r) Shuffle board\
            \n\t(b) Random brute force solution\
            \n\t(d) Dijkstra solution\
            \n\t(a) A* solution.\
            \n\t(s) Show board.")
        print("---------------------------")
        inputOption = input("Option:")

        # manual solution
        if inputOption == 'm':
            myboard.printState()
            pos_moves = myboard.possibleMoves()
            print(pos_moves)
            pieceName = input("Select piece: ")
            if pieceName in pos_moves:
                dict_val = pos_moves[pieceName]
                if len(dict_val) > 1:
                    msg = "Select direction: "
                    for v in dict_val:
                        msg += "(" + v + ")"
                    msg += "."
                    move = input(msg)
                    myboard.move(pieceName, move)
                else:
                    myboard.move(pieceName, dict_val[0])
                myboard.printState()
            else:
                print('Impossible move')

        # Shuffle board
        if inputOption == 'r':
            st = 0
            for _ in range(0, g_random_moves + 1):
                moves_dict = myboard.possibleMoves()
                pos_moves_listed = list(moves_dict)
                option = random.choice(pos_moves_listed)
                print(option)
                directions = moves_dict[option]
                direction = random.choice(directions)
                myboard.move(option, direction)
                myboard.printState()
                print("Board shuffled", st, "times.")
                st += 1

        # Brute force
        if inputOption == 'b':
            st = 0
            while myboard.defective != 0:
                moves_dict = myboard.possibleMoves()
                pos_moves_listed = list(moves_dict)
                option = random.choice(pos_moves_listed)
                directions = moves_dict[option]
                direction = random.choice(directions)
                myboard.move(option, direction)
                if st % 11 == 0:
                    myboard.printState()
                    print("Board shuffled", st, "times.")
                st += 1
            myboard.printState()
            print("Board shuffled", st, "times.")

        # educated guess solution
        if inputOption == 'a':
            queue = [Node(moveNode(myboard))]
            while queue:
                queuedNode = queue.pop(0)
                nextMoves = queuedNode.node.nodeMoves()
                if nextMoves is None:
                    print('\n\n')
                    queuedNode.node.board.printState()
                    print("Seen size " + str(len(moveNode.seen)))
                    return

                for ns in nextMoves:
                    bisect.insort_left(queue, Node(ns))

            print("No solution found")
            return

        if inputOption == 's':
            myboard.printState()



start_time = time.time()

playBoard()

end_time = time.time()
elapsed_seconds = end_time - start_time

hours = elapsed_seconds // 3600
minutes = (elapsed_seconds % 3600) // 60
seconds = elapsed_seconds % 60

print(f"Execution time: {int(hours)} hours, {int(minutes)} minutes, {seconds:.2f} seconds")