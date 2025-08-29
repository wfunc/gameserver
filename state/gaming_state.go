package state

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
	// This is a simple tick-based update. For real-time games, you might want a more sophisticated loop.
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
	switch s.Room.GetGameType() {
	case "slot_machine":
		s.GameData = s.initializeSlotMachineData()
	default:
		s.GameData = make(map[string]interface{})
	}
}

func (s *GamingState) initializeSlotMachineData() map[string]interface{} {
	return map[string]interface{}{
		"reels":      [3]int{0, 0, 0},
		"spin_count": 0,
	}
}

func (s *GamingState) notifyGameStart() {
	startMsg := map[string]interface{}{
		"game_data": s.GameData,
	}
	data, _ := json.Marshal(startMsg)
	s.Room.Broadcast(network.MsgTypeGameStart, data)
}

func (s *GamingState) updateSlotMachineLogic(player Player) {
	gameData, ok := s.GameData.(map[string]interface{})
	if !ok {
		return
	}

	// Simulate spinning the reels
	reels := [3]int{rand.Intn(8), rand.Intn(8), rand.Intn(8)}
	gameData["reels"] = reels
	gameData["spin_count"] = gameData["spin_count"].(int) + 1

	// Calculate the result of this spin
	result := s.calculateSlotResult(reels)
	gameData["last_result"] = result

	s.GameData = gameData

	// Immediately sync the new state to all players
	s.syncGameState()
}

func (s *GamingState) syncGameState() {
	data, err := json.Marshal(s.GameData)
	if err != nil {
		logger.Log.Errorf("Error marshalling sync message: %v", err)
		return
	}
	s.Room.Broadcast(network.MsgTypeGameSync, data)
}

func (s *GamingState) endGame() {
	logger.Log.Infof("房间 %s 游戏结束", s.Room.GetID())
	s.calculateFinalResults()
	s.notifyGameEnd()

	// TODO: Transition to a SettlementState
	// settlementState := NewSettlementState(s.Room, s.Results)
	// s.Room.ChangeState(settlementState)
}

func (s *GamingState) calculateFinalResults() {
	switch s.Room.GetGameType() {
	case "slot_machine":
		s.Results = s.calculateSlotMachineResults()
	}
}

func (s *GamingState) calculateSlotMachineResults() map[string]interface{} {
	// For a slot machine, the result is per-spin, but we could have a final tally.
	// This is just a placeholder.
	return map[string]interface{}{"final_payout": 0}
}

func (s *GamingState) notifyGameEnd() {
	data, _ := json.Marshal(s.Results)
	s.Room.Broadcast(network.MsgTypeGameEnd, data)
}

func (s *GamingState) cleanupGameData() {
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
