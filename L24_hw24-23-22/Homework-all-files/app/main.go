package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Constants for canvas and paddle dimensions
const (
	CanvasWidth  = 800
	CanvasHeight = 600
	PaddleHeight = 100
	PaddleWidth  = 20
	MaxPaddleY   = CanvasHeight - PaddleHeight
)

// Message types
const (
	AssignMessage  = "assign"
	MoveMessage    = "move"
	UpdateMessage  = "update"
	GameOverMsg    = "gameover"
	ErrorMessage   = "error"
	StartGameMsg   = "start_game"
	RestartGameMsg = "restart_game"
	RoomChangeMsg  = "room_change"
	PauseGameMsg   = "pause_game"
	ResumeGameMsg  = "resume_game"
)

// Message structure
type Message struct {
	Type       string  `json:"type"`
	RoomID     string  `json:"room_id,omitempty"`
	Player     string  `json:"player,omitempty"`
	Y          *int    `json:"y,omitempty"`
	LeftY      int     `json:"leftY,omitempty"`
	RightY     int     `json:"rightY,omitempty"`
	BallX      float64 `json:"ballX,omitempty"`
	BallY      float64 `json:"ballY,omitempty"`
	Winner     string  `json:"winner,omitempty"` // For game over messages
	ScoreLeft  int     `json:"scoreLeft,omitempty"`
	ScoreRight int     `json:"scoreRight,omitempty"`
}

type ErrorDetail struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// Ball structure representing the ball's state
type Ball struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	Vx float64 `json:"vx"`
	Vy float64 `json:"vy"`
}

// GameRoom structure representing each game room
type GameRoom struct {
	ID            string
	Players       map[*websocket.Conn]string // 'left' and 'right'
	GameState     GameState
	Mutex         sync.Mutex
	Status        string // "waiting", "active", "paused"
	IsLoopRunning bool   // Add this field
}

// Game state structure
type GameState struct {
	PanYLeft   int
	PanYRight  int
	Ball       Ball
	ScoreLeft  int
	ScoreRight int
}

// Initialize global variables
var (
	rooms      = make(map[string]*GameRoom)
	roomsMutex = sync.Mutex{}

	upgrader = websocket.Upgrader{
		// Allow all origins for simplicity. In production, restrict this.
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ticker = time.NewTicker(time.Millisecond * 160) // ~60 FPS
)

// AssignRoom assigns a player to an existing waiting room or creates a new one.
func AssignRoom(conn *websocket.Conn) *GameRoom {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	// Search for a room with status "waiting"
	log.Printf("Assigning player %s to a room", conn.RemoteAddr())
	log.Printf("Available rooms: %v", rooms)
	for _, room := range rooms {
		room.Mutex.Lock()
		if room.Status == "waiting" && len(room.Players) == 1 {
			// Assign player as 'right'
			room.Players[conn] = "right"
			room.Status = "active"
			room.Mutex.Unlock()
			log.Printf("Player assigned to room %s as right", room.ID)
			return room
		}
		room.Mutex.Unlock()
	}

	// No available room, create a new one
	newRoomID := uuid.New().String()
	newRoom := &GameRoom{
		ID:      newRoomID,
		Players: make(map[*websocket.Conn]string),
		GameState: GameState{
			PanYLeft:  CanvasHeight/2 - PaddleHeight/2, // 250
			PanYRight: CanvasHeight/2 - PaddleHeight/2, // 250
			Ball: Ball{
				X:  float64(CanvasWidth / 2),
				Y:  float64(CanvasHeight / 2),
				Vx: 4.0,
				Vy: 4.0,
			},
			ScoreLeft:  0,
			ScoreRight: 0,
		},
		Status:        "waiting",
		IsLoopRunning: false, // Initialize the new field
	}

	// Assign player as 'left'
	newRoom.Players[conn] = "left"
	rooms[newRoomID] = newRoom
	log.Printf("Player assigned to new room %s as left", newRoomID)
	return newRoom
}

// handleConnections manages WebSocket connections and room assignments
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer ws.Close()

	// Assign room to the new connection
	room := AssignRoom(ws)
	log.Printf("Player %s assigned to room %s", ws.RemoteAddr(), room.ID)
	player := room.Players[ws]

	// Send assign message with room ID and player role
	assignMsg := Message{
		Type:   AssignMessage,
		RoomID: room.ID,
		Player: player,
	}
	if err := ws.WriteJSON(assignMsg); err != nil {
		log.Println("Error sending assign message:", err)
		return
	}

	// If room is now active, start the game loop for the room
	if room.Status == "active" {
		room.Mutex.Lock()
		if !room.IsLoopRunning {
			room.IsLoopRunning = true
			go RoomGameHandler(room)
		}
		room.Mutex.Unlock()

		startGameMsg := Message{
			Type: StartGameMsg,
		}
		BroadcastToRoom(room, startGameMsg)
	}

	log.Printf("Player %s assigned to room %s as %s", ws.RemoteAddr(), room.ID, player)

	// Listen for messages from the client
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		// In handleConnections, after detecting a read error:
		if err != nil {
			log.Printf("Error reading message from %s: %v", ws.RemoteAddr(), err)
			errorDetail := ErrorDetail{
				Error:   "ReadError",
				Details: "Failed to read message from the server.",
			}
			sendErrorMessage(ws, errorDetail)
			break
		}

		//log.Printf("Received message from %s in room %s: %+v", ws.RemoteAddr(), room.ID, msg)

		switch msg.Type {
		case MoveMessage:
			if msg.Player != player || msg.Y == nil {
				log.Printf("Invalid move message from %s: %+v", ws.RemoteAddr(), msg)
				errorDetail := ErrorDetail{
					Error:   "ReadError",
					Details: "Failed to read message from the server.",
				}
				sendErrorMessage(ws, errorDetail)
				continue
			}
			ProcessMove(room, player, *msg.Y)
		case RestartGameMsg:
			if room.Status != "active" {
				log.Printf("Cannot restart game in room %s: Game not active", room.ID)
				errorDetail := ErrorDetail{
					Error:   "ReadError",
					Details: "Failed to read message from the server.",
				}
				sendErrorMessage(ws, errorDetail)
				continue
			}
			RestartGame(room)
		case PauseGameMsg:
			room.Mutex.Lock()
			room.Status = "paused"
			room.Mutex.Unlock()
			pauseMsg := Message{Type: PauseGameMsg}
			BroadcastToRoom(room, pauseMsg)
		case ResumeGameMsg:
			room.Mutex.Lock()
			room.Status = "active"
			room.Mutex.Unlock()
			resumeMsg := Message{Type: ResumeGameMsg}
			BroadcastToRoom(room, resumeMsg)
		case RoomChangeMsg:
			// Handle room change by removing the player and possibly assigning to a new room
			ChangeRoom(ws, room)
		default:
			log.Printf("Unknown message type from %s: %s", ws.RemoteAddr(), msg.Type)
			errorDetail := ErrorDetail{
				Error:   "ReadError",
				Details: "Failed to read message from the server.",
			}
			sendErrorMessage(ws, errorDetail)
		}
	}

	// Cleanup after disconnect
	CleanupRoom(ws, room)
}

// RoomGameHandler manages the game logic within a specific room
func RoomGameHandler(room *GameRoom) {
	defer func() {
		room.Mutex.Lock()
		room.IsLoopRunning = false
		room.Mutex.Unlock()
	}()

	for {
		<-ticker.C
		room.Mutex.Lock()
		if room.Status == "paused" {
			room.Mutex.Unlock()
			continue
		}
		if room.Status != "active" {
			room.Mutex.Unlock()
			return
		}

		// Update ball position
		room.GameState.Ball.X += room.GameState.Ball.Vx
		room.GameState.Ball.Y += room.GameState.Ball.Vy

		// Collision with top wall
		if room.GameState.Ball.Y <= 0 {
			room.GameState.Ball.Y = 0
			room.GameState.Ball.Vy = -room.GameState.Ball.Vy
		}

		// Collision with bottom wall
		if room.GameState.Ball.Y >= float64(CanvasHeight) {
			room.GameState.Ball.Y = float64(CanvasHeight)
			room.GameState.Ball.Vy = -room.GameState.Ball.Vy
		}

		// Collision with left paddle
		if room.GameState.Ball.X <= float64(PaddleWidth) {
			if int(room.GameState.Ball.Y) >= room.GameState.PanYLeft && int(room.GameState.Ball.Y) <= (room.GameState.PanYLeft+PaddleHeight) {
				room.GameState.Ball.X = float64(PaddleWidth)
				room.GameState.Ball.Vx = -room.GameState.Ball.Vx
			}
		}

		// Collision with right paddle
		if room.GameState.Ball.X >= float64(CanvasWidth-PaddleWidth) {
			if int(room.GameState.Ball.Y) >= room.GameState.PanYRight && int(room.GameState.Ball.Y) <= (room.GameState.PanYRight+PaddleHeight) {
				room.GameState.Ball.X = float64(CanvasWidth - PaddleWidth)
				room.GameState.Ball.Vx = -room.GameState.Ball.Vx
			}
		}

		// Check for game over
		var winner string
		if room.GameState.Ball.X < 0 {
			// Ball touched the left wall, right player scores
			room.GameState.ScoreRight += 1
			winner = "right"
			room.GameState.Ball.X = float64(CanvasWidth / 2)
			room.GameState.Ball.Y = float64(CanvasHeight / 2)
			room.GameState.Ball.Vx = 4.0
			room.GameState.Ball.Vy = 4.0
			room.GameState.PanYLeft = CanvasHeight/2 - PaddleHeight/2  // 250
			room.GameState.PanYRight = CanvasHeight/2 - PaddleHeight/2 // 250
		}
		if room.GameState.Ball.X > float64(CanvasWidth) {
			// Ball touched the right wall, left player scores
			room.GameState.ScoreLeft += 1
			winner = "left"
			room.GameState.Ball.X = float64(CanvasWidth / 2)
			room.GameState.Ball.Y = float64(CanvasHeight / 2)
			room.GameState.Ball.Vx = -4.0
			room.GameState.Ball.Vy = 4.0
			room.GameState.PanYLeft = CanvasHeight/2 - PaddleHeight/2  // 250
			room.GameState.PanYRight = CanvasHeight/2 - PaddleHeight/2 // 250
		}

		room.Mutex.Unlock()

		if winner != "" {
			// Broadcast game over message
			gameOverMsg := Message{
				Type:   GameOverMsg,
				Winner: winner,
			}
			BroadcastToRoom(room, gameOverMsg)
			continue
		}

		// Broadcast game state to all clients in the room
		BroadcastToRoom(room, Message{
			Type:       UpdateMessage,
			LeftY:      room.GameState.PanYLeft,
			RightY:     room.GameState.PanYRight,
			BallX:      room.GameState.Ball.X,
			BallY:      room.GameState.Ball.Y,
			ScoreLeft:  room.GameState.ScoreLeft,
			ScoreRight: room.GameState.ScoreRight,
		})
	}
}

// ProcessMove updates the paddle position based on the player's move
func ProcessMove(room *GameRoom, player string, y int) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// Clamp Y position
	if y < 0 {
		y = 0
	}
	if y > MaxPaddleY {
		y = MaxPaddleY
	}

	if player == "left" {
		room.GameState.PanYLeft = y
	} else if player == "right" {
		room.GameState.PanYRight = y
	}

	// Optionally, broadcast the updated paddle position immediately
	// BroadcastToRoom(room, Message{
	//     Type:   "update_paddle",
	//     Player: player,
	//     Y:      &y,
	// })
}

// RestartGame resets the game state within the room and starts the game loop
func RestartGame(room *GameRoom) {
	room.Mutex.Lock()

	// Reset game state
	room.GameState.PanYLeft = CanvasHeight/2 - PaddleHeight/2
	room.GameState.PanYRight = CanvasHeight/2 - PaddleHeight/2
	room.GameState.Ball = Ball{
		X:  float64(CanvasWidth / 2),
		Y:  float64(CanvasHeight / 2),
		Vx: 4.0,
		Vy: 4.0,
	}
	room.GameState.ScoreLeft = 0
	room.GameState.ScoreRight = 0

	// Restart game loop
	room.Status = "active"
	if !room.IsLoopRunning {
		room.IsLoopRunning = true
		room.Mutex.Unlock()
		go RoomGameHandler(room)
	}
	room.Mutex.Unlock()
	// Broadcast restart game message with initial game state
	BroadcastToRoom(room, Message{
		Type:       RestartGameMsg,
		LeftY:      room.GameState.PanYLeft,
		RightY:     room.GameState.PanYRight,
		BallX:      room.GameState.Ball.X,
		BallY:      room.GameState.Ball.Y,
		ScoreLeft:  room.GameState.ScoreLeft,
		ScoreRight: room.GameState.ScoreRight,
	})
}

func sendErrorMessage(conn *websocket.Conn, errorDetail ErrorDetail) {
	errMsg := Message{
		Type:       ErrorMessage,
		Y:          nil,
		LeftY:      0,
		RightY:     0,
		BallX:      0,
		BallY:      0,
		Winner:     "",
		ScoreLeft:  0,
		ScoreRight: 0,
		RoomID:     "",
		Player:     "",
	}

	msgBytes, err := json.Marshal(errMsg)
	if err != nil {
		log.Printf("Failed to marshal error message: %v", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		log.Printf("Failed to send error message to client %v: %v", conn.RemoteAddr(), err)
	}
}

// ChangeRoom handles changing a player's room, removing them from the current room and assigning to a new one
func ChangeRoom(conn *websocket.Conn, currentRoom *GameRoom) {
	currentPlayer := removeCurrentPlayerFromTheRoom(conn, currentRoom)

	// Assign to new room
	newRoom := AssignRoom(conn)
	log.Printf("Player %s removed from room %s as %s", conn.RemoteAddr(), currentRoom.ID, currentPlayer)
	newPlayer := newRoom.Players[conn]

	// Send assign message with new room ID and player role
	assignMsg := Message{
		Type:   AssignMessage,
		RoomID: newRoom.ID,
		Player: newPlayer,
	}
	if err := conn.WriteJSON(assignMsg); err != nil {
		log.Printf("Error sending assign message during room change: %v", err)
		errorDetail := ErrorDetail{
			Error:   "ReadError",
			Details: "Error sending assign message during room change.",
		}
		sendErrorMessage(conn, errorDetail)
		return
	}

	// If new room becomes active, ensure game loop starts
	if newRoom.Status == "active" {
		newRoom.Mutex.Lock()
		if !newRoom.IsLoopRunning {
			newRoom.IsLoopRunning = true
			go RoomGameHandler(newRoom)
		}
		newRoom.Mutex.Unlock()

		// Notify all players in the new room about game start
		startGameMsg := Message{
			Type:       StartGameMsg,
			LeftY:      newRoom.GameState.PanYLeft,
			RightY:     newRoom.GameState.PanYRight,
			BallX:      newRoom.GameState.Ball.X,
			BallY:      newRoom.GameState.Ball.Y,
			ScoreLeft:  newRoom.GameState.ScoreLeft,
			ScoreRight: newRoom.GameState.ScoreRight,
		}
		BroadcastToRoom(newRoom, startGameMsg)
	}

	log.Printf("Player %s changed from room %s as %s to room %s as %s",
		conn.RemoteAddr(), currentRoom.ID, currentPlayer, newRoom.ID, newPlayer)
}

func removeCurrentPlayerFromTheRoom(conn *websocket.Conn, currentRoom *GameRoom) string {
	// Store player role before removal
	currentPlayer := currentRoom.Players[conn]

	// Remove player from current room
	currentRoom.Mutex.Lock()
	delete(currentRoom.Players, conn)

	// Reset game state and notify remaining player
	if len(currentRoom.Players) == 1 {
		currentRoom.Status = "waiting"
		// Stop the game loop by changing status
		if currentRoom.IsLoopRunning {
			currentRoom.Status = "waiting"
			// Game loop will exit on next tick due to status change
		}

		// Notify remaining player about room state change
		for remainingConn, remainingPlayer := range currentRoom.Players {
			stateChangeMsg := Message{
				Type:   RoomChangeMsg,
				RoomID: currentRoom.ID,
				Player: remainingPlayer,
			}
			if err := remainingConn.WriteJSON(stateChangeMsg); err != nil {
				log.Printf("Error notifying remaining player: %v", err)
			}
		}
	} else if len(currentRoom.Players) == 0 {
		// Delete room if empty
		roomsMutex.Lock()
		delete(rooms, currentRoom.ID)
		roomsMutex.Unlock()
		log.Printf("Deleted empty room %s", currentRoom.ID)

	}
	currentRoom.Mutex.Unlock()
	return currentPlayer
}

// CleanupRoom removes a player from a room and deletes the room if empty
func CleanupRoom(conn *websocket.Conn, room *GameRoom) {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	room.Mutex.Lock()
	delete(room.Players, conn)
	if len(room.Players) == 0 {
		// Delete room if empty
		delete(rooms, room.ID)
		log.Printf("Deleted empty room %s after cleanup", room.ID)
	} else {
		room.Status = "waiting"
	}
	room.Mutex.Unlock()
}

// BroadcastToRoom sends a message to all players in the room
func BroadcastToRoom(room *GameRoom, msg Message) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error marshaling message:", err)
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	for client := range room.Players {
		err := client.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			log.Println("Error broadcasting to client:", err)
			client.Close()
			delete(room.Players, client)
		}
	}
}

func main() {
	// Set up the WebSocket route
	http.HandleFunc("/ws", handleConnections)

	// Serve static files from the "public" directory
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// Start the server
	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
