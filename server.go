package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"cachego/cache"
)

var (
	WtGrp    sync.WaitGroup
	RootTrie *cache.Trie

	BufferSize = 1024
	BufferPool = newSyncPool(BufferSize)

	WorkerCount = runtime.NumCPU()
	QueueSize   = 2500
	packetQueue = make(chan Packet, QueueSize)

	ServerConn *net.UDPConn
)

type Packet struct {
	sourceAddr *net.UDPAddr
	buffer     *[]byte
	bytesCount int
}

func newSyncPool(bsz int) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			lst := make([]byte, bsz)
			return &lst
		},
	}
}

func startServer(serverSocket, duration string) {
	defaultDuration, _ := str2IntDefaultMinMax(duration, 300, 300, 900)
	RootTrie = cache.NewTrie(time.Duration(defaultDuration) * time.Second)

	fmt.Print("Starting server...")
	startListening(serverSocket)
	startWorkers()
	udpLoopWorkers()
	fmt.Println("Success: UDP", ServerConn.LocalAddr().String())
}

func startListening(ipsocket string) {
	socket, err := net.ResolveUDPAddr("udp", ipsocket)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	ServerConn, err = net.ListenUDP("udp", socket)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}
}

func str2IntDefaultMinMax[T int | int8 | int16 | int32 | int64](s string, d, minN, maxN T) (T, bool) {
	out, ok := str2IntCheck[T](s)
	if ok {
		if out < minN || out > maxN {
			return d, false
		}
		return out, true
	}
	return d, false
}

func str2IntCheck[T int | int8 | int16 | int32 | int64](s string) (T, bool) {
	var out T
	if len(s) == 0 {
		return out, false
	}

	idx := 0
	isN := s[idx] == '-'
	if isN {
		idx++
		if len(s) == 1 {
			return out, false
		}
	} else if s[idx] == '+' {
		idx++
	}

	for i := idx; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return out, false
		}
		out = out*10 + T(s[i]-'0')
	}

	if isN {
		out = -out
	}
	return out, true
}

func str2Int[T int | int8 | int16 | int32 | int64](s string) T {
	var out T

	if len(s) == 0 {
		return out
	}
	idx := 0
	isN := s[idx] == '-'

	if isN {
		idx++
	}
	for i := idx; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return out
		}
		out = out*10 + T(s[i]-'0')
	}

	if isN {
		return -out
	}
	return out
}

func startWorkers() {
	WtGrp.Add(WorkerCount)

	for range WorkerCount {
		go worker(packetQueue)
	}
}

func udpLoopWorkers() {
	WtGrp.Add(1)

	defer func() {
		WtGrp.Done()

		if r := recover(); r != nil {
			udpLoopWorkers()
		}
	}()

	go func() {
		for {
			buf, _ := BufferPool.Get().(*[]byte)
			n, addr, err := ServerConn.ReadFromUDP(*buf)
			if err != nil {
				fmt.Println(err)
				continue
			}
			packetQueue <- Packet{sourceAddr: addr, buffer: buf, bytesCount: n}
		}
	}()
}

func worker(queue <-chan Packet) {
	defer WtGrp.Done()

	for packet := range queue {
		processPacket(packet)
	}
}

func processPacket(packet Packet) {
	pdu := (*packet.buffer)[:packet.bytesCount]
	if len(pdu) > 0 {
		processPDU(pdu, packet.sourceAddr)
	}

	BufferPool.Put(packet.buffer)
}

func processPDU(pdu []byte, src *net.UDPAddr) {
	data := string(pdu)
	items := strings.Split(data, "||")

	switch len(items) {
	case 3:
		max := str2Int[int](items[2])
		RootTrie.InsertConditionalDefaultDuration(items[0], items[1], max)
	case 4:
		dur, _ := str2IntDefaultMinMax(items[2], 300, 300, 900)
		max := str2Int[int](items[3])
		RootTrie.InsertConditional(items[0], items[1], time.Duration(dur)*time.Second, max)
	default:
		log.Println("invalid PDU:", data, "from:", src.String())
	}
}
