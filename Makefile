
all: bin/client bin/server

bin/client: client/client.go messages/message_handler.go util/util.go
	go build -o bin/client client/client.go

bin/server: server/server.go messages/message_handler.go util/util.go
	go build -o bin/server server/server.go

clean:
	rm -rf bin/{client,server}