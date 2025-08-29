package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MsgTypeCreateRoom   = 103
	MsgTypePlayerAction = 202
)

// send formats and sends a message to the WebSocket server.
func send(c *websocket.Conn, msgID uint16, data []byte) error {
	packet := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(packet[0:2], msgID)
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(data)))
	copy(packet[4:], data)

	return c.WriteMessage(websocket.BinaryMessage, packet)
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	log.Printf("Connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// Read loop
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			// Simple parsing of the message ID
			if len(message) < 4 {
				log.Printf("Received invalid packet of size %d", len(message))
				continue
			}
			msgID := binary.BigEndian.Uint16(message[0:2])
			data := message[4:]
			log.Printf("<-\\ RECV (ID: %d): %s", msgID, string(data)) // Corrected: \\ to \ for newline in log.Printf
		}
	}()

	// Send Create Room message automatically
	log.Println("Sending Create Room request...")
	if err := send(c, MsgTypeCreateRoom, []byte{}); err != nil {
		log.Println("Write error:", err)
		return
	}

	log.Println("Client started. Type 'spin' and press Enter to play.")

	// Write loop
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection.")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close error:", err)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		default:
			// Non-blocking read from stdin
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)

			if text == "spin" {
				action := map[string]string{"type": "spin"}
				actionData, _ := json.Marshal(action)
				if err := send(c, MsgTypePlayerAction, actionData); err != nil {
					log.Println("Write error:", err)
					return
				}
				log.Println("-> SENT: spin action")
			}
		}
	}
}
