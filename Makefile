build:
	@ go build -o bin/y3cache
run:build
	@ ./bin/y3cache
runfollower:build
	@ ./bin/y3cache --listenaddr :4000 --leaderaddr :3000

test:
	@go test -v ./...

runset:
	@go run ./client/runtest/ --set

runget:
	@go run ./client/runtest/ --get
