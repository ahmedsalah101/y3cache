package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Status byte

func (s Status) String() string {
	switch s {
	case StatusError:
		return "ERR"
	case StatusOK:
		return "OK"
	case StatusKeyNotFound:
		return "KEYNOTFOUND"
	default:
		return "NONE"
	}
}

const (
	StatusNone Status = iota
	StatusOK
	StatusError
	StatusKeyNotFound
)

type CommandB byte

const (
	CmdNone CommandB = iota
	CmdSet
	CmdGet
	CmdDel
	CmdJoin
)

type ResponseSet struct {
	Status Status
}

func (r *ResponseSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.Status)
	return buf.Bytes()
}

func ParseSetResponse(r io.Reader) (*ResponseSet, error) {
	resp := &ResponseSet{}
	err := binary.Read(r, binary.LittleEndian, &resp.Status)
	return resp, err
}

type ResponseGet struct {
	Status Status
	Value  []byte
}

func (r *ResponseGet) Bytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, r.Status)
	valueLen := int32(len(r.Value))
	binary.Write(buf, binary.LittleEndian, valueLen)
	binary.Write(buf, binary.LittleEndian, r.Value)
	return buf.Bytes()
}

func ParseGetResponse(r io.Reader) (*ResponseGet, error) {
	// fmt.Println("[PROTO] ENTERED")
	resp := &ResponseGet{}
	err := binary.Read(r, binary.LittleEndian, &resp.Status)
	// fmt.Printf("[PROTO] Status %v %v", resp.Status, err)
	var valueLen int32
	binary.Read(r, binary.LittleEndian, &valueLen)

	resp.Value = make([]byte, valueLen)
	err = binary.Read(r, binary.LittleEndian, &resp.Value)
	return resp, err
}

func ParseCommand(r io.Reader) (any, error) {
	var cmd CommandB
	if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
		return nil, err
	}
	switch cmd {
	case CmdSet:
		return parseSetCommnad(r), nil
	case CmdGet:
		return parseGetCommnad(r), nil
	case CmdJoin:
		return parseJoinCommnad(r), nil
	default:
		return nil, fmt.Errorf("invalid command")
	}
}

type CommandSet struct {
	Key   []byte
	Value []byte
	TTL   int32
}

func (c *CommandSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, CmdSet)
	k, v := int32(len(c.Key)), int32(len(c.Value))
	binary.Write(buf, binary.LittleEndian, k)
	binary.Write(buf, binary.LittleEndian, c.Key)
	binary.Write(buf, binary.LittleEndian, v)
	binary.Write(buf, binary.LittleEndian, c.Value)
	binary.Write(buf, binary.LittleEndian, c.TTL)

	return buf.Bytes()
}

func parseSetCommnad(r io.Reader) *CommandSet {
	cmd := &CommandSet{}

	var k, v int32

	binary.Read(r, binary.LittleEndian, &k)
	key := make([]byte, k)
	binary.Read(r, binary.LittleEndian, &key)

	binary.Read(r, binary.LittleEndian, &v)
	value := make([]byte, v)
	binary.Read(r, binary.LittleEndian, &value)

	var TTL int32
	binary.Read(r, binary.LittleEndian, &TTL)

	cmd.TTL = TTL
	cmd.Key = key
	cmd.Value = value
	return cmd
}

type CommandGet struct {
	Key []byte
}

func (c *CommandGet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, CmdGet)
	k := int32(len(c.Key))
	binary.Write(buf, binary.LittleEndian, k)
	binary.Write(buf, binary.LittleEndian, c.Key)
	return buf.Bytes()
}

func parseGetCommnad(r io.Reader) *CommandGet {
	cmd := &CommandGet{}

	var k int32

	binary.Read(r, binary.LittleEndian, &k)
	key := make([]byte, k)
	binary.Read(r, binary.LittleEndian, &key)
	cmd.Key = key
	return cmd
}

type CommandJoin struct {
	NodeId      []byte
	RaftAddress []byte
}

func (c *CommandJoin) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, CmdJoin)
	k, v := int32(len(c.NodeId)), int32(len(c.RaftAddress))
	binary.Write(buf, binary.LittleEndian, k)
	binary.Write(buf, binary.LittleEndian, c.NodeId)
	binary.Write(buf, binary.LittleEndian, v)
	binary.Write(buf, binary.LittleEndian, c.RaftAddress)
	return buf.Bytes()
}

func parseJoinCommnad(r io.Reader) *CommandJoin {
	cmd := &CommandJoin{}
	var k, v int32
	binary.Read(r, binary.LittleEndian, &k)
	nodeId := make([]byte, k)
	binary.Read(r, binary.LittleEndian, &nodeId)
	binary.Read(r, binary.LittleEndian, &v)
	raftAddr := make([]byte, v)
	binary.Read(r, binary.LittleEndian, &raftAddr)

	cmd.NodeId = nodeId
	cmd.RaftAddress = raftAddr
	return cmd
}
