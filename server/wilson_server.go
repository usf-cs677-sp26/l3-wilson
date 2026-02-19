package main

import (
	"crypto/md5"
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) error {
	log.Println("Attempting to store", request.FileName)

	file, err := os.OpenFile(request.FileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		return err
	}

	msgHandler.SendResponse(true, "Ready for data")

	md5Hash := md5.New()
	w := io.MultiWriter(file, md5Hash)
	io.CopyN(w, msgHandler, int64(request.Size))
	file.Close()

	serverCheck := md5Hash.Sum(nil)

	clientCheckMsg, err := msgHandler.Receive()
	if err != nil {
		os.Remove(request.FileName)
		return fmt.Errorf("error receiving checksum: %w", err)
	}
	clientCheck := clientCheckMsg.GetChecksum().Checksum

	if !util.VerifyChecksum(serverCheck, clientCheck) {
		os.Remove(request.FileName)
		msgHandler.SendResponse(false, "Checksum mismatch")
		return fmt.Errorf("checksum mismatch for %s", request.FileName)
	}

	log.Println("Successfully stored", request.FileName)
	msgHandler.SendResponse(true, "File stored successfully")
	return nil
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) error {
	log.Println("Attempting to retrieve", request.FileName)

	info, err := os.Stat(request.FileName)
	if err != nil {
		msgHandler.SendRetrievalResponse(false, err.Error(), 0)
		return err
	}

	msgHandler.SendRetrievalResponse(true, "Ready to send", uint64(info.Size()))

	file, err := os.Open(request.FileName)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	md5Hash := md5.New()
	w := io.MultiWriter(msgHandler, md5Hash)
	io.CopyN(w, file, info.Size())
	file.Close()

	checksum := md5Hash.Sum(nil)
	msgHandler.SendChecksumVerification(checksum)
	return nil
}

func handleClient(msgHandler *messages.MessageHandler) {
	defer msgHandler.Close()

	for {
		wrapper, err := msgHandler.Receive()
		if err != nil {
			log.Println(err)
			return
		}

		switch msg := wrapper.Msg.(type) {
		case *messages.Wrapper_StorageReq:
			if err := handleStorage(msgHandler, msg.StorageReq); err != nil {
				log.Println(err)
				return
			}
		case *messages.Wrapper_RetrievalReq:
			if err := handleRetrieval(msgHandler, msg.RetrievalReq); err != nil {
				log.Println(err)
				return
			}
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
		fmt.Printf("Not enough arguments. Usage: %s port [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	port := os.Args[1]
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()

	dir := "."
	if len(os.Args) >= 3 {
		dir = os.Args[2]
	}
	if err := os.Chdir(dir); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Listening on port:", port)
	fmt.Println("Download directory:", dir)
	for {
		if conn, err := listener.Accept(); err == nil {
			log.Println("Accepted connection", conn.RemoteAddr())
			handler := messages.NewMessageHandler(conn)
			go handleClient(handler)
		}
	}
}
