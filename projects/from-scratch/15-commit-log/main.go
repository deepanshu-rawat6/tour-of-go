package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// --- Segment: append-only log file + index ---

type Segment struct {
	logFile    *os.File
	indexFile  *os.File
	baseOffset uint64
	nextOffset uint64
	mu         sync.Mutex
}

func NewSegment(dir string, baseOffset uint64) (*Segment, error) {
	os.MkdirAll(dir, 0755)
	logPath := filepath.Join(dir, fmt.Sprintf("%020d.log", baseOffset))
	idxPath := filepath.Join(dir, fmt.Sprintf("%020d.index", baseOffset))

	logF, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	idxF, err := os.OpenFile(idxPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// Count existing records from index
	idxStat, _ := idxF.Stat()
	count := uint64(idxStat.Size()) / 12 // 4 bytes offset + 8 bytes position

	return &Segment{
		logFile:    logF,
		indexFile:  idxF,
		baseOffset: baseOffset,
		nextOffset: baseOffset + count,
	}, nil
}

func (s *Segment) Append(data []byte) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current position in log file
	pos, _ := s.logFile.Seek(0, io.SeekEnd)

	// Write record: [length:4][crc32:4][data:N]
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(data)))
	binary.BigEndian.PutUint32(header[4:8], crc32.ChecksumIEEE(data))
	s.logFile.Write(header)
	s.logFile.Write(data)

	// Write index entry: [relative_offset:4][position:8]
	entry := make([]byte, 12)
	binary.BigEndian.PutUint32(entry[0:4], uint32(s.nextOffset-s.baseOffset))
	binary.BigEndian.PutUint64(entry[4:12], uint64(pos))
	s.indexFile.Write(entry)

	offset := s.nextOffset
	s.nextOffset++
	return offset, nil
}

func (s *Segment) Read(offset uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	relOffset := offset - s.baseOffset
	// Read index entry
	indexPos := int64(relOffset * 12)
	entry := make([]byte, 12)
	if _, err := s.indexFile.ReadAt(entry, indexPos); err != nil {
		return nil, fmt.Errorf("offset %d not found", offset)
	}
	filePos := binary.BigEndian.Uint64(entry[4:12])

	// Read record from log
	header := make([]byte, 8)
	if _, err := s.logFile.ReadAt(header, int64(filePos)); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[0:4])
	expectedCRC := binary.BigEndian.Uint32(header[4:8])

	data := make([]byte, length)
	s.logFile.ReadAt(data, int64(filePos)+8)

	if crc32.ChecksumIEEE(data) != expectedCRC {
		return nil, fmt.Errorf("CRC mismatch at offset %d", offset)
	}
	return data, nil
}

func (s *Segment) Close() {
	s.logFile.Close()
	s.indexFile.Close()
}

// --- Partition: ordered sequence of segments ---

type Partition struct {
	dir     string
	segment *Segment
	mu      sync.RWMutex
}

func NewPartition(dir string) (*Partition, error) {
	seg, err := NewSegment(dir, 0)
	if err != nil {
		return nil, err
	}
	return &Partition{dir: dir, segment: seg}, nil
}

func (p *Partition) Append(data []byte) (uint64, error) {
	return p.segment.Append(data)
}

func (p *Partition) Read(offset uint64) ([]byte, error) {
	return p.segment.Read(offset)
}

// --- Broker: manages topics and partitions ---

type Broker struct {
	dataDir    string
	partitions map[string]*Partition // "topic/partition" → Partition
	mu         sync.RWMutex
}

func NewBroker(dataDir string) *Broker {
	return &Broker{dataDir: dataDir, partitions: make(map[string]*Partition)}
}

func (b *Broker) getPartition(topic string, partition int) (*Partition, error) {
	key := fmt.Sprintf("%s/%d", topic, partition)
	b.mu.RLock()
	p, ok := b.partitions[key]
	b.mu.RUnlock()
	if ok {
		return p, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	// Double-check
	if p, ok = b.partitions[key]; ok {
		return p, nil
	}

	dir := filepath.Join(b.dataDir, "topics", topic, fmt.Sprintf("partition-%d", partition))
	p, err := NewPartition(dir)
	if err != nil {
		return nil, err
	}
	b.partitions[key] = p
	return p, nil
}

func (b *Broker) Produce(topic string, partition int, data []byte) (uint64, error) {
	p, err := b.getPartition(topic, partition)
	if err != nil {
		return 0, err
	}
	return p.Append(data)
}

func (b *Broker) Consume(topic string, partition int, offset uint64) ([]byte, error) {
	p, err := b.getPartition(topic, partition)
	if err != nil {
		return nil, err
	}
	return p.Read(offset)
}

// --- TCP Server ---

func handleConn(conn net.Conn, broker *Broker) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch strings.ToUpper(parts[0]) {
		case "PRODUCE":
			// PRODUCE <topic> <partition> <message>
			if len(parts) < 4 {
				fmt.Fprintf(conn, "ERR usage: PRODUCE <topic> <partition> <message>\r\n")
				continue
			}
			topic := parts[1]
			partition, _ := strconv.Atoi(parts[2])
			msg := strings.Join(parts[3:], " ")
			offset, err := broker.Produce(topic, partition, []byte(msg))
			if err != nil {
				fmt.Fprintf(conn, "ERR %v\r\n", err)
			} else {
				fmt.Fprintf(conn, "OK %d\r\n", offset)
			}

		case "CONSUME":
			// CONSUME <topic> <partition> <offset>
			if len(parts) < 4 {
				fmt.Fprintf(conn, "ERR usage: CONSUME <topic> <partition> <offset>\r\n")
				continue
			}
			topic := parts[1]
			partition, _ := strconv.Atoi(parts[2])
			offset, _ := strconv.ParseUint(parts[3], 10, 64)
			data, err := broker.Consume(topic, partition, offset)
			if err != nil {
				fmt.Fprintf(conn, "ERR %v\r\n", err)
			} else {
				fmt.Fprintf(conn, "DATA %s\r\n", string(data))
			}

		default:
			fmt.Fprintf(conn, "ERR unknown command: %s\r\n", parts[0])
		}
	}
}

func main() {
	dataDir := "./data"
	if d := os.Getenv("DATA_DIR"); d != "" {
		dataDir = d
	}
	addr := ":9092"
	if a := os.Getenv("ADDR"); a != "" {
		addr = a
	}

	broker := NewBroker(dataDir)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Printf("commit-log broker listening on %s (data: %s)\n", addr, dataDir)
	fmt.Println("Commands: PRODUCE <topic> <partition> <msg> | CONSUME <topic> <partition> <offset>")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn, broker)
	}
}
