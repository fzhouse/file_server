package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tealeg/xlsx"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var id string
var res1_low uint32
var res2_low uint32
var res3_low uint32
var res1_high uint32
var res2_high uint32
var res3_high uint32

type Base struct {
	Target   string
	Ip       string
	Platform string
	Location string
	Time     string
	Org      string
}

type PingStat struct {
	Pkt_tot   int
	Pkt_loss  int
	Rate_loss float64
	Delay     float64
}

type Hop struct {
	Delay int
	addr  string
	loc   IPgeo
}

type Route struct {
	hops []Hop
}

type IPgeo struct {
	Ip       string
	Hostname string
	City     string
	Region   string
	Country  string
	Loc      string
	Org      string
}

func init() {
	flag.StringVar(&id, "i", "0", "Log id")
	flag.Parse()
	res1_low = ip_to_dec("10.0.0.0")
	res1_high = ip_to_dec("10.255.255.255")
	res2_low = ip_to_dec("172.16.0.0")
	res2_high = ip_to_dec("172.31.255.255")
	res3_low = ip_to_dec("192.168.0.0")
	res3_high = ip_to_dec("192.168.255.255")
}

func get_Delay(Delaystrs []string) (int, error) {
	var Delays []int
	for i := 0; i < 3; i++ {
		d, err := strconv.Atoi(Delaystrs[i])
		if err != nil {
			return 0, err
		}
		Delays = append(Delays, d)
	}
	sort.Ints(Delays)
	if Delays[0] == -1 {
		return Delays[2], nil
	} else {
		return Delays[1], nil
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
		dec += uint32(d) << uint(8*(3-i))
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
	cli := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://ipinfo.io/%s", addr), nil)
	if err != nil {
		log.Fatal(fmt.Sprintf("request error: %v", err))
	}
	req.Header.Add("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		log.Fatal(fmt.Sprintf("HTTP get error: %v", err))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(fmt.Sprintf("read response body error: %v", err))
	}
	err = json.Unmarshal(body, &loc)
	if err != nil {
		log.Fatal(fmt.Sprintf("parsing json error: %v", err))
	}
	return loc
}

func ToSlice(arr interface{}) []interface{} {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice {
		panic("toslice arr not slice")
	}
	l := v.Len()
	ret := make([]interface{}, l)
	for i := 0; i < l; i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

func writeMulti(sheet *xlsx.Sheet, name string, val []interface{}) {
	row := sheet.AddRow()
	cell := row.AddCell()
	cell.Value = name
	for _, v := range val {
		cell = row.AddCell()
		cell.SetValue(v)
	}
}

func writeSingle(sheet *xlsx.Sheet, name string, val interface{}) {
	obj := reflect.ValueOf(val)
	ref := obj.Elem()
	row := sheet.AddRow()
	cell := row.AddCell()
	cell.Value = name
	for i := 0; i < ref.NumField(); i++ {
		field := ref.Field(i)
		cell = row.AddCell()
		cell.SetValue(field.Interface())
	}
}

func writeSingleRow(sheet *xlsx.Sheet, val interface{}) {
	obj := reflect.ValueOf(val)
	ref := obj.Elem()
	typ := ref.Type()
	for i := 0; i < ref.NumField(); i++ {
		field := ref.Field(i)
		row := sheet.AddRow()
		cell := row.AddCell()
		cell.Value = typ.Field(i).Name
		cell = row.AddCell()
		cell.SetValue(field.Interface())
	}
}

func baseLogStat(fi *os.File, sheet *xlsx.Sheet) {
	var base Base
	fd, err := ioutil.ReadAll(fi)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(fd, &base)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(base)
	writeSingleRow(sheet, &base)
}

func pingLogStat(fi *os.File, sheet *xlsx.Sheet) {
	ping := PingStat{}
	delay_tot := 0.0
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
		info := strings.Split(line, ",")
		if len(info) == 4 {
			ping.Pkt_tot += 1
			if info[2] == "-1" {
				ping.Pkt_loss += 1
			} else {
				d, err := strconv.Atoi(info[3])
				if err != nil {
					log.Fatal(err)
				}
				delay_tot += float64(d)
			}
		}
	}
	ping.Delay = delay_tot / float64(ping.Pkt_tot-ping.Pkt_loss)
	ping.Rate_loss = float64(ping.Pkt_loss) / float64(ping.Pkt_tot)
	log.Println(ping)
	writeSingleRow(sheet, &ping)
}

func tracertLogStat(fi *os.File, sheet *xlsx.Sheet) {
	var route Route
	count := 0
	buf := bufio.NewReader(fi)
	writeMulti(sheet, "Hop", ToSlice([]string{"IP", "Hostname", "City", "Region", "Country", "Loc", "Org"}))
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		line = strings.TrimSpace(line)
		info := strings.Split(line, ",")
		Delay, err := get_Delay(info[1:4])
		if err != nil {
			log.Fatal(err)
		}
		log.Printf(info[4])
		loc := get_location(info[4])
		count += 1
		route.hops = append(route.hops, Hop{Delay, info[4], loc})
		writeSingle(sheet, fmt.Sprintf("%d", count), &loc)
		log.Print(loc)
	}
}

func main() {
	fo := xlsx.NewFile()

	sheet, err := fo.AddSheet("base")
	if err != nil {
		log.Fatal(err)
	}
	fi, err := os.Open(fmt.Sprintf("base_%s.log", id))
	if err != nil {
		log.Fatal(err)
	}
	baseLogStat(fi, sheet)
	fi.Close()

	sheet, err = fo.AddSheet("ping")
	if err != nil {
		log.Fatal(err)
	}
	fi, err = os.Open(fmt.Sprintf("ping_%s.log", id))
	if err != nil {
		log.Fatal(err)
	}
	pingLogStat(fi, sheet)
	fi.Close()

	sheet, err = fo.AddSheet("tracert")
	if err != nil {
		log.Fatal(err)
	}
	fi, err = os.Open(fmt.Sprintf("tracert_%s.log", id))
	if err != nil {
		log.Fatal(err)
	}
	tracertLogStat(fi, sheet)
	fi.Close()
	err = fo.Save(fmt.Sprintf("%s.xlsx", id))
	if err != nil {
		log.Fatal(err)
	}
}
