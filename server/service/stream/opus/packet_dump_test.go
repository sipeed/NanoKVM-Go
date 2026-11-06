package opus

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpusPacketDumperWritesPortableRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microphone.nvopus")
	t.Setenv("TEST_OPUS_PACKET_DUMP", path)
	payload := []byte{0xfc, 0x7d, 0x0d, 0x93}
	dumper := &opusPacketDumper{}
	dumper.write("TEST_OPUS_PACKET_DUMP", 42, 20*time.Millisecond, payload)
	dumper.mu.Lock()
	if dumper.file != nil {
		if err := dumper.file.Close(); err != nil {
			t.Fatal(err)
		}
		dumper.file = nil
	}
	dumper.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(opusPacketDumpMagic)+32+len(payload) {
		t.Fatalf("dump length=%d", len(data))
	}
	if string(data[:8]) != opusPacketDumpMagic {
		t.Fatalf("dump magic=%q", data[:8])
	}
	header := data[8:40]
	if binary.LittleEndian.Uint64(header[8:16]) != 42 {
		t.Fatal("sequence was not encoded as little endian uint64")
	}
	if binary.LittleEndian.Uint64(header[16:24]) != 20_000 {
		t.Fatal("duration was not encoded in microseconds")
	}
	if binary.LittleEndian.Uint64(header[24:32]) != uint64(len(payload)) {
		t.Fatal("payload length was not encoded")
	}
	if !bytes.Equal(data[40:], payload) {
		t.Fatalf("payload=%x", data[40:])
	}
}
