# About

a distributed KV-Store Cache. The cache utilizes a custom low-level protocol based on TCP to establish communication with other nodes in the distributed system. To ensure high availability and fault tolerance, the cache employs the RAFT consensus algorithm.

## Protocol Explained

1. Nodes communicates with each other via a simple protocol over TCP With three main commands: `SET` , `GET` and `JOIN`
2. Each Command has some numeric code (see `protocol.go`)
3. every message has `Cmd` which defines which Command, `key` , `value` and `TTL`
4. the message is encoded in a byte form (in LittleEndian) based on the type of command
   1. if `SET` (code 1) : it's `1[LENGTH_OF_KEY][KEY][LENGTH_OF_VALUE][VALUE][TTL]`
   2. if `GET` (code 2): it's `2[LENGTH_OF_KEY][KEY]`
   3. if `JOIN` (code 3): it's `3[LENGTH_OF_NODE_ID][NODE_ID][LENGTH_OF_RAFT_ADRR][RAFT_ADDR]`
5. the message is Decoded in the same way based on the type of command and then determining the format of decoding

## How does this work ?

#### Initialization

1. when node is started, it checks if it's the leader
2. if it's,it starts accepting connections.
3. if it's not the leader, it dials the leader sending the join command
4. when the join commnad is recieved by the leader node it uses raft lib to check the state of the current node  (reject if not the leader), and add the a voter with server id and raft addr = the sender node id and raft addr so the raft algorithm be able to execute the voting and election process

#### Mutating state (`SET` Commnad)

1. if `SET` commnad is send it checks if it's the leader or not, if it's the leader it calls the `apply` of the raft FSM (see `fsm/fsm.go`)
2. it parses the data and validate that it's a `SET` commnad to proceed.

#### Reading State (`GET` Command)

  any node can provide data to be read

# Want to Try ?

> NOTES:
> 
> - Configurations of ports , etc.. are hard-coded in the Makefile, feel free to tune it.
> - remember that node_* files created by the application saves the state, so to start from a clean state, remove them first.

## Building the project

```
make build
```

## Running the leader node

```
make run
```

## Running Followers

1st Follower:

```
make runfollower1
```

2nd Follower:

```
make runfollower2
```



## Testing

first you need to set some values, you can run `make runset` to seed some via the leader node 

output:

```
❯ make runset
conntected successfully..
setting...
setting...
received:
set Sucessfully
received:
set Sucessfully
setting...
//Up to 10 ...
```

then you can run `make runget` to get the values from some follower  (see `Makefile`)

```
❯ make runget
conntected successfully..
Get Sucessfully val: V_0
Get Sucessfully val: V_1
Get Sucessfully val: V_2
Get Sucessfully val: V_3
Get Sucessfully val: V_4
Get Sucessfully val: V_5
Get Sucessfully val: V_6
Get Sucessfully val: V_7
Get Sucessfully val: V_8
Get Sucessfully val: V_9
```

## Acknowledge

Anthony GG with the fantastic [series](https://www.youtube.com/watch?v=s2zAh9g_Y2c)

Yusuf Syaifudin and his awesmone [article](https://yusufs.medium.com/creating-distributed-kv-database-by-implementing-raft-consensus-using-golang-d0884eef2e28)
