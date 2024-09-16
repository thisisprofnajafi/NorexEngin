
# NOREX - Real-Time Multiplayer Game

## Game Concept:
The game is a mobile multiplayer platform where users can create rooms and play classic board/card games like **Uno**, **Chess**, and **Memory Games**. Users can invite friends or join public rooms to play in real-time with voice and text chat functionality integrated. Games will have 2 to 10 players, offering a social, fun, and competitive experience.

## Core Features:
1. **Real-Time Gameplay**:
    - The games are designed to be played live with all moves happening instantly. The game engine manages the user turns, rules, and any interactive elements.

2. **Game Types**:
    - **Uno**: A card game where users play matching cards by color or number, with various action cards to change gameplay dynamics.
    - **Chess**: A classic strategy board game between two players where the goal is to checkmate the opponent's king.
    - **Memory Game**: Players flip cards and try to match pairs, testing their memory skills.

3. **Room Management**:
    - Players can create private or public rooms to play games.
    - Rooms can hold between 2 and 10 players.

4. **Voice and Text Chat**:
    - Players can communicate with each other via integrated text and voice chat systems during gameplay.

5. **Real-Time Data Handling**:
    - All game actions, chat messages, and game state changes are handled in real-time by the server.
    - Temporary data such as chat logs, game state, and user actions are removed after the game ends.

## Technical Stack:
- **Backend**: GoLang is used for the real-time game engine, server, and managing user turn logic.
- **Database**:
    - **Persistent Database**: To store user profiles, room settings, game history, etc.
    - **Real-Time Database**: To handle game state, turns, and real-time actions. Redis or ScyllaDB for real-time data handling.
- **Frontend**: React Native (or Flutter) for cross-platform mobile support.
- **Communication**: WebSockets for real-time communication.

## Scalability:
- The system is designed to handle **10,000 concurrent real-time games** with up to 10 users in each game room.
- Efficient resource usage and optimized real-time data handling ensure smooth gameplay, even with thousands of users online at the same time.

## Game Flow:
1. Users can log in and create or join game rooms.
2. Once all players are ready, the game begins, and user actions are processed by the real-time game engine.
3. Players communicate through text or voice chat during the game.
4. After the game ends, all temporary game data is deleted, and users return to the lobby to play again.

## Future Features:
- **Customizable Avatars and Profiles**.
- **Ranked Matchmaking** for competitive gameplay.
- **New Game Modes** based on player demand (e.g., Poker, Checkers, etc.).
- **Cross-Platform Support** for both mobile and desktop.
