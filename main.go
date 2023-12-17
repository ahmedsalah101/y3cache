package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/spf13/viper"

	"y3cache/cache"
	"y3cache/fsm"
)

type config struct {
	Server configServer `mapstructure:"server"`
	Raft   configRaft   `mapstructure:"raft"`
}
type configServer struct {
	Port       int `mapstructure:"port"`
	LeaderPort int `mapstructure:"leader_port"`
}
type configRaft struct {
	NodeId    string `mapstructure:"node_id"`
	Port      int    `mapstructure:"port"`
	VolumeDir string `mapstructure:"volume_dir"`
}

const (
	serverPort = "SERVER_PORT"
	leaderPort = "LEADER_PORT"
	raftNodeId = "RAFT_NODE_ID"
	raftPort   = "RAFT_PORT"
	raftVolDir = "RAFT_VOL_DIR"
)

var confKeys = []string{
	serverPort,
	raftNodeId,
	raftPort,
	raftVolDir,
}

const (
	maxPool            = 3
	tcpTimeout         = 10 * time.Second
	raftSnapShotRetain = 2
	raftLogCacheSize   = 512
)

func main() {
	v := viper.New()
	v.AutomaticEnv()
	conf := config{
		Server: configServer{
			Port:       v.GetInt(serverPort),
			LeaderPort: v.GetInt(leaderPort),
		},
		Raft: configRaft{
			NodeId:    v.GetString(raftNodeId),
			Port:      v.GetInt(raftPort),
			VolumeDir: v.GetString(raftVolDir),
		},
	}
	log.Printf("%+v\n", conf)

	raftBindAddr := fmt.Sprintf("127.0.0.1:%d", conf.Raft.Port)
	raftConf := raft.DefaultConfig()
	raftConf.LocalID = raft.ServerID(conf.Raft.NodeId)
	raftConf.SnapshotThreshold = 1024
	y3Cache := cache.New()

	y3FSM := fsm.NewY3CacheFSM(y3Cache)
	if conf.Raft.VolumeDir == "" {
		log.Fatal("please enter a valid dir")
		return
	}
	if err := os.MkdirAll(conf.Raft.VolumeDir, os.FileMode(0744)); err != nil {
		log.Fatal("couldn't create dir: ", err)
		return
	}
	store, err := raftboltdb.NewBoltStore(
		filepath.Join(conf.Raft.VolumeDir, "raft.dataRepo"),
	)
	if err != nil {
		log.Fatal(err)
		return
	}
	cacheStore, err := raft.NewLogCache(raftLogCacheSize, store)
	if err != nil {
		log.Fatal(err)
		return
	}
	snpStore, err := raft.NewFileSnapshotStore(
		conf.Raft.VolumeDir,
		raftSnapShotRetain,
		os.Stdout,
	)
	if err != nil {
		log.Fatal(err)
		return
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", raftBindAddr)
	if err != nil {
		log.Fatal(err)
		return
	}
	transport, err := raft.NewTCPTransport(
		raftBindAddr,
		tcpAddr,
		maxPool,
		tcpTimeout,
		os.Stdout,
	)
	fmt.Println(transport.LocalAddr())
	raftServer, err := raft.NewRaft(
		raftConf,
		y3FSM,
		cacheStore,
		store,
		snpStore,
		transport,
	)
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(conf.Raft.NodeId),
				Address: transport.LocalAddr(),
			},
		},
	}
	raftServer.BootstrapCluster(configuration)

	flag.Parse()
	opts := ServerOpts{
		NodeID:      conf.Raft.NodeId,
		RaftAddress: raftBindAddr,
		ListenAddr:  fmt.Sprintf(":%d", conf.Server.Port),
		LeaderAddr:  fmt.Sprintf(":%d", conf.Server.LeaderPort),
		IsLeader:    conf.Server.LeaderPort == 0,
	}
	server := NewServer(opts, y3Cache, raftServer)
	server.Start()
}
