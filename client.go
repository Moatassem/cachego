package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type (
	UDPClient struct {
		conn         *net.UDPConn
		responseMap  map[string]chan bool
		responseLock sync.Mutex
	}

	CacheAction = int
)

const (
	CacheActionDelete = -1
)

func NewUDPClient(serverAddr string) (*UDPClient, error) {
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	client := &UDPClient{
		conn:        conn,
		responseMap: make(map[string]chan bool),
	}

	go client.listenForResponses()

	return client, nil
}

func (c *UDPClient) listenForResponses() {
	buf := make([]byte, 1024)
	for {
		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading response:", err)
			continue
		}

		data := string(buf[:n])
		items := strings.Split(data, "||")

		correlationID := items[0] + "||" + items[1]

		c.responseLock.Lock()
		if ch, ok := c.responseMap[correlationID]; ok {
			flag, _ := strconv.ParseBool(items[2])
			ch <- flag
		}
		c.responseLock.Unlock()
	}
}

func (c *UDPClient) InvokeCacheRequest(correlationID string, action CacheAction, timeout time.Duration) (bool, error) {
	responseChan := make(chan bool, 1)

	c.responseLock.Lock()
	c.responseMap[correlationID] = responseChan
	c.responseLock.Unlock()

	defer func() {
		c.responseLock.Lock()
		defer c.responseLock.Unlock()

		close(responseChan)
		delete(c.responseMap, correlationID)
	}()

	data := fmt.Sprintf("%s||%d", correlationID, action)

	_, err := c.conn.Write([]byte(data))
	if err != nil {
		return false, err
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		return false, fmt.Errorf("timeout waiting for response")
	}
}
