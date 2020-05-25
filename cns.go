// cns.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	listener                                          *net.TCPListener
	udpFlag                                           string
	proxyKey                                          []byte
	udp_timeout, tcp_keepAlive                        time.Duration
	enable_dns_tcpOverUdp, enable_httpDNS, enable_TFO bool
)

func isHttpHeader(header []byte) bool {
	return bytes.HasPrefix(header, []byte("CONNECT")) ||
		bytes.HasPrefix(header, []byte("GET")) ||
		bytes.HasPrefix(header, []byte("POST")) ||
		bytes.HasPrefix(header, []byte("HEAD")) ||
		bytes.HasPrefix(header, []byte("PUT")) ||
		bytes.HasPrefix(header, []byte("COPY")) ||
		bytes.HasPrefix(header, []byte("DELETE")) ||
		bytes.HasPrefix(header, []byte("MOVE")) ||
		bytes.HasPrefix(header, []byte("OPTIONS")) ||
		bytes.HasPrefix(header, []byte("LINK")) ||
		bytes.HasPrefix(header, []byte("UNLINK")) ||
		bytes.HasPrefix(header, []byte("TRACE")) ||
		bytes.HasPrefix(header, []byte("PATCH")) ||
		bytes.HasPrefix(header, []byte("WRAPPED"))
}

func rspHeader(header []byte) []byte {
	if bytes.Contains(header, []byte("WebSocket")) {
		return []byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: CuteBi Network Tunnel, (%>w<%)\r\n\r\n")
	} else if bytes.HasPrefix(header, []byte("CON")) {
		return []byte("HTTP/1.1 200 Connection established\r\nServer: CuteBi Network Tunnel, (%>w<%)\r\nConnection: keep-alive\r\n\r\n")
	} else {
		return []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nServer: CuteBi Network Tunnel, (%>w<%)\r\nConnection: keep-alive\r\n\r\n")
	}
}

func handleConn(cConn *net.TCPConn, payload []byte) {
	RLen, err := cConn.Read(payload)
	if err != nil || RLen <= 0 {
		cConn.Close()
		return
	}
	if !isHttpHeader(payload[:RLen]) {
		handleUdpSession(cConn, payload[:RLen])
	} else {
		if !enable_httpDNS || !RespondHttpdns(cConn, payload[:RLen]) { /*优先处理httpDNS请求*/
			if WLen, err := cConn.Write(rspHeader(payload[:RLen])); err != nil || WLen <= 0 {
				cConn.Close()
				return
			}
			if bytes.Contains(payload[:RLen], []byte(udpFlag)) {
				handleConn(cConn, payload) //httpUDP需要读取到二进制数据才进行处理
			} else {
				handleTcpSession(cConn, payload)
			}
		}
	}
}

func pidSaveToFile(pidPath string) {
	fp, err := os.Create(pidPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = fp.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		fmt.Println(err)
	}
	fp.Close()
}

func handleCmd() {
	var listenAddrString, proxyKeyString, CuteBi_XorCrypt_passwordStr, pidPath string
	var isHelp, enable_daemon bool

	flag.StringVar(&proxyKeyString, "proxy-key", "Meng", "tcp request proxy host key")
	flag.StringVar(&udpFlag, "udp-flag", "httpUDP", "udp request flag string")
	flag.StringVar(&listenAddrString, "listen-addr", ":80", "listen aaddress")
	flag.StringVar(&CuteBi_XorCrypt_passwordStr, "encrypt-password", "quanyec", "encrypt password")
	flag.Int64Var((*int64)(&udp_timeout), "udp-timeout", 30, "udp timeout second")
	flag.Int64Var((*int64)(&tcp_keepAlive), "tcp-keepalive", 60, "tcp keepalive second")
	flag.StringVar(&pidPath, "pid-path", "", "pid file path")
	flag.BoolVar(&enable_dns_tcpOverUdp, "dns-tcpOverUdp", true, "tcpDNS Over udpDNS switch")
	flag.BoolVar(&enable_httpDNS, "enable-httpDNS", true, "httpDNS server switch")
	flag.BoolVar(&enable_TFO, "enable-TFO", true, "listener tcpFastOpen switch")
	flag.BoolVar(&enable_daemon, "daemon", true, "daemon mode switch")
	flag.BoolVar(&isHelp, "help", false, "display this message")

	flag.Parse()
	if isHelp {
		fmt.Println("　/) /)\nฅ(՞•ﻌ•՞)ฅ\nCuteBi Network Server 0.2.1\n" +
			"Author: CuteBi(Mmmdbybyd)\nE-mail: 915445800@qq.com\n")
		flag.Usage()
		os.Exit(0)
	}
	if enable_daemon {
		exec.Command(os.Args[0], []string(append(os.Args[1:], "-daemon=false"))...).Start()
		os.Exit(0)
	}
	listenAddr, err := net.ResolveTCPAddr("tcp", listenAddrString)
	listener, err = net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if enable_TFO {
		enableTcpFastOpen(listener)
	}
	if pidPath != "" {
		pidSaveToFile(pidPath)
	}
	proxyKey = []byte("\n" + proxyKeyString + ": ")
	CuteBi_XorCrypt_password = []byte(CuteBi_XorCrypt_passwordStr)
	udp_timeout *= time.Second
	tcp_keepAlive *= time.Second
}

func settings() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func initProcess() {
	handleCmd()
	setsid()
	setMaxNofile()
	signal.Ignore(syscall.SIGPIPE)
}

func handling() {
	for {
		conn, err := listener.AcceptTCP()
		if err == nil {
			conn.SetKeepAlive(true)
			conn.SetKeepAlivePeriod(tcp_keepAlive)
			go handleConn(conn, make([]byte, 8192))
		} else {
			log.Println(err)
			time.Sleep(3 * time.Second)
		}
	}
	listener.Close()
}

func main() {
	settings()
	initProcess()
	handling()
}
