package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
)

type bilibiliPacket struct {
	packetLen uint32
	headerLen uint16
	ver       uint16
	op        uint32
	seq       uint32
	data      []byte
}

func encode(data []byte, op int) []byte {
	packetLen := 16 + len(data)
	buf := bytes.NewBuffer(make([]byte, 0, packetLen))
	_ = binary.Write(buf, binary.BigEndian, uint32(packetLen))
	_ = binary.Write(buf, binary.BigEndian, uint16(16))
	_ = binary.Write(buf, binary.BigEndian, uint16(2))
	_ = binary.Write(buf, binary.BigEndian, uint32(op))
	_ = binary.Write(buf, binary.BigEndian, uint32(1))
	_ = binary.Write(buf, binary.BigEndian, data)
	return buf.Bytes()
}

func decode(data []byte) []bilibiliPacket {
	var result []bilibiliPacket
	buf := bytes.NewBuffer(data)
	for i := 0; buf.Len() > 0; i++ {
		packet := bilibiliPacket{}
		_ = binary.Read(buf, binary.BigEndian, &packet.packetLen)
		_ = binary.Read(buf, binary.BigEndian, &packet.headerLen)
		_ = binary.Read(buf, binary.BigEndian, &packet.ver)
		_ = binary.Read(buf, binary.BigEndian, &packet.op)
		_ = binary.Read(buf, binary.BigEndian, &packet.seq)
		dataLen := packet.packetLen - uint32(packet.headerLen)
		packet.data = make([]byte, dataLen)
		_ = binary.Read(buf, binary.BigEndian, &packet.data)
		result = append(result, packet)
	}
	return result
}

func encodeRoomInit(roomId int) []byte {
	json2 := simplejson.New()
	json2.Set("uid", 0)
	json2.Set("roomid", roomId)
	data, _ := json2.Encode()
	return data
}

func decodeJSON(data []byte) *simplejson.Json {
	json2, _ := simplejson.NewJson(data)
	return json2
}

func decodePopular(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

func main() {
	defer log.Println("Bilibili Live WebSocket Go Client Close")
	log.Println("Bilibili Live WebSocket Go Client Start")

	// Ctrl-C interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Create websocket
	c, _, err := websocket.DefaultDialer.Dial("wss://broadcastlv.chat.bilibili.com:2245/sub", nil)
	if err != nil {
		log.Fatal("Connect Error:", err)
		return
	}
	defer c.Close()

	// Send hello
	err = c.WriteMessage(websocket.TextMessage, encode(encodeRoomInit(2029840), 7))
	if err != nil {
		log.Println("Write Error:", err)
		return
	}
	log.Println("Send:", "Hello")

	// Heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			err := c.WriteMessage(websocket.TextMessage, encode([]byte{}, 2))
			if err != nil {
				log.Println("Write Error:", err)
				return
			}
			log.Println("Send:", "Heartbeat")
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
			packets := decode(message)
			for i := range packets {
				packet := packets[i]
				switch packet.op {
				case 8:
					log.Println("Received:", "Hello Response")
				case 3:
					popular := decodePopular(packet.data)
					log.Println("Received:", "Heartbeat Response.", "Popular:", popular)
				case 5:
					body := decodeJSON(packet.data)
					cmd, _ := body.Get("cmd").String()
					switch cmd {
					case "DANMU_MSG":
						text, _ := body.Get("info").GetIndex(1).String()
						uid, _ := body.Get("info").GetIndex(2).GetIndex(0).Int()
						username, _ := body.Get("info").GetIndex(2).GetIndex(1).String()
						medal, _ := body.Get("info").GetIndex(3).GetIndex(1).String()
						medalLevel, _ := body.Get("info").GetIndex(3).GetIndex(0).Int()
						userLevel, _ := body.Get("info").GetIndex(4).GetIndex(0).Int()
						log.Println("Received:", cmd, username, ":", text)
						log.Println("Received Danmaku User Info:", "[", medal, "|", medalLevel, "]", "[ UL", userLevel, "]", "uid =", uid)
					case "SEND_GIFT":
						username, _ := body.Get("data").Get("uname").String()
						action, _ := body.Get("data").Get("action").String()
						num, _ := body.Get("data").Get("num").Int()
						giftName, _ := body.Get("data").Get("giftName").String()
						price, _ := body.Get("data").Get("price").Int()
						totalCoin, _ := body.Get("data").Get("total_coin").Int()
						coinType, _ := body.Get("data").Get("coin_type").String()
						uid, _ := body.Get("data").Get("uid").Int()
						remain, _ := body.Get("data").Get("remain").Int()
						silver, _ := body.Get("data").Get("silver").Int()
						gold, _ := body.Get("data").Get("gold").Int()
						log.Println("Received:", cmd, username, action, num, giftName)
						log.Println("Received Gift:", num, "x", price, "=", totalCoin, coinType)
						log.Println("Received Gift User Info:", "uid =", uid, "remain =", remain, "silver =", silver, "gold =", gold)
					case "WELCOME":
						username, _ := body.Get("data").Get("uname").String()
						log.Println("Received:", cmd, username)
					default:
						pretty, _ := body.EncodePretty()
						log.Println("Received:", "Unknown Command", string(pretty))
					}
				default:
					log.Println("Unknown Packet:", packet)
				}
			}
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
