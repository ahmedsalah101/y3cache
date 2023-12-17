package fsm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/raft"

	"y3cache/cache"
	"y3cache/proto"
)

type CommnadPayload struct {
	Operation string
	Key       []byte
	Value     []byte
}

type ApplyResponse struct {
	Error error
	Data  any
}

type y3cacheFSM struct {
	c cache.Cacher
}

func (y y3cacheFSM) Apply(log *raft.Log) any {
	switch log.Type {
	case raft.LogCommand:
		rd := bytes.NewReader(log.Data)
		cmd, err := proto.ParseCommand(rd)
		if err != nil {
			fmt.Println("parse command error:", err)
			return nil
		}
		switch v := cmd.(type) {
		case *proto.CommandSet:
			err := y.c.Set(v.Key, v.Value, 0)
			if err != nil {
				return &proto.ResponseSet{
					Status: proto.StatusError,
				}
			}
			return &proto.ResponseSet{
				Status: proto.StatusOK,
			}
		}
	}
	_, _ = fmt.Fprintf(os.Stderr, "not raft command type\n")
	return nil
}

// you can use this method get all data from the cache
// and pass it to the FSMSnapshot so at the call of
// persist method of the FSMSnapshot it can save it
func (y y3cacheFSM) Snapshot() (raft.FSMSnapshot, error) {
	return NoOpSnp{}, nil
}

// snapshot: the snapshot written to the sink so you can
// read it and handle it to restore that snapshot
func (y y3cacheFSM) Restore(snapshot io.ReadCloser) error {
	defer func() {
		if err := snapshot.Close(); err != nil {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"[FINALLY RESTORE] close error %s",
				err.Error(),
			)
		}
	}()
	_, _ = fmt.Fprintf(
		os.Stdout,
		"[START RESTORE] read all message from snapshot\n",
	)
	var totalRestored int
	decoder := json.NewDecoder(snapshot)
	for decoder.More() {
		data := &CommnadPayload{}
		err := decoder.Decode(data)
		fmt.Printf("data: %v, total: %d", data, totalRestored)
		if err != nil {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"[END RESTORE] error decode data %s\n", err.Error())
			return err
		}
		if err := y.c.Set(data.Key, data.Value, 0); err != nil {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"[END RESTORE] error persist data %s\n", err.Error())
			return err
		}
		totalRestored++
	}
	_, err := decoder.Token()
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"[END RESTORE] error %s\n", err.Error())
		return err
	}

	_, _ = fmt.Fprintf(
		os.Stdout,
		"[END RESTORE] success restore %d message in snapshot\n", totalRestored)
	return nil
}

func NewY3CacheFSM(y cache.Cacher) raft.FSM {
	return &y3cacheFSM{
		c: y,
	}
}

type NoOpSnp struct{}

// use the NoOpSnp to get current snapshot made by Snapshot()
// and write it to g
func (s NoOpSnp) Persist(g raft.SnapshotSink) error {
	return nil
}
func (s NoOpSnp) Release() {}
