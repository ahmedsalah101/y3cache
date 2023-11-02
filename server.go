package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
	"y3cache/cache"
	"y3cache/client"
	"y3cache/proto"

	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type ServerOpts struct {
	ListenAddr string
	IsLeader   bool
	LeaderAddr string
}

type Server struct {
	ServerOpts
	members map[*client.Client]struct{}
	cache   cache.Cacher
	raft    *raft.Raft
	// logger  *zap.Logger
	logger *zap.SugaredLogger
}

func NewServer(opts ServerOpts, c cache.Cacher, r *raft.Raft) *Server {
	ll, _ := zap.NewProduction()
	l := ll.Sugar()
	return &Server{
		ServerOpts: opts,
		// TODO: only allocate when server is the leader
		members: make(map[*client.Client]struct{}),
		raft:    r,
		cache:   c,
		logger:  l,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen error: %s", err)
	}

	if !s.IsLeader && len(s.LeaderAddr) != 0 {
		if err := s.dialLeader(); err != nil {
			log.Println(err)
		}
	}
	// if !s.IsLeader && len(s.LeaderAddr) != 0 {
	// 	go func() {
	// 		if err := s.dialLeader(); err != nil {
	// 			log.Println(err)
	// 		}
	// 	}()
	// }
	s.logger.Infow(
		"server starting",
		"addr",
		s.ListenAddr,
		"leader",
		s.IsLeader,
	)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %s\n", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) dialLeader() error {
	conn, err := net.Dial("tcp", s.LeaderAddr)
	if err != nil {
		return fmt.Errorf("failed to dial leader [%s]", s.LeaderAddr)
	}
	defer conn.Close()
	log.Println("connected to leader:", s.LeaderAddr)
	j := &proto.CommandJoin{
		NodeId:      []byte("node2"),
		RaftAddress: []byte("127.0.0.1:1112"),
	}
	_, err = conn.Write(j.Bytes())
	if err != nil {
		return err
	}
	// binary.Write(conn, binary.LittleEndian, proto.CmdJoin)

	// s.handleConn(conn)
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	// buf := make([]byte, 2048)
	// fmt.Println("[SERV] connection made:", conn.RemoteAddr())
	for {
		cmd, err := proto.ParseCommand(conn)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("parse command error:", err)
			break
		}
		go s.handleCommand(conn, cmd)
	}
	// fmt.Println("[SERV] connection closed:", conn.RemoteAddr())
}

func (s *Server) handleCommand(conn net.Conn, cmd any) {
	fmt.Println("[SERV LEADER] Recieved Command, Handling...")
	switch v := cmd.(type) {
	case *proto.CommandSet:
		if s.raft.State() != raft.Leader {
			log.Println("[SERV FOLLOWER] recieving SET command, ERR!")
			rs := &proto.ResponseSet{
				Status: proto.StatusError,
			}
			_, err := conn.Write(rs.Bytes())
			if err != nil {
				log.Println("[SERV FOLLOWER] error while responding to client")
			}
			return
			// return fmt.Errorf("not the leader")
		}
		s.handleSetCommand(conn, v)
	case *proto.CommandGet:
		s.handleGetCommand(conn, v)
	case *proto.CommandJoin:
		if s.raft.State() != raft.Leader {
			log.Println("[SERV FOLLOWER] recieving SET command, ERR!")
			rs := &proto.ResponseSet{
				Status: proto.StatusError,
			}
			_, err := conn.Write(rs.Bytes())
			if err != nil {
				log.Println("[SERV FOLLOWER] error while responding to client")
			}
			return
			// return fmt.Errorf("not the leader")
		}
		s.handleJoinCommnad(conn, v)
	}
	// fmt.Println(cmd)
}

func (s *Server) handleJoinCommnad(
	conn net.Conn,
	cmd *proto.CommandJoin,
) error {
	fmt.Printf(
		"[SERV J] %s,addr: %s\n",
		string(cmd.NodeId),
		string(cmd.RaftAddress),
	)
	// fmt.Println("member just joined the cluster:", conn.RemoteAddr())
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader")
	}
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return fmt.Errorf("failed to get raft conf %s", err.Error())
	}
	f := s.raft.AddVoter(
		raft.ServerID(cmd.NodeId),
		raft.ServerAddress(cmd.RaftAddress),
		0,
		0,
	)
	if f.Error() != nil {
		return fmt.Errorf("error add voter: %s", f.Error().Error())
	}
	fmt.Printf(
		"node %s at %s joined successfully\n",
		cmd.NodeId,
		cmd.RaftAddress,
	)
	pp(s.raft.Stats())
	// s.members[client.NewFromConn(conn)] = struct{}{}
	return nil
}

func pp(d map[string]string) {
	fmt.Printf("{\n")
	for k, v := range d {
		fmt.Printf("%s: %s.\n", k, v)
	}
	fmt.Printf("}\n")
}

func (s *Server) handleSetCommand(conn net.Conn, cmd *proto.CommandSet) error {
	fmt.Printf("applying.. %s\n", string(cmd.Key))
	applyFuture := s.raft.Apply(cmd.Bytes(), 500*time.Millisecond)
	fmt.Printf("applied %s\n", string(cmd.Key))
	if err := applyFuture.Error(); err != nil {
		fmt.Println("error while applying: ", err)
		return fmt.Errorf(
			"error persisting data in raft cluster: %s",
			err.Error(),
		)
	}

	fmt.Printf("applied with no errors %s\n", string(cmd.Key))
	// go func() {
	// 	// WARN take care of concurrency
	// 	for member := range s.members {
	// 		err := member.Set(context.TODO(), cmd.Key, cmd.Value)
	// 		if err != nil {
	// 			log.Println("forward to member error:", err)
	// 		}
	// 	}
	// }()

	// pRs := &proto.ResponseSet{}
	r, ok := applyFuture.Response().(*proto.ResponseSet)

	// fmt.Printf("applied with no resp %v, and isApplyResp: %v\n", r, ok)
	if !ok {
		// if r.Error != nil {
		// 	pRs.Status = proto.StatusError
		// 	conn.Write(pRs.Bytes())
		// }
		return fmt.Errorf("error response is not match apply response\n")
	}

	log.Printf("[SERV] SET %s to %s\n", cmd.Key, cmd.Value)
	log.Printf("[SERV] sucess persisting data\n")
	// pRs.Status = proto.StatusOK
	// conn.Write(pRs.Bytes())
	conn.Write(r.Bytes())
	return nil
	// resp := proto.ResponseSet{}
	// err := s.cache.Set(cmd.Key, cmd.Value, time.Duration(cmd.TTL))
	// if err != nil {
	// 	resp.Status = proto.StatusError
	// 	_, err := conn.Write(resp.Bytes())
	// 	return err
	// }
	//
	// resp.Status = proto.StatusOK
	// _, err = conn.Write(resp.Bytes())

	// return err
}

func (s *Server) handleGetCommand(conn net.Conn, cmd *proto.CommandGet) error {
	// log.Printf("[SERV] GET %s\n", cmd.Key)
	resp := proto.ResponseGet{}
	value, err := s.cache.Get(cmd.Key)
	if err != nil {
		resp.Status = proto.StatusKeyNotFound
		_, err := conn.Write(resp.Bytes())
		return err
	}
	resp.Status = proto.StatusOK
	resp.Value = value
	_, err = conn.Write(resp.Bytes())
	// fmt.Printf(
	// 	"[SERV] %v written to addr: %v\n",
	// 	resp.Bytes(),
	// 	conn.RemoteAddr(),
	// )
	return err
}
