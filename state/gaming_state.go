package state

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/wfunc/gameserver/logger"
	"github.com/wfunc/gameserver/network"
)

// Action represents a player action that can be unmarshalled from a packet.
type Action struct {
	Type string `json:"type"`
}

// GamingState 游戏进行状态
type GamingState struct {
	RoomStateBase
	GameDuration  time.Duration
	RemainingTime time.Duration
	GameData      interface{}
	Results       map[string]interface{}
	TimerID       int64
	dataMutex     sync.RWMutex // Mutex to protect GameData and Results
}

// NewGamingState 创建新的游戏状态
func NewGamingState(room RoomContext, duration time.Duration) *GamingState {
	return &GamingState{
		RoomStateBase: RoomStateBase{
			ID:   "gaming",
			Room: room,
		},
		GameDuration:  duration,
		RemainingTime: duration,
		Results:       make(map[string]interface{}),
	}
}

// HandleAction handles actions from players.
func (s *GamingState) HandleAction(player Player, actionData []byte) error {
	var action Action
	if err := json.Unmarshal(actionData, &action); err != nil {
		return fmt.Errorf("failed to unmarshal action data: %w", err)
	}

	if s.Room.GetGameType() == "slot_machine" {
		if action.Type == "spin" {
			logger.Log.Infof("Player %s triggered a spin in room %s", player.GetID(), s.Room.GetID())
			s.updateSlotMachineLogic(player)
		}
	}
	return nil
}

// OnEnter 进入游戏状态
func (s *GamingState) OnEnter() {
	logger.Log.Infof("房间 %s 进入游戏状态，游戏时长: %v", s.Room.GetID(), s.GameDuration)
	s.initializeGameData()
	s.notifyGameStart()
}

// OnExit 退出游戏状态
func (s *GamingState) OnExit() {
	logger.Log.Infof("房间 %s 退出游戏状态", s.Room.GetID())
	s.cleanupGameData()
}

// OnUpdate 游戏状态更新
func (s *GamingState) OnUpdate() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	s.RemainingTime -= 100 * time.Millisecond
	if s.RemainingTime <= 0 {
		s.endGame()
		return
	}
}

// GetID 获取状态ID
func (s *GamingState) GetID() string {
	return s.ID
}

func (s *GamingState) initializeGameData() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	switch s.Room.GetGameType() {
	case "slot_machine":
		s.GameData = s.initializeSlotMachineData()
	default:
		s.GameData = make(map[string]interface{})
	}
}

func (s *GamingState) initializeSlotMachineData() map[string]interface{} {
	return map[string]interface{}{
		"reels":       [3]int{0, 0, 0},
		"spin_count":  0,
		"last_result": nil,
	}
}

func (s *GamingState) notifyGameStart() {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	logger.Log.Debugf("Data before marshal in notifyGameStart: %+v", s.GameData)
	data, err := json.Marshal(s.GameData)
	if err != nil {
		logger.Log.Errorf("Failed to marshal game start data: %v", err)
		return
	}
	logger.Log.Infof("Broadcasting GameStart with data: %s", string(data))
	s.Room.Broadcast(network.MsgTypeGameStart, data)
}

func (s *GamingState) updateSlotMachineLogic(player Player) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	gameData, ok := s.GameData.(map[string]interface{})
	if !ok {
		return
	}

	reels := [3]int{rand.Intn(8), rand.Intn(8), rand.Intn(8)}
	gameData["reels"] = reels
	gameData["spin_count"] = gameData["spin_count"].(int) + 1

	result := s.calculateSlotResult(reels)
	gameData["last_result"] = result

	s.GameData = gameData

	s.syncGameState()
}

func (s *GamingState) syncGameState() {
	// This function is called from within updateSlotMachineLogic, which already holds the lock.
	logger.Log.Debugf("Data before marshal in syncGameState: %+v", s.GameData)
	data, err := json.Marshal(s.GameData)
	if err != nil {
		logger.Log.Errorf("Error marshalling sync message: %v", err)
		return
	}
	logger.Log.Infof("Broadcasting GameSync with data: %s", string(data))
	s.Room.Broadcast(network.MsgTypeGameSync, data)
}

func (s *GamingState) endGame() {
	// This function is called from OnUpdate, which already holds the lock.
	logger.Log.Infof("房间 %s 游戏结束", s.Room.GetID())
	s.calculateFinalResults()
	s.notifyGameEnd()

	waitingState := NewWaitingState(s.Room)
	s.Room.ChangeState(waitingState)
}

func (s *GamingState) calculateFinalResults() {
	switch s.Room.GetGameType() {
	case "slot_machine":
		s.Results = s.calculateSlotMachineResults()
	}
}

func (s *GamingState) calculateSlotMachineResults() map[string]interface{} {
	gameData, ok := s.GameData.(map[string]interface{})
	if !ok {
		return map[string]interface{}{"error": "invalid game data"}
	}

	finalResult := map[string]interface{}{
		"final_spin_count": gameData["spin_count"],
		"last_win":         nil,
	}

	if lastResult, ok := gameData["last_result"].(map[string]interface{}); ok {
		finalResult["last_win"] = lastResult["win"]
	}

	return finalResult
}

func (s *GamingState) notifyGameEnd() {
	logger.Log.Debugf("Data before marshal in notifyGameEnd: %+v", s.Results)
	data, err := json.Marshal(s.Results)
	if err != nil {
		logger.Log.Errorf("Failed to marshal game end data: %v", err)
		return
	}
	logger.Log.Infof("Broadcasting GameEnd with data: %s", string(data))
	s.Room.Broadcast(network.MsgTypeGameEnd, data)
}

func (s *GamingState) cleanupGameData() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.GameData = nil
	s.Results = nil
}

func (s *GamingState) calculateSlotResult(reels [3]int) map[string]interface{} {
	win := reels[0] == reels[1] && reels[1] == reels[2]
	payout := 0
	if win {
		switch reels[0] {
		case 7: // 7-7-7
			payout = 1000
		default:
			payout = 100
		}
	}
	return map[string]interface{}{
		"win":     win,
		"payout":  payout,
		"symbols": reels,
	}
}
