package main

import (
	"io"
	"encoding/json"
	"fmt"
	"log"
	"io/ioutil"
	"flag"
	"os"
	"bufio"
	"strings"
	"path"
	"sort"
	"net/http"
	"strconv"
)

var file string
var res1_low uint32
var res2_low uint32
var res3_low uint32
var res1_high uint32
var res2_high uint32
var res3_high uint32

func init() {
	flag.StringVar(&file, "f", "./", "Log file")
	flag.Parse()
	res1_low = ip_to_dec("10.0.0.0")
	res1_high = ip_to_dec("10.255.255.255")
	res2_low = ip_to_dec("172.16.0.0")
	res2_high = ip_to_dec("172.31.255.255")
	res3_low = ip_to_dec("192.168.0.0")
	res3_high = ip_to_dec("192.168.255.255")
}

var target string
var client string
var platform string

type Location struct {
	latitude float64
	longitude float64
}

type Hop struct {
	delay int
	addr string
	loc IPgeo
}

type Route struct {
	hops []Hop
}

type IPgeo struct {
	Ip	string
	Country_node string
	Country_name string
	Region_code string
	Region_name string
	City string
	Zip_code string
	Time_zone string
	Latitude float64
	Longitude float64
	Metro_code int
}

func get_delay(delaystrs []string) (int, error) {
	var delays []int
	for i := 0; i < 3; i++ {
		d, err := strconv.Atoi(delaystrs[i])
		if err != nil {
			return 0, err
		}
		delays = append(delays, d)
	}
	sort.Ints(delays)
	if delays[0] == -1 {
		return delays[2], nil
	} else {
		return delays[1], nil
	}
}

func ip_to_dec(addr string) uint32 {
	digits := strings.Split(addr, ".")
	var dec uint32 = 0
	for i := 0; i < 4; i++ {
		d, err := strconv.Atoi(digits[i])
		if err != nil {
			log.Fatal(err)
		}
		dec += uint32(d)<<uint(8*(3-i))
	}
	return dec
}

func is_internal_ip(addr string) bool {
	dec := ip_to_dec(addr)
	if (dec >= res1_low && dec <= res1_high) || (dec >= res2_low && dec <= res2_high) || (dec >= res3_low && dec <= res3_high) {
		return true
	}
	return false
}

func get_location(addr string) IPgeo {
	loc := IPgeo{Ip: addr}
	if addr == "0.0.0.0" {
		return loc
	}
	if is_internal_ip(addr) {
		log.Printf("%s is an internal address", addr)
		return loc
	}
	resp, err := http.Get(fmt.Sprintf("http://freegeoip.net/json/%s", addr))
	if err != nil {
		return loc
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return loc
	}
	err = json.Unmarshal(body, &loc)
	if err != nil {
		return loc
	}
	return loc
}

type PingStat struct {
	pkt_tot int
	pkt_loss int
	rate_loss float64
	delay float64
}

func main() {
	var route Route
	ping := PingStat{}
	delay_tot := 0.0
	filename := path.Base(file)
	filetype := strings.Split(filename, "_")
	fi, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer fi.Close()
	buf := bufio.NewReader(fi)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		line = strings.TrimSpace(line)
		data := strings.Split(line, ":")
		if data[0] == "Target" {
			target = strings.TrimSpace(data[1])
		}
		if data[0] == "LocalAddress" {
			client = strings.TrimSpace(data[1])
		}
		if data[0] == "Platform" {
			platform = strings.TrimSpace(data[1])
		}
		if len(data) == 1 {
			if filetype[0] == "tracert" {
				info := strings.Split(line, " ")
				delay, err := get_delay(info[1:4])
				if err != nil {
					log.Fatal(err)
				}
				loc := get_location(info[4])
				route.hops = append(route.hops, Hop{delay, info[4], loc})
				log.Print(loc)
			}
			if filetype[0] == "ping" {
				info := strings.Split(line, " ")
				if len(info) == 4 {
					ping.pkt_tot += 1
					if info[2] == "-1" {
						ping.pkt_loss += 1
					} else {
						d, err := strconv.Atoi(info[3])
						if err != nil {
							log.Fatal(err)
						}
						delay_tot += float64(d)
					}
				}
			}
		}
	}
	if filetype[0] == "ping" {
		ping.delay = delay_tot / float64(ping.pkt_tot - ping.pkt_loss)
		ping.rate_loss = float64(ping.pkt_loss) / float64(ping.pkt_tot)
		log.Print(ping)
	}
}
