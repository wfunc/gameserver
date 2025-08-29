package server

import (
	"encoding/json"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wfunc/gameserver/broadcast"
	"github.com/wfunc/gameserver/logger"
	"github.com/wfunc/gameserver/network"
	"github.com/wfunc/gameserver/persistence"
	"github.com/wfunc/gameserver/room"
	"github.com/wfunc/gameserver/services"
	"github.com/wfunc/gameserver/session"
	gameserver_rpc "github.com/wfunc/gameserver/rpc"
)

type GameServer struct {
	addr           string
	upgrader       websocket.Upgrader
	roomManager    *room.Manager
	sessionManager *session.Manager
	playerService  *services.PlayerService
	broadcaster    broadcast.Broadcaster
	rpcServer      *gameserver_rpc.Server
	mutex          sync.Mutex
	shutdownChan   chan struct{}
}

func NewGameServer(addr, rpcAddr string, db persistence.Database) *GameServer {
	s := &GameServer{
		addr:           addr,
		roomManager:    room.NewRoomManager(),
		sessionManager: session.NewManager(),
		playerService:  services.NewPlayerService(db),
		shutdownChan:   make(chan struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有跨域请求
			},
		},
	}

	// 初始化广播器
	s.broadcaster = broadcast.NewRoomBroadcaster(s.roomManager, s.sessionManager)

	// 初始化RPC服务器
	rpcServer, err := gameserver_rpc.NewServer(rpcAddr)
	if err != nil {
		logger.Log.Fatalf("Failed to create RPC server: %v", err)
	}
	s.rpcServer = rpcServer

	// 注册RPC服务
	gameService := gameserver_rpc.NewGameService(s.playerService)
	rpc.Register(gameService)

	return s
}

func (s *GameServer) Start() error {
	go s.rpcServer.Start()

	http.HandleFunc("/ws", s.handleWebSocket)
	logger.Log.Infof("Game server listening on %s", s.addr)
	return http.ListenAndServe(s.addr, nil)
}

func (s *GameServer) Shutdown() {
	close(s.shutdownChan)
	s.rpcServer.Stop()
}

func (s *GameServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Infof("Failed to upgrade connection: %v", err)
		return
	}
	s.handleConnection(conn)
}

func (s *GameServer) handleConnection(conn *websocket.Conn) {
	wsConn := network.NewWSConnection(conn)
	sess := session.NewSession(uuid.New().String(), wsConn)
	s.sessionManager.Add(sess)

	logger.Log.Infof("New connection from %s, session ID: %s", wsConn.RemoteAddr(), sess.GetID())

	defer func() {
		logger.Log.Infof("Connection closed from %s, session ID: %s", wsConn.RemoteAddr(), sess.GetID())
		s.sessionManager.Remove(sess.GetID())
		if sess.RoomID != "" {
			s.roomManager.RemoveRoom(sess.RoomID)
		}
		wsConn.Close()
	}()

	for {
		select {
		case <-s.shutdownChan:
			return
		default:
			packet, err := wsConn.ReadPacket()
			if err != nil {
				return
			}
			s.handlePacket(sess, packet)
		}
	}
}

func (s *GameServer) handlePacket(sess *session.Session, packet *network.Packet) {
	switch packet.MsgID {
	case network.MsgTypeHeartbeat:
		sess.LastActive = time.Now()
	case network.MsgTypeCreateRoom:
		s.handleCreateRoom(sess, packet)
	case network.MsgTypeJoinRoom:
		s.handleJoinRoom(sess, packet)
	case network.MsgTypeLeaveRoom:
		s.handleLeaveRoom(sess, packet)
	case network.MsgTypePlayerAction:
		s.handleGameAction(sess, packet)
	default:
		logger.Log.Infof("Unknown message type: %d", packet.MsgID)
	}
}

func (s *GameServer) handleCreateRoom(session *session.Session, packet *network.Packet) {
	roomID := uuid.New().String()
	room := s.roomManager.CreateRoom(roomID, "New Room", "default_game", 4, s.broadcaster)
	room.AddPlayer(session)

	logger.Log.Infof("Session %s created room %s", session.GetID(), roomID)

	resp := map[string]string{"room_id": roomID}
	data, _ := json.Marshal(resp)
	session.Send(network.MsgTypeCreateRoom, data)
}

func (s *GameServer) handleJoinRoom(session *session.Session, packet *network.Packet) {
	var req map[string]string
	if err := json.Unmarshal(packet.Data, &req); err != nil {
		return
	}
	roomID := req["room_id"]

	room, exists := s.roomManager.GetRoom(roomID)
	if !exists {
		return
	}

	if room.AddPlayer(session) {
		logger.Log.Infof("Session %s joined room %s", session.GetID(), roomID)
	} else {
		// 房间已满
	}
}

func (s *GameServer) handleLeaveRoom(session *session.Session, packet *network.Packet) {
	if session.RoomID != "" {
		s.roomManager.GetRoom(session.RoomID)
	}
}

func (s *GameServer) handleGameAction(session *session.Session, packet *network.Packet) {
	if session.RoomID == "" {
		logger.Log.Warnf("Session %s sent game action but is not in a room", session.GetID())
		return
	}

	room, exists := s.roomManager.GetRoom(session.RoomID)
	if !exists {
		logger.Log.Errorf("Room %s not found for session %s", session.RoomID, session.GetID())
		return
	}

	currentState := room.StateMachine.GetCurrentState()
	if currentState == nil {
		logger.Log.Errorf("Room %s has a nil state", room.GetID())
		return
	}

	if err := currentState.HandleAction(session, packet.Data); err != nil {
		logger.Log.Errorf("Error handling action in room %s: %v", room.GetID(), err)
	}
}