package main

import (
	"flag"
	"fmt"
	"os"
  "os/signal"
  "syscall"
	"net"
	"time"
	"strconv"
	"math"
)

// How to Use
// - sudo go run easy_ping.go [hostname or ip address]
//
// Easy_Ping is a simple, clean implementation of the unix Ping CLI written in Go.
// I have implemented the extra credit of enabling support for both IPv4 and IPv6,
// as well as extra statistics when you press control ^c to exit the Application

const (
	defaultTimeoutSeconds = 10
	defaultNetwork = ""
)
var successful = 0
var failed = 0
var minRTT = math.MaxFloat64
var maxRTT = float64(math.MinInt64)
var avgRTT = float64(0)

func exit() {
	flag.Usage()
	os.Exit(1)
}

type Params struct {
	host    string
	timeout time.Duration
	network string
}

func parseArgs() Params {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] host port\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	timeoutPtr := flag.Int("W", defaultTimeoutSeconds, "time in seconds to wait for connections")
	network := flag.String("net", defaultNetwork, "the network to use")
	flag.Parse()

	if len(flag.Args()) < 1 {
		exit()
	}

	host := flag.Args()[0]


	return Params{
		host: host,
		timeout: time.Duration(*timeoutPtr) * time.Second,
		network: *network,
	}
}


// FormatResult converts the result returned by Ping to string.
func formatResult(err error) string {
	if err == nil {
		return "success"
	}
	switch err := err.(type) {
	case *net.OpError:
		return err.Err.Error()
	default:
		return err.Error()
	}
}
// Ping connects to the address on the named network,
// using net.DialTimeout, and immediately closes it.
// It returns the connection error. A nil value means success.
// For examples of valid values of network and address,
// see the documentation of net.Dial
func Ping(network, address string, timeout time.Duration) error {
	conn, err := net.DialTimeout(network, address, timeout)
	if conn != nil {
		defer conn.Close()
	}
	return err
}

// PingN calls Ping infinite amount of times,
// and sends the results to the given channel.
func PingN(network, address string, timeout time.Duration, c chan <- string) {
	for {
		startTime := time.Now()
		err := Ping(network, address, timeout)
		if formatResult(err) == "success" {
			successful = successful + 1
		} else {
			failed = failed + 1
		}
		latency := time.Since(startTime).Seconds() * 1000
		if latency < minRTT {
			minRTT = latency
		} else if latency > maxRTT {
			maxRTT = latency
		}
		avgRTT = (float64(avgRTT) * float64(failed + successful - 1) + latency) / float64(failed + successful)
		output := "Packets: sent = " +  strconv.Itoa(successful + failed) + ", recieved = " +
		strconv.Itoa(successful) + ", Lost = " + strconv.Itoa(failed) + "(" +
		strconv.Itoa(failed/(successful + failed)) + "% loss), Latency: " + fmt.Sprintf("%f ms", latency)


		c <- output
	}
}

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

func isIPv6(ip net.IP) bool {
	return len(ip) == net.IPv6len
}


func statistics() {
    fmt.Println("Statistics:")
		fmt.Println("Packets: sent = " +  strconv.Itoa(successful + failed) + ", recieved = " +
		strconv.Itoa(successful) + ", Lost = " + strconv.Itoa(failed) + "(" +
		strconv.Itoa(failed/(successful + failed)) + "% loss),")
		fmt.Println("Approximate round trip times in milli-seconds:")
		fmt.Println("Minimum: " + fmt.Sprintf("%f ms Maximum: %f ms, Average: %f ms", minRTT, maxRTT, avgRTT))
}




func main() {
	params := parseArgs()
// Grabs IP Address
  ipaddr, err := net.ResolveIPAddr("ip", params.host)
	if err != nil {
		panic(err)
	}

//Checks if IP Address is IPV4 or IPv6
	if isIPv4(ipaddr.IP) {
    params.network = "ip4:icmp"
	} else if isIPv6(ipaddr.IP) {
    params.network = "ip6:ipv6-icmp"
	}

	fmt.Println("Starting to ping " + ipaddr.String())

//Enables Ability to control ^C to end program and calculate statistics
  c := make(chan os.Signal)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  go func() {
      <-c
      statistics()
      os.Exit(1)
  }()



	d := make(chan string)
	go PingN(params.network, ipaddr.String(), params.timeout, d)

// Main infinite loop to ping address
	for {
		output := <-d
		fmt.Println("Reply from " + ipaddr.String() + ":" + output)
	}

}
