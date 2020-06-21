import copy
import random

# global variable to define the amount of random
# movements when the option is chosen.
g_random_moves = 10000
class Board:
    """
    A class representing a Board filled with blocks.
    - 'board' contains the data of the board.
    For deduction we can state that: for the board
    to be solved this constraints need satisfied:
    1 - Piece B needs to occupy [(2,4),(2,5),(3,4),(3,5)]
    down and centered.
    2 - Which leaves, all E coordinates must be above (4,X)
    Therefore the first objective is to move Piece E above B
    """
    def __init__(self):
        self.board = [['O', 'O', 'O', 'O', 'O', 'O'],
                      ['O', 'a', 'b', 'b', 'c', 'O'],
                      ['O', 'a', 'b', 'b', 'c', 'O'],
                      ['O', 'd', 'e', 'e', 'f', 'O'],
                      ['O', 'd', 'g', 'h', 'f', 'O'],
                      ['O', 'i', '0', '0', 'j', 'O'],
                      ['O', 'O', 'O', 'O', 'O', 'O']]
        self.pieces = {}
        self.computePieces()
        self.oppositeDirection = { 'u':'d','d':'u','l':'r','r':'l',
                                'ut':'dt','dt':'ut','lt':'rt','rt':'lt' }

    def computePieces(self):
        """
        computes the pieces and where they can be found
        """
        for line in self.board:
            for p in line:
                if p.islower():
                    self.pieces[p] = []

        for y, line in enumerate(self.board):
            for x, element in enumerate(line):
                if element in self.pieces:
                    self.pieces[element].append([x, y])

    def pieceHash(self, piece):
        return (self.pieces[piece][0][0] * len(self.board)) + self.pieces[piece][0][1]

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
        g = self.pieceHash('g')
        h = self.pieceHash('h')
        i = self.pieceHash('i')
        j = self.pieceHash('j')

        # two vertical
        a = self.pieceHash('a')
        c = self.pieceHash('c')
        d = self.pieceHash('d')
        f = self.pieceHash('f')

        # two horizontal
        e = self.pieceHash('e')
    
        # two by two
        b = self.pieceHash('b')
        return hash((g + h + i + j, a + c + d + f, e, b))
        
    @property
    def defective(self):
        """
        returns the value accumulating two factors representing
        how far is this board from the final solution:
            - how far is b from final position
            - how much e up relative to b
        """
        first_corner = self.pieces['b'][0]
        b_obj = [2, 4]
        b_manhattan = 30 * abs(b_obj[0] - first_corner[0]) + 313 * abs(b_obj[1] - first_corner[1])
        e_y_def = max(self.pieces['e'][0][1] - 3, 0) * 137
        return b_manhattan + e_y_def

    @property
    def done(self):
        """
        returns True if this board got to its objective.
        """
        return self.defective == 0

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

    def candidateMoves(self, piece, coordn):
        """
        returns whether or not the is empty spaces in all directions
        relative to coordn (coordinates)
        example of input coordn.
        [[2, 1], [3, 1], [2, 2], [3, 2]]
        example of returned value:
        {'u': True, 'd': False, 'l': False, 'r': False}, True
        for a piece that can only move up, can be moved (last ret)
        """
        moves = {'u': False, 'd': False, 'l': False, 'r': False,
                'ut': False, 'dt': False, 'lt': False, 'rt': False}

        moves['l'] = all([self.empty(c[0] - 1, c[1])
            or self.e(c[0] - 1, c[1]) == piece for c in coordn])

        if moves['l']:
            moves['lt'] = all([self.empty(c[0] - 2, c[1])
                or self.e(c[0] - 2, c[1]) == piece for c in coordn])

        moves['r'] = all([self.empty(c[0] + 1, c[1])
            or self.e(c[0] + 1, c[1]) == piece for c in coordn])

        if moves['r']:
            moves['rt'] = all([self.empty(c[0] + 2, c[1])
                or self.e(c[0] + 2, c[1]) == piece for c in coordn])
        
        moves['u'] = all([self.empty(c[0], c[1] - 1)
            or self.e(c[0], c[1] - 1) == piece for c in coordn])

        if moves['u']:
            moves['ut'] =  all([self.empty(c[0], c[1] - 2)
                or self.e(c[0], c[1] - 2) == piece for c in coordn])

        moves['d'] = all([self.empty(c[0], c[1] + 1)
            or self.e(c[0], c[1] + 1) == piece for c in coordn])

        if moves['d']:
            moves['dt']= all([self.empty(c[0], c[1] + 2)
                or self.e(c[0], c[1] + 2) == piece for c in coordn])

        can_move = any([moves['l'], moves['r'], moves['u'], moves['d'],
                        moves['lt'], moves['rt'], moves['ut'], moves['dt']])

        return moves, can_move

    def posibleMoves(self):
        """
        returns only the pieces that can move and their movable direction.
        a possible return value would be like this:
        {'g': ['d'], 'h': ['d'], 'i': ['r'], 'j': ['l']}
        for piece 'g' can move down, 'h' can move down, 'i' can move right
        and 'j' can move left.
        """
        moves = {}
        for p, c in self.pieces.items():
            pmvs, can_move = self.candidateMoves(p, c)
            if can_move:
                moves[p] = [k for k, v in pmvs.items() if v]
        return moves

    def move(self, pieceName, direction):
        """
        Updates two elements on each call.
        - the board itself.
        - each coordinate of the piece
        """
        for c in self.pieces[pieceName]:
            self.setE(c[0], c[1], '0')
        
        if direction == 'u':
            for coor in self.pieces[pieceName]:
                coor[1] -= 1

        if direction == 'd':
            for coor in self.pieces[pieceName]:
                coor[1] += 1

        if direction == 'l':
            for coor in self.pieces[pieceName]:
                coor[0] -= 1

        if direction == 'r':
            for coor in self.pieces[pieceName]:
                coor[0] += 1

        if direction == 'ut':
            for coor in self.pieces[pieceName]:
                coor[1] -= 2

        if direction == 'dt':
            for coor in self.pieces[pieceName]:
                coor[1] += 2

        if direction == 'lt':
            for coor in self.pieces[pieceName]:
                coor[0] -= 2

        if direction == 'rt':
            for coor in self.pieces[pieceName]:
                coor[0] += 2

        for c in self.pieces[pieceName]:
            self.setE(c[0], c[1], pieceName)

    def simulateMove(self, pieceName, direction):
        """
        emulates the move requested to identify
        properties of the possible table.
        """
        self.move(pieceName, direction)
        h = self.hash
        d = self.done
        self.move(pieceName, self.oppositeDirection[direction])
        return h, d

class moveNode:
    """
    A class representing all possible moves in a board
    at a given state.
    {'g': ['d'], 'h': ['d'], 'i': ['r'], 'j': ['l']}
    """
    # a set of hashes for all the seen boards
    seen = set()
    names = {'d': 'down', 'u': 'up', 'l': 'left', 'r': 'right',
            'dt': 'down twice', 'ut': 'up twice', 'lt': 'left twice', 'rt': 'right twice'}

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
        return (self.deep*10) + self.board.defective

    def flattenMoves(self):
        """
        Obtains the possible moves on the board
        and stripes them on unit possible moves.
        """
        for pieceName, directions in self.board.posibleMoves().items():
            for direction in directions:
                self.playableMoves.append([None, [pieceName, direction]])

    def run(self):
        """
        runs through the sequence of possible moves, and
        if there is novelty, creates this object for the move.
        finally returns the movements that could happen at
        this moment for the current board.
        """
        for i, (_, (piece, direction)) in enumerate(self.playableMoves):
            # we simulate the move in place
            hash, done = self.board.simulateMove(piece, direction)
            # if the result of the similation was
            # previously visited we skip this step
            if hash in moveNode.seen:
                continue
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
                    print ("Step", (step + 1), " piece:", m[0], "goes", moveNode.names[m[1]])

                return [[None, [piece,direction]]]

            moveNode.seen.add(hash)

            newBoard = copy.deepcopy(self.board)
            newBoard.move(piece, direction)
            self.playableMoves[i][0] = moveNode(newBoard, self, [piece,direction])

        cleanedSequence = [s for s in self.playableMoves if s[0] is not None]
        return cleanedSequence

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
            pos_moves = myboard.posibleMoves()
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
                moves_dict = myboard.posibleMoves()
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
                moves_dict = myboard.posibleMoves()
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
        
        # full tree solution
        if inputOption == 'd':
            queue = [[PotentialMoves(myboard), ['', '']]]
            found = False
            while len(queue) and not found:
                pm = queue.pop(0)[0]
                nss = pm.run()
                for ns in nss:
                    if not ns[0]:
                        pm.board.printState()
                        found = True
                        break
                    queue.append(ns)
        
        # educated guess solution
        if inputOption == 'a':
            queue = [[PotentialMoves(myboard), ['', '']]]
            found = False
            while len(queue) and not found:
                pm = queue.pop(0)[0]
                nss = pm.run()
                for ns in nss:
                    if not ns[0]:
                        pm.board.printState()
                        found = True
                        return
                    queue.append(ns)
                queue.sort(key=lambda pm: pm[0].f)
        
        if inputOption == 's':
            myboard.printState()

playBoard()
