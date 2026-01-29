# Sprite Eater Game

A simple demo game written in CoreLX for the Nitro Core DX console.

## Gameplay

- **Control**: Use arrow keys to move your white sprite around the screen
- **Goal**: Eat green sprites to gain points (+1 per green sprite)
- **Danger**: Avoid the red sprite! If you touch it, game over!

## How to Play

1. Compile the game:
   ```bash
   ./corelx test/roms/sprite_eater_game.corelx test/roms/sprite_eater_game.rom
   ```

2. Run in the emulator:
   ```bash
   ./nitro-core-dx test/roms/sprite_eater_game.rom
   ```

3. Use arrow keys to control your sprite:
   - **↑** = Move up
   - **↓** = Move down
   - **←** = Move left
   - **→** = Move right

## Game Features

- **Player Sprite**: White square with a face (8x8 pixels)
- **Green Food**: Green circular sprites that give you points
- **Red Food**: Red circular sprite that ends the game
- **Collision Detection**: Simple distance-based collision
- **Respawn System**: Green sprites respawn at new positions when eaten
- **Moving Red Sprite**: The red sprite moves slowly across the screen

## Technical Details

This game demonstrates:
- Input reading (`input.read()`)
- Sprite movement and positioning
- Collision detection
- Multiple sprites on screen
- Game state management
- Score tracking

## CoreLX Features Used

- Variable declarations (`:=` and typed)
- Control flow (`if`, `while`)
- Struct initialization (`Sprite()`)
- Built-in functions (`input.read()`, `sprite.set_pos()`, `oam.write()`)
- Bitwise operations (`&`, `|`)
- Arithmetic operations (`+`, `-`)

Enjoy the game!
