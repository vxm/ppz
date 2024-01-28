import bisect
import copy
import functools
import math
import random
import json
import bpy

bpy.context.scene.frame_set(0)

global ctime
ctime = 0

def advanceTime(nt):
    global ctime
    ctime += nt
    bpy.context.scene.frame_set(ctime)

def select(objName):
    ob = bpy.context.scene.objects[objName]       # Get the object
    bpy.ops.object.select_all(action='DESELECT') # Deselect all objects
    bpy.context.view_layer.objects.active = ob   # Make the cube the active object
    ob.select_set(True)
    return ob

def keyLocation(pieceName):
    select(pieceName)
    bpy.ops.anim.keyframe_insert_by_name(type="Location")

def keyAll():
    for k in range(0,14):
        keyLocation(chr(k+ord("a")))


def move(piece, direction, amount):
    bo = select(piece)

    bpy.ops.transform.translate(value=(direction[0]*amount, direction[1]*amount, 0),
                                orient_type='GLOBAL', orient_matrix=((1, 0, 0), (0, 1, 0), (0, 0, 1)),
                                orient_matrix_type='GLOBAL', constraint_axis=(False, True, False),
                                mirror=True,
                                use_proportional_edit=False,
                                proportional_edit_falloff='SMOOTH',
                                proportional_size=1,
                                use_proportional_connected=False,
                                use_proportional_projected=False,
                                release_confirm=True)


def left(piece, amount, nnt):
    advanceTime(nnt)
    move(piece, [1,0], -2 * amount)
    keyAll()

def right(piece, amount, nnt):
    advanceTime(nnt)
    move(piece, [1,0], 2 * amount)
    keyAll()

def down(piece, amount, nnt):
    advanceTime(nnt)
    move(piece, [0,1], -2 * amount)
    keyAll()

def up(piece, amount, nnt):
    advanceTime(nnt)
    move(piece, [0,1], 2 * amount)
    keyAll()


def read_and_process_moves(json_file):
    try:
        with open(json_file, 'r') as file:
            data = json.load(file)
            moves = data.get('moves', [])

            for move in moves:
                if len(move) == 4:
                    number, piece, direction, amount = move
                    print(f"Move {number}: Piece '{piece}', Direction '{direction}', Amount {amount}")
                    amount = int(amount)
                    advance_frames = 5
                    if piece == "advance_frames":
                        advance_frames += 4
                    if direction == "up":
                        up(piece, amount, advance_frames + amount)
                    if direction == "down":
                        down(piece, amount, advance_frames + amount)
                    if direction == "left":
                        left(piece, amount, advance_frames + amount)
                    if direction == "right":
                        right(piece, amount, advance_frames + amount)
                else:
                    print("Invalid move format")
    except FileNotFoundError:
        print("File not found")
    except json.JSONDecodeError:
        print("Error decoding JSON")
    except Exception as e:
        print(f"An error occurred: {e}")

# Replace 'your_file.json' with the path to your JSON file
json_file = '/Users/mariano/code/GitHub/ppz/impossible_as_list.json'

for k in range(0,14):
    keyLocation(chr(k+ord("a")))
advanceTime(10)
    
read_and_process_moves(json_file)