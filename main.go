/* SPDX-License-Identifier: CC0-1.0 */

package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	// "log"
)

type Config struct {
	Sid                      string
	ServerName               string
	SendPassword             string
	Description              string
	MainNick                 string
	MainRealHost             string
	MainHost                 string
	MainIdent                string
	MainAddr                 string
	MainUserTs               string
	MainUmode                string
	MainGecos                string
	MainJoinChannels         []string
	MainChannelModeOnJoin    string
	MainChannelModeAfterJoin string
	MainUidWithoutSid        string
	MainNickTs               string
	Connect                  string
}

func loadConfig(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	config := Config{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		switch key {
		case "sid":
			config.Sid = value
		case "server_name":
			config.ServerName = value
		case "send_password":
			config.SendPassword = value
		case "description":
			config.Description = value
		case "main_nick":
			config.MainNick = value
		case "main_real_host":
			config.MainRealHost = value
		case "main_host":
			config.MainHost = value
		case "main_ident":
			config.MainIdent = value
		case "main_addr":
			config.MainAddr = value
		case "main_user_ts":
			config.MainUserTs = value
		case "main_umode":
			config.MainUmode = value
		case "main_gecos":
			config.MainGecos = value
		case "main_join_channels":
			config.MainJoinChannels = strings.Split(value, ",")
		case "main_channel_mode_on_join":
			config.MainChannelModeOnJoin = value
		case "main_channel_mode_after_join":
			config.MainChannelModeAfterJoin = value
		case "main_uid_without_sid":
			config.MainUidWithoutSid = value
		case "main_nick_ts":
			config.MainNickTs = value
		case "connect":
			config.Connect = value
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func handleMessage(conn net.Conn, msg string, config Config) {
	var prefix string

	if strings.HasPrefix(msg, "@") {
		firstSpaceIndex := strings.Index(msg, " ")
		if firstSpaceIndex == -1 {
			fmt.Fprintf(conn, ":%s ERROR :IRCv3 tag, but no space\n", config.Sid)
			return
		}
		msg = msg[firstSpaceIndex+1:]
	}
	if strings.HasPrefix(msg, ":") {
		firstSpaceIndex := strings.Index(msg, " ")
		if firstSpaceIndex == -1 {
			fmt.Fprintf(conn, ":%s ERROR :Prefix, but no space\n", config.Sid)
			return
		}
		prefix = msg[1:firstSpaceIndex]
		msg = msg[firstSpaceIndex+1:]
	}

	var msgArray []string
	for {
		spaceIndex := strings.Index(msg, " ")
		if msg[0] == ':' {
			msgArray = append(msgArray, msg[1:])
			break
		} else if spaceIndex == -1 {
			msgArray = append(msgArray, msg)
			break
		}
		msgArray = append(msgArray, msg[:spaceIndex])
		msg = msg[spaceIndex+1:]
	}

	switch msgArray[0] {
	case "PING":
		handlePing(conn, msgArray, prefix, config)
	case "PRIVMSG":
		handlePrivMsg(conn, msgArray, prefix, config)
	default:
		// fmt.Printf("Unknown command %s\n", msgArray[0])
	}
}

func handlePing(conn net.Conn, msgArray []string, prefix string, config Config) {
	if len(msgArray) == 2 {
		if msgArray[1] != config.Sid {
			fmt.Fprintf(conn, ":%s ERROR :Got PING intended for \"%s\"\n", config.Sid, msgArray[1])
			return
		}
		if prefix == "" {
			fmt.Fprintf(conn, ":%s ERROR :Got PING with one argument without prefix\n", config.Sid)
			return
		}
		fmt.Fprintf(conn, ":%s PONG :%s\n", config.Sid, prefix)
	} else if len(msgArray) == 3 {
		fmt.Fprintf(conn, ":%s PONG %s :%s\n", config.Sid, msgArray[2], msgArray[1])
	} else {
		fmt.Fprintf(conn, ":%s ERROR :msgArray.length\n", config.Sid)
	}
}

func handlePrivMsg(conn net.Conn, msgArray []string, prefix string, config Config) {
	if prefix == "" {
		fmt.Fprintf(conn, ":%s ERROR :Got PRIVMSG without prefix\n", config.Sid)
		return
	}

	var replyTo string
	if strings.HasPrefix(msgArray[1], "#") {
		if !contains(config.MainJoinChannels, msgArray[1]) {
			return
		}
		if !strings.HasPrefix(msgArray[2], fmt.Sprintf("%s: ", config.MainNick)) {
			return
		}
		msgArray[2] = msgArray[2][len(fmt.Sprintf("%s: ", config.MainNick)):]
		replyTo = msgArray[1]
	} else if msgArray[1] == fmt.Sprintf("%s%s", config.Sid, config.MainUidWithoutSid) {
		replyTo = prefix
	} else {
		return
	}

	cmdArray := strings.Split(msgArray[2], " ")

	cmdArray[0] = strings.ToUpper(cmdArray[0])

	switch cmdArray[0] {
	case "HELP":
		fmt.Fprintf(conn, ":%s%s NOTICE %s :Hi! I am an instance of https://git.sr.ht/~runxiyu/tedfu/, an InspIRCd pseudoserver written in Go.\n", config.Sid, config.MainUidWithoutSid, replyTo)
	case "`":
		if len(cmdArray) > 1 {
			fmt.Fprintf(conn, strings.Join(cmdArray[1:], " ")+"\n")
		}
	default:
		fmt.Fprintf(conn, ":%s%s NOTICE %s :Unknown command: %s\n", config.Sid, config.MainUidWithoutSid, replyTo, cmdArray[0])
	}
}

func connect(config Config) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", config.Connect, tlsConfig)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	writer.WriteString("CAPAB START 1205\n")
	writer.WriteString("CAPAB END\n")
	writer.WriteString(fmt.Sprintf("SERVER %s %s 0 %s :%s\n", config.ServerName, config.SendPassword, config.Sid, config.Description))
	writer.WriteString(fmt.Sprintf("BURST %d\n", time.Now().Unix()))
	writer.WriteString(fmt.Sprintf(
		":%s UID %s%s %s %s %s %s %s %s %s %s :%s\n",
		config.Sid,
		config.Sid,
		config.MainUidWithoutSid,
		config.MainNickTs,
		config.MainNick,
		config.MainRealHost,
		config.MainHost,
		config.MainIdent,
		config.MainAddr,
		config.MainUserTs,
		config.MainUmode,
		config.MainGecos,
	))
	for _, channelName := range config.MainJoinChannels {
		writer.WriteString(fmt.Sprintf(
			":%s FJOIN %s %s + :%s,%s%s\n",
			config.Sid,
			channelName,
			config.MainUserTs,
			config.MainChannelModeOnJoin,
			config.Sid,
			config.MainUidWithoutSid,
		))
		writer.WriteString(fmt.Sprintf(
			":%s MODE %s +%s :%s%s\n", // should probably use IMODE or some new 1205 thing
			config.Sid,
			channelName,
			config.MainChannelModeAfterJoin,
			config.Sid,
			config.MainUidWithoutSid,
		))
	}
	writer.WriteString("ENDBURST\n")
	writer.Flush()

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading:", err)
			break
		}
		if msg[len(msg)-1] == '\n' {
			msg = msg[:len(msg)-1]
		} else {
			panic("Message not newline terminated")
		}
		fmt.Println(msg)
		handleMessage(conn, msg, config)
	}
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func main() {
	config, err := loadConfig("config.txt")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	connect(config)
}
