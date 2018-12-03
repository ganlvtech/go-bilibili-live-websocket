package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func encode(data []byte, op int) []byte {
	packetLen := 16 + len(data)
	buf := bytes.NewBuffer(make([]byte, 0, packetLen))
	binary.Write(buf, binary.BigEndian, uint32(packetLen))
	binary.Write(buf, binary.BigEndian, uint16(16))
	binary.Write(buf, binary.BigEndian, uint16(2))
	binary.Write(buf, binary.BigEndian, uint32(op))
	binary.Write(buf, binary.BigEndian, uint32(1))
	binary.Write(buf, binary.BigEndian, data)
	return buf.Bytes()
}

func main() {
	defer log.Println("Bilibili Live WebSocket Go Client Close")
	log.Println("Bilibili Live WebSocket Go Client Start")

	// Ctrl-C interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Create webscoket
	c, _, err := websocket.DefaultDialer.Dial("wss://broadcastlv.chat.bilibili.com:2245/sub", nil)
	if err != nil {
		log.Fatal("Connect Error:", err)
		return
	}
	defer c.Close()

	// Send hello
	data := []byte("{\"uid\":0,\"roomid\":23058}")
	c.WriteMessage(websocket.TextMessage, encode(data, 7))
	log.Println("Send:", "Hello")

	// Heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			err := c.WriteMessage(websocket.TextMessage, encode([]byte{}, 2))
			log.Println("Send:", "Heartbeat")
			if err != nil {
				log.Println("Write Error:", err)
			}
		}
	}()

	// Receive Message
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Read Error:", err)
				return
			}
			log.Printf("Received: %T %v", message, message)
		}
	}()

	// Wait For Ctrl-C
	<-interrupt
	log.Println("Keyboard Interrupt")
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("Write Close Message Error:", err)
		return
	}
}
