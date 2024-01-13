package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

var (
	working   = 0
	dead      = 0
	wg        sync.WaitGroup
	writeSync sync.Mutex
	proxies   = flag.String("proxies", "", "path 2 proxies file")
	output    = flag.String("output", "workingProxies.txt", "path 2 output file")
	timeout   = flag.Int("timeout", 500, "timeout in ms")
	threads   = flag.Int("threads", 500, "threads")
)

func main() {
	flag.Parse()
	start := time.Now()
	inputFile, err := os.Open(*proxies)
	if err != nil {
		fmt.Println("Failed to open input file:", err)
		os.Exit(1)
	}
	defer inputFile.Close()
	outputFile, err := os.Create(*output)
	if err != nil {
		fmt.Println("Failed to create output file:", err)
		os.Exit(1)
	}
	defer outputFile.Close()
	scanner := bufio.NewScanner(inputFile)
	writer := bufio.NewWriter(outputFile)

	ch := make(chan struct{}, *threads)

	for scanner.Scan() {
		proxy := scanner.Text()
		wg.Add(1)
		ch <- struct{}{}
		go CheckAndWrite(proxy, writer, &wg, ch)
	}
	wg.Wait()
	fmt.Println("Finished.")

	took := time.Since(start)
	fmt.Println("Took:", took, ". ", working, " working, ", dead, " dead.")
}

func CheckAndWrite(proxy string, writer *bufio.Writer, wg *sync.WaitGroup, ch chan struct{}) {
	defer wg.Done()

	if IsValidIPPort(proxy) && CheckProxy(proxy, *timeout) {

		writeSync.Lock()
		_, err := writer.WriteString(proxy + "\n")
		writer.Flush()
		writeSync.Unlock()
		working++
		fmt.Println(working, " working, ", dead, " dead.")
		if err != nil {
			fmt.Println("Failed to write to file:", err)
		}
	} else {
		dead++
	}
	<-ch
}

func CheckProxy(ipPort string, timeout int) bool {
	conn, err := net.DialTimeout("tcp", ipPort, time.Duration(timeout)*time.Millisecond)
	if err != nil {
		return false
	}
	conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
	conn.Write([]byte("CONNECT 104.16.132.229:80 HTTP/1.1\r\nHost: 104.16.132.229:80\r\nConnection: Keep-Alive\r\n\r\n"))
	reply := make([]byte, 15)
	n, err := conn.Read(reply)
	if err != nil {
		return false
	}
	if len(string(reply[:n])) == 15 && string(reply[:15]) == "HTTP/1.1 200 Co" {
		return true
	}
	return false
}
func IsValidIPPort(ipPort string) bool {
	host, port, err := net.SplitHostPort(ipPort)
	if err != nil {
		return false
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	if _, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(ip.String(), port)); err != nil {
		return false
	}

	return true
}
