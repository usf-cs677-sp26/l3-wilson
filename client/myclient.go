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
	"strings"
)

func put(msgHandler *messages.MessageHandler, fileName string) int {
	fmt.Println("PUT", fileName)

	info, err := os.Stat(fileName)
	if err != nil {
		log.Fatalln(err)
	}

	msgHandler.SendStorageRequest(fileName, uint64(info.Size()))
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}

	md5Hash := md5.New()
	w := io.MultiWriter(msgHandler, md5Hash)
	io.CopyN(w, file, info.Size())
	file.Close()

	checksum := md5Hash.Sum(nil)
	msgHandler.SendChecksumVerification(checksum)

	fmt.Println("Storage complete!")
	return 0
}

func get(msgHandler *messages.MessageHandler, fileName string) int {
	fmt.Println("GET", fileName)

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
		return 1
	}

	msgHandler.SendRetrievalRequest(fileName)
	ok, _, size := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		file.Close()
		os.Remove(fileName)
		return 1
	}

	md5Hash := md5.New()
	w := io.MultiWriter(file, md5Hash)
	io.CopyN(w, msgHandler, int64(size))
	file.Close()

	clientCheck := md5Hash.Sum(nil)
	checkMsg, err := msgHandler.Receive()
	if err != nil {
		log.Println("Error receiving checksum:", err)
		return 1
	}
	serverCheck := checkMsg.GetChecksum().Checksum

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully retrieved file.")
	} else {
		log.Println("FAILED to retrieve file. Invalid checksum.")
		return 1
	}

	return 0
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Not enough arguments. Usage: %s server:port put|get file-name [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	host := os.Args[1]
	action := strings.ToLower(os.Args[2])
	fileName := os.Args[3]

	if action != "put" && action != "get" {
		log.Fatalln("Invalid action", action)
	}

	dir := "."
	if len(os.Args) >= 5 {
		dir = os.Args[4]
	}
	if err := os.Chdir(dir); err != nil {
		log.Fatalln(err)
	}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln(err.Error())
	}
	msgHandler := messages.NewMessageHandler(conn)
	defer msgHandler.Close()

	if action == "put" {
		os.Exit(put(msgHandler, fileName))
	} else {
		os.Exit(get(msgHandler, fileName))
	}
}
