package proto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSetCommand(t *testing.T) {
	cmd := &CommandSet{
		Key:   []byte("Foo"),
		Value: []byte("Bar"),
		TTL:   2,
	}
	r := bytes.NewReader(cmd.Bytes())
	pcmd, err := ParseCommand(r)
	assert.Equal(t, cmd, pcmd)
	assert.Nil(t, err)
}

func TestParseGetCommand(t *testing.T) {
	cmd := &CommandGet{
		Key: []byte("Foo"),
	}
	r := bytes.NewReader(cmd.Bytes())
	pcmd, err := ParseCommand(r)
	assert.Equal(t, cmd, pcmd)
	assert.Nil(t, err)
}

func BenchmarkParseCommand(b *testing.B) {
	cmd := &CommandSet{
		Key:   []byte("Foo"),
		Value: []byte("Bar"),
		TTL:   2,
	}
	r := bytes.NewReader(cmd.Bytes())
	for i := 0; i < b.N; i++ {
		ParseCommand(r)
		r.Seek(0, 0)
	}
}
