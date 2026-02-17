
all: bin/client bin/server bin/jonathan/client bin/wilson/client bin/jonathan/server bin/wilson/server

bin/client: client/client.go messages/message_handler.go util/util.go
	go build -o bin/client client/client.go

bin/server: server/server.go messages/message_handler.go util/util.go
	go build -o bin/server server/server.go

bin/jonathan/client: client/jonathan_client.go messages/message_handler.go util/util.go
	go build -o bin/jonathan/client client/jonathan_client.go

bin/jonathan/server: server/jonathan_server.go messages/message_handler.go util/util.go
	go build -o bin/jonathan/server server/jonathan_server.go

bin/wilson/client: client/wilson_client.go messages/message_handler.go util/util.go
	go build -o bin/wilson/client client/wilson_client.go

bin/wilson/server: server/wilson_server.go messages/message_handler.go util/util.go
	go build -o bin/wilson/server server/wilson_server.go

clean:
	rm -rf bin/{client,server}