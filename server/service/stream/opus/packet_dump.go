package opus

import (
	"encoding/binary"
	"os"
	"sync"
	"time"
)

const opusPacketDumpMagic = "NVOPDMP1"

type opusPacketDumper struct {
	once sync.Once
	mu   sync.Mutex
	file *os.File
}

func (d *opusPacketDumper) write(env string, sequence uint64, duration time.Duration, payload []byte) {
	d.once.Do(func() {
		path := os.Getenv(env)
		if path == "" {
			return
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return
		}
		if _, err := file.WriteString(opusPacketDumpMagic); err != nil {
			_ = file.Close()
			return
		}
		d.file = file
	})
	if d.file == nil || len(payload) == 0 {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	header := [4]uint64{
		uint64(time.Now().UnixNano()),
		sequence,
		uint64(duration / time.Microsecond),
		uint64(len(payload)),
	}
	for _, value := range header {
		if err := binary.Write(d.file, binary.LittleEndian, value); err != nil {
			_ = d.file.Close()
			d.file = nil
			return
		}
	}
	if _, err := d.file.Write(payload); err != nil {
		_ = d.file.Close()
		d.file = nil
	}
}

var microphoneOpusDump opusPacketDumper
