package main

import (
	"crypto/md5"
	"file-transfer/messages"
	"file-transfer/util"
	"io"
	"log"
	"net"
	"os"
)

func put(url, filePath string) (bool, string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err.Error()
	}

	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()

	conn, err := net.Dial("tcp", url)
	if err != nil {
		return false, err.Error()
	}

	msgHandler := messages.NewMessageHandler(conn)
	defer conn.Close()

	msgHandler.SendStorageRequest(fileName, uint64(fileSize))
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return false, "Error receiving response"
	}

	md5 := md5.New()

	file, _ := os.Open(filePath)
	w := io.MultiWriter(msgHandler, md5)
	io.CopyN(w, file, fileSize)
	file.Close()

	checksum := md5.Sum(nil)

	msgHandler.SendChecksumVerification(checksum)
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return false, "Error receiving checksum response"
	}

	return true, ""
}

func get(url, filePath, destinationDir string) (bool, string) {
	if err := os.Chdir(destinationDir); err != nil {
		return false, err.Error()
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		return false, err.Error()
	}

	conn, err := net.Dial("tcp", url)
	if err != nil {
		return false, err.Error()
	}

	msgHandler := messages.NewMessageHandler(conn)
	defer conn.Close()

	msgHandler.SendRetrievalRequest(filePath)
	ok, _, size := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		return false, "Error receiving retrieval response"
	}

	md5 := md5.New()

	w := io.MultiWriter(file, md5)
	io.CopyN(w, msgHandler, int64(size))
	file.Close()

	clientCheck := md5.Sum(nil)
	checkMsg, _ := msgHandler.Receive()
	serverCheck := checkMsg.GetChecksum().Checksum

	if util.VerifyChecksum(serverCheck, clientCheck) {
		return true, ""
	}

	return false, "Error verifying checksum"
}

func main() {
	args := os.Args[1:]

	if len(args) < 3 {
		log.Fatalln("Usage: ./client host:port action file-name [destination-dir]")
	}

	url := args[0]
	action := args[1]
	filePath := args[2]

	if action == "put" {
		if ok, err := put(url, filePath); !ok {
			log.Fatalln("Error to put file", err)
		}
	} else if action == "get" {
		destinationDir := "."
		if len(args) == 4 {
			destinationDir = args[3]
		}

		if ok, err := get(url, filePath, destinationDir); !ok {
			log.Fatalln("Error to get file", err)
		}
	}
}
