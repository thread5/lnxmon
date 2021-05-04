package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var api string = "http://127.0.0.1:1234/api"
var project string = "DEFAULT"

//
// const version string = "20190918"
// const version string = "20200806"
//
const version string = "20200921"

func get_id() string {
	var id string

	hostname, _ := os.Hostname()
	data := []byte(hostname)
	md5sum := md5.Sum(data)

	id = fmt.Sprintf("%x", md5sum)

	return id
}

func get_hostname() string {
	var hostname string

	_hostname, _ := os.Hostname()

	hostname = _hostname

	return hostname
}

func get_ip() string {
	var ip string

	var ips []string

	cmd := "ip -family inet address"
	cmd_result := exec_cmd_with_timeout(cmd)
	re := regexp.MustCompile(`inet\s*(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	re_result := re.FindAllStringSubmatch(cmd_result, -1)
	for _, v := range re_result {
		if v[1] != "127.0.0.1" {
			ips = append(ips, v[1])
		}
	}

	ip = strings.Join(ips, ",")

	return ip
}

func get_os_type() string {
	var os_type string

	var _os_type string

	var file string
	file = "/etc/centos-release"
	_, err := os.Stat(file)
	if err != nil {
		file = "/etc/redhat-release"
		_, err := os.Stat(file)
		if err != nil {
			file = "/etc/issue"
		}
	}

	content, _ := ioutil.ReadFile(file)
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 {
		_os_type = strings.TrimSpace(lines[0])
	}

	os_type = _os_type

	return os_type
}

func get_architecture() string {
	var architecture string

	var _architecture string

	cmd := "getconf LONG_BIT |head -c -1; echo -n \"-bit \"; uname -rm"
	cmd_result := exec_cmd_with_timeout(cmd)
	_architecture = strings.TrimSpace(cmd_result)

	architecture = _architecture

	return architecture
}

func get_cpu_processors() string {
	var cpu_processors string

	var _cpu_processors int64

	file, _ := os.Open("/proc/cpuinfo")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			_cpu_processors += 1
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	cpu_processors = fmt.Sprintf("%d", _cpu_processors)

	return cpu_processors
}

func get_mem_size() string {
	var mem_size string

	var _mem_size int64

	file, _ := os.Open("/proc/meminfo")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "MemTotal:") {
			x, _ := strconv.Atoi(strings.Fields(text)[1])
			_mem_size = int64(x)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	mem_size = fmt.Sprintf("%.0f", math.Round(float64(_mem_size)/(1024*1024)))

	return mem_size
}

func get_disk_size() string {
	var disk_size string

	var _disk_size uint64

	file, _ := os.Open("/proc/mounts")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "/dev") &&
			!strings.Contains(text, "chroot") &&
			!strings.Contains(text, "docker") {
			mount_point := strings.Fields(text)[1]
			var stat syscall.Statfs_t
			syscall.Statfs(mount_point, &stat)
			_disk_size += stat.Blocks * uint64(stat.Bsize)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	disk_size = fmt.Sprintf("%.0f", math.Round(float64(_disk_size)/(1024*1024*1024)))

	return disk_size
}

func get_uptime() string {
	var uptime string

	content, _ := ioutil.ReadFile("/proc/uptime")
	x, _ := strconv.ParseFloat(strings.Fields(string(content))[0], 64)

	uptime = fmt.Sprintf("%.2f", math.Round(x/(3600*24)))

	return uptime
}

func get_loadavg() string {
	var loadavg string

	content, _ := ioutil.ReadFile("/proc/loadavg")
	xs := strings.Fields(string(content))

	loadavg = fmt.Sprintf("%s,%s,%s", xs[0], xs[1], xs[2])

	return loadavg
}

func get_cpu_usage() string {
	var cpu_usage string

	file, _ := os.Open("/proc/stat")
	defer file.Close()
	var xs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		xs = strings.Fields(scanner.Text())
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	user, _ := strconv.Atoi(xs[1])
	nice, _ := strconv.Atoi(xs[2])
	system, _ := strconv.Atoi(xs[3])
	idle, _ := strconv.Atoi(xs[4])
	iowait, _ := strconv.Atoi(xs[5])
	irq, _ := strconv.Atoi(xs[6])
	softirq, _ := strconv.Atoi(xs[7])
	used := user + nice + system
	total := user + nice + system + idle + iowait + irq + softirq

	time.Sleep(1 * time.Second)

	file2, _ := os.Open("/proc/stat")
	defer file2.Close()
	var xs2 []string
	scanner2 := bufio.NewScanner(file2)
	for scanner2.Scan() {
		xs2 = strings.Fields(scanner2.Text())
		break
	}
	if err := scanner2.Err(); err != nil {
		panic(err)
	}
	user2, _ := strconv.Atoi(xs2[1])
	nice2, _ := strconv.Atoi(xs2[2])
	system2, _ := strconv.Atoi(xs2[3])
	idle2, _ := strconv.Atoi(xs2[4])
	iowait2, _ := strconv.Atoi(xs2[5])
	irq2, _ := strconv.Atoi(xs2[6])
	softirq2, _ := strconv.Atoi(xs2[7])
	used2 := user2 + nice2 + system2
	total2 := user2 + nice2 + system2 + idle2 + iowait2 + irq2 + softirq2

	_cpu_usage := float64(used2-used) / float64(total2-total)
	iowait_usage := float64(iowait2-iowait) / float64(total2-total)

	cpu_usage = fmt.Sprintf("%.2f,%.2f", _cpu_usage*100, iowait_usage*100)

	return cpu_usage
}

func get_mem_usage() string {
	var mem_usage string

	var memtotal int64
	var memfree int64
	var buffers int64
	var cached int64
	var swaptotal int64
	var swapfree int64

	file, _ := os.Open("/proc/meminfo")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "MemTotal:") {
			_memtotal, _ := strconv.Atoi(strings.Fields(text)[1])
			memtotal = int64(_memtotal)
		} else if strings.HasPrefix(text, "MemFree") {
			_memfree, _ := strconv.Atoi(strings.Fields(text)[1])
			memfree = int64(_memfree)
		} else if strings.HasPrefix(text, "Buffers") {
			_buffers, _ := strconv.Atoi(strings.Fields(text)[1])
			buffers = int64(_buffers)
		} else if strings.HasPrefix(text, "Cached") {
			_cached, _ := strconv.Atoi(strings.Fields(text)[1])
			cached = int64(_cached)
		} else if strings.HasPrefix(text, "SwapTotal") {
			_swaptotal, _ := strconv.Atoi(strings.Fields(text)[1])
			swaptotal = int64(_swaptotal)
		} else if strings.HasPrefix(text, "SwapFree") {
			_swapfree, _ := strconv.Atoi(strings.Fields(text)[1])
			swapfree = int64(_swapfree)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	mem_total := math.Round(float64(memtotal) / (1024 * 1024))
	swap_total := math.Round(float64(swaptotal) / (1024 * 1024))
	_mem_usage := float64(memtotal-memfree-buffers-cached) / float64(memtotal) * 100
	swap_usage := float64(swaptotal-swapfree) / (float64(swaptotal) + 0.1) * 100

	mem_usage = fmt.Sprintf("%.0f,%.2f,%.0f,%.2f", mem_total, _mem_usage, swap_total, swap_usage)

	return mem_usage
}

/*
type Statfs_t struct {
    Type    int64
    Bsize   int64
    Blocks  uint64
    Bfree   uint64
    Bavail  uint64
    Files   uint64
    Ffree   uint64
    Fsid    Fsid
    Namelen int64
    Frsize  int64
    Flags   int64
    Spare   [4]int64
}
*/
func get_disk_usage() string {
	var disk_usage string

	var disk_list []string

	file, _ := os.Open("/proc/mounts")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "/dev") &&
			!strings.Contains(text, "chroot") &&
			!strings.Contains(text, "docker") {
			mount_point := strings.Fields(text)[1]
			var stat syscall.Statfs_t
			syscall.Statfs(mount_point, &stat)
			total := uint64(stat.Frsize) * stat.Blocks
			used := uint64(stat.Frsize) * (stat.Blocks - stat.Bfree)
			var inode_usage float64
			if stat.Files != 0 {
				inode_usage = float64(stat.Files-stat.Ffree) / float64(stat.Files) * 100
			}

			disk_total := math.Round(float64(total) / (1024 * 1024 * 1024))
			_disk_usage := float64(used) / float64(total) * 100
			x := fmt.Sprintf("%s_%.0f_%.2f_%.2f", mount_point, disk_total, _disk_usage, inode_usage)
			disk_list = append(disk_list, x)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	disk_usage = strings.Join(disk_list, ",")

	return disk_usage
}

func get_disk_io_rate() string {
	var disk_io_rate string

	var disk_type int64

	re := regexp.MustCompile(`sd[a-z] `)
	re2 := regexp.MustCompile(`xvd[a-z] `)
	re3 := regexp.MustCompile(`xvd[a-z][0-9] `)
	re4 := regexp.MustCompile(`vd[a-z] `)
	re5 := regexp.MustCompile(`vd[a-z][0-9] `)

	file, _ := os.Open("/proc/diskstats")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if re.MatchString(text) {
			break
		} else if re2.MatchString(text) {
			disk_type = 2
			break
		} else if re3.MatchString(text) {
			disk_type = 3
			break
		} else if re4.MatchString(text) {
			disk_type = 4
			break
		} else if re5.MatchString(text) {
			disk_type = 5
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	var read_rate float64
	var write_rate float64
	var current_ios int64

	if disk_type == 0 {
		read_rate, write_rate, current_ios = parse_disk_io_rate(re)
	} else if disk_type == 2 {
		read_rate, write_rate, current_ios = parse_disk_io_rate(re2)
	} else if disk_type == 3 {
		read_rate, write_rate, current_ios = parse_disk_io_rate(re3)
	} else if disk_type == 4 {
		read_rate, write_rate, current_ios = parse_disk_io_rate(re4)
	} else if disk_type == 5 {
		read_rate, write_rate, current_ios = parse_disk_io_rate(re5)
	}

	disk_io_rate = fmt.Sprintf("%.2f,%.2f,%d", read_rate, write_rate, current_ios)

	return disk_io_rate
}

func parse_disk_io_rate(re *regexp.Regexp) (float64, float64, int64) {
	var read_rate float64
	var write_rate float64
	var current_ios int64

	var rsectors int64
	var wsectors int64
	var rsectors2 int64
	var wsectors2 int64

	file, _ := os.Open("/proc/diskstats")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if re.MatchString(text) {
			xs := strings.Fields(text)
			_rsectors, _ := strconv.Atoi(xs[5])
			_wsectors, _ := strconv.Atoi(xs[9])
			rsectors += int64(_rsectors)
			wsectors += int64(_wsectors)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	time.Sleep(1 * time.Second)

	file2, _ := os.Open("/proc/diskstats")
	defer file2.Close()
	scanner2 := bufio.NewScanner(file2)
	for scanner2.Scan() {
		text2 := scanner2.Text()
		if re.MatchString(text2) {
			xs2 := strings.Fields(text2)
			_rsectors2, _ := strconv.Atoi(xs2[5])
			_wsectors2, _ := strconv.Atoi(xs2[9])
			_current_ios, _ := strconv.Atoi(xs2[11])
			rsectors2 += int64(_rsectors2)
			wsectors2 += int64(_wsectors2)
			current_ios += int64(_current_ios)
		}
	}
	if err := scanner2.Err(); err != nil {
		panic(err)
	}

	read_rate = float64(rsectors2-rsectors) * 512 / 1024
	write_rate = float64(wsectors2-wsectors) * 512 / 1024

	return read_rate, write_rate, current_ios
}

func get_nic_io_rate() string {
	var nic_io_rate string

	var receive_bytes_rate float64
	var receive_packets_rate int64
	var transmit_bytes_rate float64
	var transmit_packets_rate int64

	var receive_bytes int64
	var transmit_bytes int64
	var receive_packets int64
	var transmit_packets int64

	file, _ := os.Open("/proc/net/dev")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if !strings.Contains(text, "Inter") &&
			!strings.Contains(text, "face") &&
			!strings.Contains(text, "lo:") {
			xs := strings.Fields(text)
			_receive_bytes, _ := strconv.Atoi(xs[1])
			_receive_packets, _ := strconv.Atoi(xs[2])
			_transmit_bytes, _ := strconv.Atoi(xs[9])
			_transmit_packets, _ := strconv.Atoi(xs[10])
			receive_bytes += int64(_receive_bytes)
			receive_packets += int64(_receive_packets)
			transmit_bytes += int64(_transmit_bytes)
			transmit_packets += int64(_transmit_packets)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	time.Sleep(1 * time.Second)

	var receive_bytes2 int64
	var transmit_bytes2 int64
	var receive_packets2 int64
	var transmit_packets2 int64

	file2, _ := os.Open("/proc/net/dev")
	defer file2.Close()
	scanner2 := bufio.NewScanner(file2)
	for scanner2.Scan() {
		text2 := scanner2.Text()
		if !strings.Contains(text2, "Inter") &&
			!strings.Contains(text2, "face") &&
			!strings.Contains(text2, "lo:") {
			xs2 := strings.Fields(text2)
			_receive_bytes2, _ := strconv.Atoi(xs2[1])
			_receive_packets2, _ := strconv.Atoi(xs2[2])
			_transmit_bytes2, _ := strconv.Atoi(xs2[9])
			_transmit_packets2, _ := strconv.Atoi(xs2[10])
			receive_bytes2 += int64(_receive_bytes2)
			receive_packets2 += int64(_receive_packets2)
			transmit_bytes2 += int64(_transmit_bytes2)
			transmit_packets2 += int64(_transmit_packets2)
		}
	}
	if err := scanner2.Err(); err != nil {
		panic(err)
	}

	receive_bytes_rate = float64(receive_bytes2-receive_bytes) / 1024
	receive_packets_rate = receive_packets2 - receive_packets
	transmit_bytes_rate = float64(transmit_bytes2-transmit_bytes) / 1024
	transmit_packets_rate = transmit_packets2 - transmit_packets

	nic_io_rate = fmt.Sprintf(
		"%.2f,%d,%.2f,%d",
		receive_bytes_rate,
		receive_packets_rate,
		transmit_bytes_rate,
		transmit_packets_rate,
	)

	return nic_io_rate
}

func get_tcp_sockets() string {
	var tcp_sockets string

	var inuse int64
	var tw int64

	file, _ := os.Open("/proc/net/sockstat")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "TCP:") {
			xs := strings.Fields(text)
			_inuse, _ := strconv.Atoi(xs[2])
			_tw, _ := strconv.Atoi(xs[6])
			inuse += int64(_inuse)
			tw += int64(_tw)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	_, err := os.Stat("/proc/net/sockstat6")
	if err == nil {
		file2, _ := os.Open("/proc/net/sockstat6")
		defer file2.Close()
		scanner2 := bufio.NewScanner(file2)
		for scanner2.Scan() {
			text2 := scanner2.Text()
			if strings.HasPrefix(text2, "TCP6:") {
				xs2 := strings.Fields(text2)
				_inuse2, _ := strconv.Atoi(xs2[2])
				inuse += int64(_inuse2)
				break
			}
		}
		if err := scanner2.Err(); err != nil {
			panic(err)
		}
	} else if os.IsNotExist(err) {
	} else {
	}

	tcp_sockets = fmt.Sprintf("%d,%d", inuse, tw)

	return tcp_sockets
}

/*
users, who, w
*/
func get_users() string {
	cmd := "users"
	cmd_result := exec_cmd_with_timeout(cmd)

	users := len(strings.Fields(cmd_result))
	users2 := fmt.Sprintf("%d", users)

	return users2
}

func get_current_local_time() string {
	var current_local_time string
	current_local_time = time.Now().Format("2006-01-02 15:04:05")
	return current_local_time
}

func get_current_utc_time() string {
	var current_utc_time string
	current_utc_time = time.Now().UTC().Format("2006-01-02 15:04:05")
	return current_utc_time
}

/*
[{"type": "sinfo", "data": {"hn": "localhost", ..}}]
*/
func encode_static_info() string {
	var static_info string

	x := []map[string]interface{}{{
		"type": "sinfo",
		"data": map[string]interface{}{
			"id":   get_id(),
			"hn":   get_hostname(),
			"ip":   get_ip(),
			"os":   get_os_type(),
			"arch": get_architecture(),
			"nps":  get_cpu_processors(),
			"ms":   get_mem_size(),
			"ds":   get_disk_size(),
			"upt":  get_uptime(),
			"ht":   get_current_local_time(),
			"pid":  project,
			"ver":  version,
		},
	}}
	y, _ := json.Marshal(x)

	static_info = string(y)

	return static_info
}

/*
return [{"type": "dinfo", "data": {"hn": "localhost", ..}}]
*/
func encode_dynamic_info() string {
	var dynamic_info string

	x := []map[string]interface{}{{
		"type": "dinfo",
		"data": map[string]interface{}{
			"id":    get_id(),
			"hn":    get_hostname(),
			"ip":    get_ip(),
			"ldg":   get_loadavg(),
			"cpu":   get_cpu_usage(),
			"mem":   get_mem_usage(),
			"disk":  get_disk_usage(),
			"dio":   get_disk_io_rate(),
			"nio":   get_nic_io_rate(),
			"skt":   get_tcp_sockets(),
			"users": get_users(),
			"ht":    get_current_local_time(),
			"pid":   project,
		},
	}}
	y, _ := json.Marshal(x)

	dynamic_info = string(y)

	return dynamic_info
}

func do_http_post(api string, action string, info string, token string) int64 {
	var http_code int64

	formData := url.Values{
		"action": {action},
		"info":   {info},
		"token":  {token},
	}
	log.Println(formData.Encode())

	req, err := http.NewRequest("POST", api, strings.NewReader(formData.Encode()))
	if err != nil {
		panic(err)
	}

	// req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		// panic(err)
		log.Println(err)
		log.Println("Keep trying...")
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	if resp != nil {
		log.Println("response Status:", resp.Status)
		log.Println("response Headers:", resp.Header)

		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("response Body:", string(body))

		x, _ := strconv.Atoi(resp.Status)
		http_code = int64(x)
	}

	return http_code
}

func sys_print_out() {
	fmt.Printf("HOST STATIC INFORMATION\n")
	fmt.Printf("ID\n")
	fmt.Printf("-- %s\n", get_id())
	fmt.Printf("Hostname\n")
	fmt.Printf("-- %s\n", get_hostname())
	fmt.Printf("IP\n")
	fmt.Printf("-- %s\n", get_ip())
	fmt.Printf("OS Type\n")
	fmt.Printf("-- %s\n", get_os_type())
	fmt.Printf("Architecture\n")
	fmt.Printf("-- %s\n", get_architecture())
	fmt.Printf("CPU Processors\n")
	fmt.Printf("-- %s\n", get_cpu_processors())
	fmt.Printf("Mem Size(G)\n")
	fmt.Printf("-- %s\n", get_mem_size())
	fmt.Printf("Disk Size(G)\n")
	fmt.Printf("-- %s\n", get_disk_size())
	fmt.Printf("Uptime(days)\n")
	fmt.Printf("-- %s\n", get_uptime())
	fmt.Printf("Current UTC Time\n")
	fmt.Printf("-- %s\n", get_current_local_time())
	fmt.Printf("\n")
	fmt.Printf("HOST DYNAMIC INFORMATION\n")
	fmt.Printf("Loadavg(1m,5m,15m)\n")
	fmt.Printf("-- %s\n", get_loadavg())
	fmt.Printf("CPU Usage(cpu_usage%%,iowait%%)\n")
	fmt.Printf("-- %s\n", get_cpu_usage())
	fmt.Printf("Mem Usage(mem_total(G),mem_usage%%,swap_total(G),swap_usage%%)\n")
	fmt.Printf("-- %s\n", get_mem_usage())
	fmt.Printf("Disk Usage(mountPoint_diskTotal(G)_diskUsage%%_inodeUsage%%,..)\n")
	fmt.Printf("-- %s\n", strings.ReplaceAll(get_disk_usage(), ",", "\n   "))
	fmt.Printf("Disk I/O Rate(read_rate(KB/s),write_rate(KB/s),current_requests)\n")
	fmt.Printf("-- %s\n", get_disk_io_rate())
	fmt.Printf("NIC I/O Rate(receive_rate(KB/s),receive_packets,transmit_rate(KB/s),transmit_packets)\n")
	fmt.Printf("-- %s\n", get_nic_io_rate())
	fmt.Printf("TCP Sockets(inuse,timewait)\n")
	fmt.Printf("-- %s\n", get_tcp_sockets())
	fmt.Printf("Users currently logged in\n")
	fmt.Printf("-- %s\n", get_users())
	fmt.Printf("\n")
	fmt.Printf("HOST STATIC INFORMATION\n")
	fmt.Printf("%s\n", encode_static_info())
	fmt.Printf("\n")
	fmt.Printf("HOST DYNAMIC INFORMATION\n")
	fmt.Printf("%s\n", encode_dynamic_info())
}

func routine_5(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		// log.Println(time.Now())
		// time.Sleep(1 * time.Second)

		encoded_static_info := encode_static_info()
		do_http_post(api, "report", encoded_static_info, "123456")

		time.Sleep(300 * time.Second)
		// time.Sleep(5 * time.Second)
	}
}

func routine_1(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		encode_dynamic_info := encode_dynamic_info()
		do_http_post(api, "report", encode_dynamic_info, "123456")

		time.Sleep(60 * time.Second)
		// time.Sleep(2 * time.Second)
	}
}

func go_routines() {}

func exec_cmd_with_timeout(cmd string) string {
	executor := exec.Command("sh", "-c", cmd)

	var buf bytes.Buffer
	executor.Stdout = &buf
	executor.Start()

	done := make(chan error)
	go func() { done <- executor.Wait() }()

	timeout := time.After(10 * time.Second)

	select {
	case <-timeout:
		executor.Process.Kill()
		return fmt.Sprintf("Command timed out after 10 secs")
	case err := <-done:
		var output string
		output = fmt.Sprintf("%s", buf.String())
		if err != nil {
			output = fmt.Sprintf("%s\n\nNon-zero exit code: %s", output, err)
		}
		return output
	}
}

/*
func main() {
	reflect.TypeOf(0)

	var args []string = os.Args

	var usage string = fmt.Sprintf(
		"Usage: %s {start|stop|restart|status|test|version}",
		args[0],
	)

	if len(args) != 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	switch args[1] {
	case "start":
		fmt.Println("start")
		var wg sync.WaitGroup
		wg.Add(1)
		go routine_5(&wg)
		wg.Add(1)
		go routine_1(&wg)
		wg.Wait()
	case "stop":
		fmt.Println("stop")
	case "restart":
		fmt.Println("restart")
	case "status":
		fmt.Println("status")
	case "test":
		sys_print_out()
	case "version":
		fmt.Println(version)
	default:
		fmt.Println(usage)
		os.Exit(2)
	}
}
*/

func main() {
	reflect.TypeOf(0)

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	_host := flag.String("host", "127.0.0.1", "Host")
	_port := flag.Int("port", 1234, "Port")
	_project := flag.String("project", "DEFAULT", "Project")
	flag.Parse()
	host := *_host
	port := strconv.Itoa(*_port)
	api = fmt.Sprintf("http://%s:%s/api", host, port)
	log.Println(fmt.Sprintf("API is %s", api))
	project = *_project
	log.Println(fmt.Sprintf("Project is %s", project))

	// sys_print_out()
	// os.Exit(0)

	log.Println("start")
	var wg sync.WaitGroup
	wg.Add(1)
	go routine_5(&wg)
	wg.Add(1)
	go routine_1(&wg)
	wg.Wait()

	/*
		var args []string = os.Args

		var usage string = fmt.Sprintf(
			"Usage: %s {start|test|version}",
			args[0],
		)

		if len(args) == 1 {
			log.Println("start")
			var wg sync.WaitGroup
			wg.Add(1)
			go routine_5(&wg)
			wg.Add(1)
			go routine_1(&wg)
			wg.Wait()
		} else if len(args) == 2 {
			switch args[1] {
			case "start":
				log.Println("start")
				var wg sync.WaitGroup
				wg.Add(1)
				go routine_5(&wg)
				wg.Add(1)
				go routine_1(&wg)
				wg.Wait()
			case "test":
				sys_print_out()
			case "version":
				log.Println(version)
			default:
				log.Println(usage)
				os.Exit(2)
			}
		} else {
			log.Println(usage)
			os.Exit(1)
		}
	*/
}
