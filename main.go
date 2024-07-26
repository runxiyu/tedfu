/* SPDX-License-Identifier: CC0-1.0 */

package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
)

func connect() error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", "127.0.0.1:12345", tlsConfig)
	if err != nil {
		log.Fatal("Error connecting: ", err)
		return err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	writer.WriteString("e\n")
	writer.Flush()

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Print("Error reading: ", err)
			return err
		}
		msg = msg[:len(msg)-1]
		fmt.Println(msg)

		// TODO: Handle the mssage
	}
}

func main() {
	if err := connect(); err != nil {
		panic(err)
	}
}
