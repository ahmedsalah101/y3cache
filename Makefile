build:
	@ go build -o bin/y3cache

run:build
	@ SERVER_PORT=2221 RAFT_NODE_ID=node1 RAFT_PORT=1111 RAFT_VOL_DIR=node_1_data ./bin/y3cache

runfollower1:build
	@ SERVER_PORT=2222 LEADER_PORT=2221 RAFT_NODE_ID=node2 RAFT_PORT=1112 RAFT_VOL_DIR=node_2_data ./bin/y3cache

runfollower2:build
	@ SERVER_PORT=2223 LEADER_PORT=2221 RAFT_NODE_ID=node3 RAFT_PORT=1113 RAFT_VOL_DIR=node_3_data ./bin/y3cache
test:
	@go test -v ./...

runset:
	@go run ./client/runtest/ --set --node-port=2221

runget:
	@go run ./client/runtest/ --get --node-port=2222
