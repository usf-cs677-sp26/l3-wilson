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

func put(msgHandler *messages.MessageHandler, fileName string) error {
	fmt.Println("PUT", fileName)

	info, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	msgHandler.SendStorageRequest(fileName, uint64(info.Size()))
	if ok, msg := msgHandler.ReceiveResponse(); !ok {
		return fmt.Errorf("server rejected storage request: %s", msg)
	}

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	md5Hash := md5.New()
	w := io.MultiWriter(msgHandler, md5Hash)
	io.CopyN(w, file, info.Size())
	file.Close()

	checksum := md5Hash.Sum(nil)
	msgHandler.SendChecksumVerification(checksum)

	if ok, msg := msgHandler.ReceiveResponse(); !ok {
		return fmt.Errorf("checksum mismatch: %s", msg)
	}

	fmt.Println("Storage complete!")
	return nil
}

func get(msgHandler *messages.MessageHandler, fileName string) error {
	fmt.Println("GET", fileName)

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	msgHandler.SendRetrievalRequest(fileName)
	ok, msg, size := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		file.Close()
		os.Remove(fileName)
		return fmt.Errorf("server rejected retrieval request: %s", msg)
	}

	md5Hash := md5.New()
	w := io.MultiWriter(file, md5Hash)
	io.CopyN(w, msgHandler, int64(size))
	file.Close()

	clientCheck := md5Hash.Sum(nil)
	checkMsg, err := msgHandler.Receive()
	if err != nil {
		os.Remove(fileName)
		return fmt.Errorf("error receiving checksum: %w", err)
	}
	serverCheck := checkMsg.GetChecksum().Checksum

	if !util.VerifyChecksum(serverCheck, clientCheck) {
		os.Remove(fileName)
		return fmt.Errorf("checksum mismatch — file corrupted")
	}

	log.Println("Successfully retrieved file.")
	return nil
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
		log.Fatalln(err)
	}
	msgHandler := messages.NewMessageHandler(conn)
	defer msgHandler.Close()

	if action == "put" {
		err = put(msgHandler, fileName)
	} else {
		err = get(msgHandler, fileName)
	}
	if err != nil {
		log.Fatalln(err)
	}
}
