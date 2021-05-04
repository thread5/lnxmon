package main

import (
	_ "./libs/go-sqlite3"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func whatif(err error) {
	if err != nil {
		panic(err)
	}
}

func dontcare(err error) {
	if err != nil {
		log.Println(err)
		log.Println("Don't care")
	}
}

func notfound(w http.ResponseWriter, r *http.Request) {
}

func api(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	info := r.FormValue("info")
	token := r.FormValue("token")
	log.Println(action)
	log.Println(info)
	log.Println(token)

	if strings.Contains(info, `"type":"sinfo"`) {
		sinfo := []struct {
			Data struct {
				ARCH string `json:"arch"`
				DS   string `json:"ds"`
				HN   string `json:"hn"`
				HT   string `json:"ht"`
				ID   string `json:"id"`
				IP   string `json:"ip"`
				MS   string `json:"ms"`
				NPS  string `json:"nps"`
				OS   string `json:"os"`
				PID  string `json:"pid"`
				UPT  string `json:"upt"`
				VER  string `json:"ver"`
			} `json:"data"`
			Type string `json:"type"`
		}{}

		json.Unmarshal([]byte(info), &sinfo)

		db, err := sql.Open("sqlite3", "./lnxmon.db")
		whatif(err)
		defer db.Close()

		sql := `
			UPDATE tb_host_static_info
			SET
				hostname=?,
				ip=?,
				os_type=?,
				architecture=?,
				cpu_processors=?,
				mem_size=?,
				disk_size=?,
				uptime=?,
				heart_time=?,
				client_version=?
			WHERE project_id=? AND host_id=?
		`
		stmt, err := db.Prepare(sql)
		whatif(err)
		defer stmt.Close()
		res, err := stmt.Exec(
			sinfo[0].Data.HN,
			sinfo[0].Data.IP,
			sinfo[0].Data.OS,
			sinfo[0].Data.ARCH,
			sinfo[0].Data.NPS,
			sinfo[0].Data.MS,
			sinfo[0].Data.DS,
			sinfo[0].Data.UPT,
			sinfo[0].Data.HT,
			sinfo[0].Data.VER,
			strings.ToLower(sinfo[0].Data.PID),
			sinfo[0].Data.ID,
		)
		whatif(err)

		affected, err := res.RowsAffected()
		whatif(err)
		if affected == 0 {
			sql2 := `
				INSERT INTO tb_host_static_info (
					project_id,
					host_id,
					hostname,
					ip,
					os_type,
					architecture,
					cpu_processors,
					mem_size,
					disk_size,
					uptime,
					heart_time,
					client_version
				) VALUES (
					?,?,?,?,?,?,?,?,?,?,?,?
				)
			`
			stmt2, _ := db.Prepare(sql2)
			defer stmt2.Close()
			_, err2 := stmt2.Exec(
				strings.ToLower(sinfo[0].Data.PID),
				sinfo[0].Data.ID,
				sinfo[0].Data.HN,
				sinfo[0].Data.IP,
				sinfo[0].Data.OS,
				sinfo[0].Data.ARCH,
				sinfo[0].Data.NPS,
				sinfo[0].Data.MS,
				sinfo[0].Data.DS,
				sinfo[0].Data.UPT,
				sinfo[0].Data.HT,
				sinfo[0].Data.VER,
			)
			whatif(err2)
		}
	} else if strings.Contains(info, `"type":"dinfo"`) {
		dinfo := []struct {
			Data struct {
				CPU   string `json:"cpu"`
				DIO   string `json:"dio"`
				DISK  string `json:"disk"`
				HN    string `json:"hn"`
				HT    string `json:"ht"`
				ID    string `json:"id"`
				IP    string `json:"ip"`
				LDG   string `json:"ldg"`
				MEM   string `json:"mem"`
				NIO   string `json:"nio"`
				PID   string `json:"pid"`
				SKT   string `json:"skt"`
				USERS string `json:"users"`
			} `json:"data"`
			Type string `json:"type"`
		}{}

		json.Unmarshal([]byte(info), &dinfo)

		db, err := sql.Open("sqlite3", "./lnxmon.db")
		whatif(err)
		defer db.Close()

		sql := `
			INSERT INTO tb_host_dynamic_info_%s (
				host_id,
				hostname,
				ip,
				loadavg,
				cpu_usage,
				mem_usage,
				disk_usage,
				disk_io_rate,
				nic_io_rate,
				tcp_sockets,
				users,
				heart_time
			) VALUES (
				?,?,?,?,?,?,?,?,?,?,?,?
			)
		`
		pid := strings.ToLower(dinfo[0].Data.PID)
		sql = fmt.Sprintf(sql, pid)
		stmt, err := db.Prepare(sql)
		if err != nil {
			dontcare(err)
			log.Println(err.Error())
			if strings.Contains(err.Error(), "no such table") {
				createDynamicTable(pid)
			}
			stmt, err = db.Prepare(sql)
		}
		whatif(err)
		defer stmt.Close()
		res, err := stmt.Exec(
			dinfo[0].Data.ID,
			dinfo[0].Data.HN,
			dinfo[0].Data.IP,
			dinfo[0].Data.LDG,
			dinfo[0].Data.CPU,
			dinfo[0].Data.MEM,
			dinfo[0].Data.DISK,
			dinfo[0].Data.DIO,
			dinfo[0].Data.NIO,
			dinfo[0].Data.SKT,
			dinfo[0].Data.USERS,
			dinfo[0].Data.HT,
		)
		whatif(err)

		lastInsertedId, err := res.LastInsertId()
		whatif(err)
		log.Println(lastInsertedId)

		sql2 := `
			UPDATE tb_host_static_info
			SET max_host_dynamic_info_id=?, heart_time=?
			WHERE project_id=? and host_id=?
		`
		stmt2, err := db.Prepare(sql2)
		whatif(err)
		defer stmt2.Close()
		_, err2 := stmt2.Exec(
			lastInsertedId,
			dinfo[0].Data.HT,
			strings.ToLower(dinfo[0].Data.PID),
			dinfo[0].Data.ID,
		)
		whatif(err2)
	} else {
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	log.Println(time.Now().String())

	// project_id := "TEST"
	// host_id := "83d91f4fa8e238902136848e715c82ef"

	// host_id := r.FormValue("id")
	// host_id2 := r.URL.Query().Get("id")
	// host_id3 := r.URL.Query()["id"]

	_project_id := r.FormValue("pid")
	_project_id = strings.TrimSpace(_project_id)
	_host_id := r.FormValue("id")
	user_agent := r.Header.Get("User-Agent")
	_ts := r.FormValue("ts")
	_ts = strings.TrimSpace(_ts)
	_v := r.FormValue("v")

	ts, err := strconv.Atoi(_ts)
	if err != nil {
		ts = 240
	}

	is_request_from_pc := true
	if strings.Contains(user_agent, "Android") || strings.Contains(user_agent, "iPhone") {
		is_request_from_pc = false
	}

	timeNow := time.Now()
	begin_time := timeNow.Add(-time.Minute * time.Duration(ts)).Format("2006-01-02 15:04:05")
	var end_time string
	if ts <= 1440 {
		end_time = timeNow.Format("2006-01-02 15:04:05")
	} else {
		end_time = timeNow.Add(-time.Minute * time.Duration(ts-1440)).Format("2006-01-02 15:04:05")
	}
	log.Println("ts: ", ts)
	log.Println("begin_time: ", begin_time)
	log.Println("end_time: ", end_time)

	db, err := sql.Open("sqlite3", "./lnxmon.db")
	whatif(err)
	defer db.Close()

	// var project_list []string
	var project_list []map[string]interface{}
	{
		mysql := `
			SELECT DISTINCT project_id
			FROM tb_host_static_info
			ORDER BY project_id
		`
		stmt, err := db.Prepare(mysql)
		whatif(err)
		defer stmt.Close()
		rows, err := stmt.Query()
		whatif(err)
		defer rows.Close()

		var project_id string
		for rows.Next() {
			err = rows.Scan(&project_id)
			whatif(err)
			project_list = append(
				project_list,
				map[string]interface{}{
					"id":   project_id,
					"name": strings.ToUpper(project_id),
				},
			)
			if _project_id == "" {
				_project_id = project_id
			}
		}
	}
	if _project_id == "" {
		_project_id = "default"
	}

	/*
		mysql := `
			SELECT
				project_id,
				host_id,
				hostname,
				alias,
				ip,
				os_type,
				architecture,
				cpu_processors,
				mem_size,
				disk_size,
				uptime,
				heart_time
			FROM tb_host_static_info
		`
	*/
	mysql := `
		SELECT
		  x.project_id,
		  x.host_id,
		  x.hostname,
		  x.alias,
		  x.ip,
		  x.os_type,
		  x.architecture,
		  x.cpu_processors,
		  x.mem_size,
		  x.disk_size,
		  x.uptime,
		  x.heart_time,
		  y.loadavg,
		  y.cpu_usage,
		  y.mem_usage,
		  y.disk_usage,
		  y.users
		FROM tb_host_static_info x
		JOIN tb_host_dynamic_info_%s y
		ON x.max_host_dynamic_info_id=y.rowid
		WHERE x.project_id=?
		ORDER BY x.hostname
	`
	mysql = fmt.Sprintf(mysql, _project_id)
	stmt, err := db.Prepare(mysql)
	whatif(err)
	defer stmt.Close()
	rows, err := stmt.Query(_project_id)
	// rows, err := stmt.Query()
	whatif(err)
	defer rows.Close()

	var host_list []map[string]interface{}
	var project_id string
	var host_id string
	var hostname string
	var alias string
	var _alias sql.NullString
	var ip string
	var ips []string
	var os_type string
	var architecture string
	var cpu_processors string
	var x_cpu_processors string
	var mem_size string
	var disk_size string
	var uptime string
	// var heart_time string
	var heart_time time.Time
	var _loadavg string
	var _cpu_usage string
	var _mem_usage string
	var _disk_usage string
	var _users string
	var is_overload bool = false
	var is_overcpu bool = false
	var is_overmem bool = false
	var is_overdisk bool = false

	for rows.Next() {
		is_overload = false
		is_overcpu = false
		is_overmem = false
		is_overdisk = false

		err = rows.Scan(
			&project_id,
			&host_id,
			&hostname,
			&_alias,
			&ip,
			&os_type,
			&architecture,
			&cpu_processors,
			&mem_size,
			&disk_size,
			&uptime,
			&heart_time,
			&_loadavg,
			&_cpu_usage,
			&_mem_usage,
			&_disk_usage,
			&_users,
		)
		whatif(err)
		if !_alias.Valid {
			alias = _alias.String
		}

		xs := strings.Split(_loadavg, ",")
		loadavg_1m, err := strconv.ParseFloat(xs[0], 64)
		whatif(err)
		loadavg_5m, err := strconv.ParseFloat(xs[1], 64)
		whatif(err)
		loadavg_15m, err := strconv.ParseFloat(xs[2], 64)
		whatif(err)
		_cpu_processors, err := strconv.ParseFloat(cpu_processors, 64)
		whatif(err)
		if loadavg_1m > _cpu_processors || loadavg_5m > _cpu_processors || loadavg_15m > _cpu_processors {
			is_overload = true
		}

		ys := strings.Split(_cpu_usage, ",")
		cpu_usage, err := strconv.ParseFloat(ys[0], 64)
		whatif(err)
		cpu_iowait, err := strconv.ParseFloat(ys[1], 64)
		whatif(err)
		_cpu_usage = fmt.Sprintf("%.f", cpu_usage)
		if cpu_iowait > cpu_usage {
			_cpu_usage = fmt.Sprintf("%.f", cpu_iowait)
		}
		if cpu_usage > 80 || cpu_iowait > 80 {
			is_overcpu = true
		}

		us := strings.Split(_mem_usage, ",")
		mem_usage, err := strconv.ParseFloat(us[1], 64)
		whatif(err)
		swap_usage, err := strconv.ParseFloat(us[3], 64)
		whatif(err)
		_mem_usage = fmt.Sprintf("%.f", mem_usage)
		if swap_usage > mem_usage {
			_mem_usage = fmt.Sprintf("%.f", swap_usage)
		}
		if mem_usage > 80 || swap_usage > 80 {
			is_overmem = true
		}

		var whatever float64
		qs := strings.Split(_disk_usage, ",")
		for _, v := range qs {
			ls := strings.Split(v, "_")
			disk_usage := ls[2]
			inode_usage := ls[3]
			_disk_usage, err := strconv.ParseFloat(disk_usage, 64)
			whatif(err)
			_inode_usage, err := strconv.ParseFloat(inode_usage, 64)
			whatif(err)
			if _disk_usage > whatever {
				whatever = _disk_usage
			}
			if _inode_usage > whatever {
				whatever = _inode_usage
			}
			if _disk_usage > 85 || _inode_usage > 85 {
				is_overdisk = true
			}
		}
		_disk_usage = fmt.Sprintf("%.f", whatever)

		ips = strings.Split(ip, ",")

		host_list = append(
			host_list,
			map[string]interface{}{
				"project_id":     project_id,
				"host_id":        host_id,
				"hostname":       hostname,
				"alias":          alias,
				"ip":             ip,
				"ips":            ips,
				"os_type":        os_type,
				"architecture":   architecture,
				"cpu_processors": cpu_processors,
				"mem_size":       mem_size,
				"disk_size":      disk_size,
				"uptime":         uptime,
				"heart_time":     heart_time.Format("2006-01-02 15:04:05"),
				"loadavg":        _loadavg,
				"cpu_usage":      _cpu_usage,
				"mem_usage":      _mem_usage,
				"disk_usage":     _disk_usage,
				"users":          _users,
				"is_overload":    is_overload,
				"is_overcpu":     is_overcpu,
				"is_overmem":     is_overmem,
				"is_overdisk":    is_overdisk,
			},
		)
		if _host_id == "" {
			_host_id = host_id
		}
		if _host_id == host_id {
			x_cpu_processors = cpu_processors
		}
	}

	log.Println(time.Now().String())

	mysql2 := `
		SELECT
			loadavg,
			cpu_usage,
			mem_usage,
			disk_usage,
			disk_io_rate,
			nic_io_rate,
			tcp_sockets,
			users,
			heart_time
		FROM tb_host_dynamic_info_%s
		WHERE
			host_id=?
			AND
			heart_time>=?
			AND
			heart_time<=?
		LIMIT 1440
	`
	mysql2 = fmt.Sprintf(mysql2, _project_id)
	stmt2, err := db.Prepare(mysql2)
	whatif(err)
	defer stmt2.Close()
	rows2, err := stmt2.Query(_host_id, begin_time, end_time)
	whatif(err)
	defer rows2.Close()

	var loadavg string
	var cpu_usage string
	var mem_usage string
	var disk_usage string
	var disk_io_rate string
	var nic_io_rate string
	var tcp_sockets string
	var users string
	var heart_time2 string
	var loadavg_array []map[string]interface{}
	var loadavg_1m_data []float64
	var loadavg_5m_data []float64
	var loadavg_15m_data []float64
	var cpu_usage_array []map[string]interface{}
	var cpu_usage_data []float64
	var cpu_iowait_data []float64
	var mem_usage_array []map[string]interface{}
	var mem_usage_data []float64
	var swap_usage_data []float64
	var disk_usage_array []map[string]interface{}
	var disk_tmp_array map[string][]float64
	var disk_io_rate_array []map[string]interface{}
	var disk_read_rate_data []float64
	var disk_write_rate_data []float64
	var nic_io_rate_array []map[string]interface{}
	var nic_receive_rate_data []float64
	var nic_transmit_rate_data []float64
	var tcp_sockets_array []map[string]interface{}
	var tcp_sockets_inuse_data []float64
	var tcp_sockets_timewait_data []float64
	var users_array []map[string]interface{}
	var users_data []float64
	var heart_time_array []string

	disk_tmp_array = make(map[string][]float64)

	for rows2.Next() {
		err = rows2.Scan(
			&loadavg,
			&cpu_usage,
			&mem_usage,
			&disk_usage,
			&disk_io_rate,
			&nic_io_rate,
			&tcp_sockets,
			&users,
			&heart_time2,
		)
		whatif(err)
		xs := strings.Split(loadavg, ",")
		loadavg_1m, err := strconv.ParseFloat(xs[0], 64)
		whatif(err)
		loadavg_5m, err := strconv.ParseFloat(xs[1], 64)
		whatif(err)
		loadavg_15m, err := strconv.ParseFloat(xs[2], 64)
		whatif(err)
		loadavg_1m_data = append(loadavg_1m_data, loadavg_1m)
		loadavg_5m_data = append(loadavg_5m_data, loadavg_5m)
		loadavg_15m_data = append(loadavg_15m_data, loadavg_15m)

		ys := strings.Split(cpu_usage, ",")
		cpu_usage, err := strconv.ParseFloat(ys[0], 64)
		whatif(err)
		cpu_iowait, err := strconv.ParseFloat(ys[1], 64)
		whatif(err)
		cpu_usage_data = append(cpu_usage_data, cpu_usage)
		cpu_iowait_data = append(cpu_iowait_data, cpu_iowait)

		us := strings.Split(mem_usage, ",")
		mem_usage, err := strconv.ParseFloat(us[1], 64)
		whatif(err)
		swap_usage, err := strconv.ParseFloat(us[3], 64)
		whatif(err)
		mem_usage_data = append(mem_usage_data, mem_usage)
		swap_usage_data = append(swap_usage_data, swap_usage)

		qs := strings.Split(disk_usage, ",")
		for _, v := range qs {
			ls := strings.Split(v, "_")

			mount_point := ls[0]
			disk_size := ls[1]
			disk_usage := ls[2]
			inode_usage := ls[3]
			tmp_ka := fmt.Sprintf("Disk Usage of %s (%sG)", mount_point, disk_size)
			tmp_kb := fmt.Sprintf("Inode Usage of %s (%sG)", mount_point, disk_size)

			_disk_usage, err := strconv.ParseFloat(disk_usage, 64)
			whatif(err)
			_inode_usage, err := strconv.ParseFloat(inode_usage, 64)
			whatif(err)

			disk_tmp_array[tmp_ka] = append(disk_tmp_array[tmp_ka], _disk_usage)
			disk_tmp_array[tmp_kb] = append(disk_tmp_array[tmp_kb], _inode_usage)
		}

		as := strings.Split(disk_io_rate, ",")
		disk_read_rate, err := strconv.ParseFloat(as[0], 64)
		whatif(err)
		disk_write_rate, err := strconv.ParseFloat(as[1], 64)
		whatif(err)
		disk_read_rate_data = append(disk_read_rate_data, disk_read_rate)
		disk_write_rate_data = append(disk_write_rate_data, disk_write_rate)

		bs := strings.Split(nic_io_rate, ",")
		nic_receive_rate, err := strconv.ParseFloat(bs[0], 64)
		whatif(err)
		nic_transmit_rate, err := strconv.ParseFloat(bs[1], 64)
		whatif(err)
		nic_receive_rate_data = append(nic_receive_rate_data, nic_receive_rate)
		nic_transmit_rate_data = append(nic_transmit_rate_data, nic_transmit_rate)

		cs := strings.Split(tcp_sockets, ",")
		tcp_sockets_inuse, err := strconv.ParseFloat(cs[0], 64)
		whatif(err)
		tcp_sockets_timewait, err := strconv.ParseFloat(cs[1], 64)
		whatif(err)
		tcp_sockets_inuse_data = append(tcp_sockets_inuse_data, tcp_sockets_inuse)
		tcp_sockets_timewait_data = append(tcp_sockets_timewait_data, tcp_sockets_timewait)

		users, err := strconv.ParseFloat(users, 64)
		whatif(err)
		users_data = append(users_data, users)

		heart_time_array = append(heart_time_array, heart_time2)
	}
	loadavg_array = append(loadavg_array, map[string]interface{}{"name": "loadavg_1m", "data": loadavg_1m_data})
	loadavg_array = append(loadavg_array, map[string]interface{}{"name": "loadavg_5m", "data": loadavg_5m_data})
	loadavg_array = append(loadavg_array, map[string]interface{}{"name": "loadavg_15m", "data": loadavg_15m_data})
	cpu_usage_array = append(cpu_usage_array, map[string]interface{}{"name": "cpu_usage", "data": cpu_usage_data})
	cpu_usage_array = append(cpu_usage_array, map[string]interface{}{"name": "cpu_iowait", "data": cpu_iowait_data})
	mem_usage_array = append(mem_usage_array, map[string]interface{}{"name": "mem_usage", "data": mem_usage_data})
	mem_usage_array = append(mem_usage_array, map[string]interface{}{"name": "swap_usage", "data": swap_usage_data})

	for k, v := range disk_tmp_array {
		disk_usage_array = append(disk_usage_array, map[string]interface{}{"name": k, "data": v})
	}

	disk_io_rate_array = append(disk_io_rate_array, map[string]interface{}{"name": "read_rate", "data": disk_read_rate_data})
	disk_io_rate_array = append(disk_io_rate_array, map[string]interface{}{"name": "write_rate", "data": disk_write_rate_data})
	nic_io_rate_array = append(nic_io_rate_array, map[string]interface{}{"name": "reveive_rate", "data": nic_receive_rate_data})
	nic_io_rate_array = append(nic_io_rate_array, map[string]interface{}{"name": "transmit_rate", "data": nic_transmit_rate_data})
	tcp_sockets_array = append(tcp_sockets_array, map[string]interface{}{"name": "inuse", "data": tcp_sockets_inuse_data})
	tcp_sockets_array = append(tcp_sockets_array, map[string]interface{}{"name": "timewait", "data": tcp_sockets_timewait_data})
	users_array = append(users_array, map[string]interface{}{"name": "users", "data": users_data})

	y, err := json.Marshal(loadavg_array)
	whatif(err)
	m, err := json.Marshal(cpu_usage_array)
	whatif(err)
	n, err := json.Marshal(mem_usage_array)
	whatif(err)
	e, err := json.Marshal(disk_usage_array)
	whatif(err)
	f, err := json.Marshal(disk_io_rate_array)
	whatif(err)
	p, err := json.Marshal(nic_io_rate_array)
	whatif(err)
	c, err := json.Marshal(tcp_sockets_array)
	whatif(err)
	u, err := json.Marshal(users_array)
	whatif(err)
	z, err := json.Marshal(heart_time_array)
	whatif(err)

	data := struct {
		X_project_list []map[string]interface{}
		X_host_list    []map[string]interface{}
		// X_project_id         string
		X_host_id string
		// X_hostname           string
		// X_alias              string
		// X_ip                 string
		// X_os_type            string
		// X_architecture       string
		X_cpu_processors string
		// X_mem_size           string
		// X_disk_size          string
		// X_uptime             string
		// X_heart_time         string
		X_current_id         string
		X_current_pid        string
		X_current_ts         int
		Y                    string
		Z                    string
		M                    string
		N                    string
		F                    string
		P                    string
		C                    string
		U                    string
		E                    string
		X_is_request_from_pc bool
		X_v                  string
	}{
		X_project_list: project_list,
		X_host_list:    host_list,
		// X_project_id:         project_id,
		X_host_id: _host_id,
		// X_hostname:           hostname,
		// X_alias:              alias,
		// X_ip:                 ip,
		// X_os_type:            os_type,
		// X_architecture:       architecture,
		// X_cpu_processors:     cpu_processors,
		X_cpu_processors: x_cpu_processors,
		// X_mem_size:           mem_size,
		// X_disk_size:          disk_size,
		// X_uptime:             uptime,
		// X_heart_time:         heart_time,
		X_current_id:         _host_id,
		X_current_pid:        _project_id,
		X_current_ts:         ts,
		Y:                    string(y),
		Z:                    string(z),
		M:                    string(m),
		N:                    string(n),
		F:                    string(f),
		P:                    string(p),
		C:                    string(c),
		U:                    string(u),
		E:                    string(e),
		X_is_request_from_pc: is_request_from_pc,
		X_v:                  _v,
	}

	log.Println(time.Now().String())

	HTML := ""

	if HTML == "" {
		t, _ := template.ParseFiles("templates/index.html")
		t.Execute(w, data)
	} else {
		t, _ := template.New("X").Parse(HTML)
		t.Execute(w, data)
	}

	log.Println(time.Now().String())
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
	}
}

func initDB() {
	createStaticTable()
	createDynamicTable("DEFAULT")
}

func createStaticTable() {
	db, err := sql.Open("sqlite3", "./lnxmon.db")
	whatif(err)
	defer db.Close()

	sql := "SELECT 1 FROM tb_host_static_info"
	res, err := db.Query(sql)
	dontcare(err)
	if res != nil {
		defer res.Close()
	}
	if res == nil {
		sql2 := `
			CREATE TABLE tb_host_static_info (
				project_id               VARCHAR(20)  NOT NULL,
				host_id                  CHAR(32)     NOT NULL,
				hostname                 VARCHAR(64)  DEFAULT NULL,
				alias                    VARCHAR(64)  DEFAULT NULL,
				ip                       VARCHAR(100) DEFAULT NULL,
				os_type                  VARCHAR(64)  DEFAULT NULL,
				architecture             VARCHAR(64)  DEFAULT NULL,
				cpu_processors           VARCHAR(4)   DEFAULT NULL,
				mem_size                 VARCHAR(10)  DEFAULT NULL,
				disk_size                VARCHAR(10)  DEFAULT NULL,
				uptime                   VARCHAR(10)  DEFAULT NULL,
				heart_time               DATETIME     DEFAULT NULL,
				max_host_dynamic_info_id INT          DEFAULT 0,
				client_version           VARCHAR(10)  DEFAULT NULL,
				PRIMARY KEY (project_id,host_id)
			)
		`
		_, err2 := db.Exec(sql2)
		whatif(err2)
	}
}

func createDynamicTable(pid string) {
	pid = strings.ToLower(pid)

	db, err := sql.Open("sqlite3", "./lnxmon.db")
	whatif(err)
	defer db.Close()

	sql := "SELECT 1 FROM tb_host_dynamic_info_%s"
	sql = fmt.Sprintf(sql, pid)
	res, err := db.Query(sql)
	dontcare(err)
	if res != nil {
		defer res.Close()
	}
	if res == nil {
		sql2 := `
			CREATE TABLE tb_host_dynamic_info_%s (
				id           INT UNSIGNED,
				host_id      CHAR(32)     NOT NULL,
				hostname     VARCHAR(64)  DEFAULT NULL,
				ip           VARCHAR(100) DEFAULT NULL,
				loadavg      VARCHAR(20)  DEFAULT NULL,
				cpu_usage    VARCHAR(20)  DEFAULT NULL,
				mem_usage    VARCHAR(20)  DEFAULT NULL,
				disk_usage   VARCHAR(255) DEFAULT NULL,
				disk_io_rate VARCHAR(32)  DEFAULT NULL,
				nic_io_rate  VARCHAR(32)  DEFAULT NULL,
				tcp_sockets  VARCHAR(20)  DEFAULT NULL,
				users        VARCHAR(20)  DEFAULT NULL,
				heart_time   DATETIME     DEFAULT NULL,
				PRIMARY KEY (id)
			)
		`
		sql2 = fmt.Sprintf(sql2, pid)
		_, err2 := db.Exec(sql2)
		whatif(err2)
	}
}

func main() {
	reflect.TypeOf(0)

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	_host := flag.String("host", "0.0.0.0", "Host")
	_port := flag.Int("port", 1234, "Port")
	flag.Parse()
	host := *_host
	port := strconv.Itoa(*_port)
	address := fmt.Sprintf("%s:%s", host, port)

	initDB()

	http.HandleFunc("/", makeHandler(index))
	http.HandleFunc("/favicon.ico", makeHandler(notfound))
	http.HandleFunc("/api", makeHandler(api))

	fs := http.FileServer(http.Dir("./static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println(fmt.Sprintf("ListenAndServe http://%s/", address))
	// err := http.ListenAndServe(":1234", nil)
	err := http.ListenAndServe(address, nil)

	log.Fatal(err)
}
