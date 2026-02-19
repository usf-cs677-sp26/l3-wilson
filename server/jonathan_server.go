package main

import (
	"crypto/md5"
	"file-transfer/messages"
	"file-transfer/util"
	"io"
	"log"
	"net"
	"os"
	"path"
)

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	fileName := path.Base(request.GetFileName())

	log.Println("Attempting to store", fileName)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	msgHandler.SendResponse(true, "Ready for data")

	md5 := md5.New()

	w := io.MultiWriter(file, md5)
	io.CopyN(w, msgHandler, int64(request.Size))
	file.Close()

	serverCheck := md5.Sum(nil)

	clientCheckMsg, _ := msgHandler.Receive()
	clientCheck := clientCheckMsg.GetChecksum().Checksum

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully stored file.")
		msgHandler.SendResponse(true, "Successfully stored file.")
	} else {
		log.Println("Failed to store file. Invalid checksum.")
		msgHandler.SendResponse(false, "Unable to store file.")
	}
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	log.Println("Attempting to retrieve", request.FileName)

	info, err := os.Stat(request.FileName)
	if err != nil {
		log.Fatalln(err)
	}

	msgHandler.SendRetrievalResponse(true, "Ready to send", uint64(info.Size()))

	file, _ := os.Open(request.FileName)

	md5 := md5.New()

	w := io.MultiWriter(msgHandler, md5)
	io.CopyN(w, file, info.Size())
	file.Close()

	checksum := md5.Sum(nil)
	msgHandler.SendChecksumVerification(checksum)
}

func handleClient(msgHandler *messages.MessageHandler) {
	defer msgHandler.Close()

	for {
		wrapperMsg, err := msgHandler.Receive()
		if err != nil {
			log.Println(err)
		}

		switch msg := wrapperMsg.Msg.(type) {
		case *messages.Wrapper_StorageReq:
			handleStorage(msgHandler, msg.StorageReq)
			continue
		case *messages.Wrapper_RetrievalReq:
			handleRetrieval(msgHandler, msg.RetrievalReq)
			continue
		case nil:
			log.Println("Received an empty message, terminating client")
			return
		default:
			log.Printf("Unexpected message type: %T", msg)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: Usage: %s port [download-dir]\n", os.Args[0])
	}

	port := os.Args[1]
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer listener.Close()

	dir := "."
	if len(os.Args) >= 3 {
		dir = os.Args[2]
	}
	if err := os.Chdir(dir); err != nil {
		log.Fatalln(err)
	}

	log.Println("Listening on port:", port)
	log.Println("Download directory:", dir)

	for {
		if conn, err := listener.Accept(); err == nil {
			log.Println("New connection", conn.RemoteAddr())
			handler := messages.NewMessageHandler(conn)
			go handleClient(handler)
		}
	}
}
