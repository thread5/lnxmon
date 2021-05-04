import atexit
import datetime
import hashlib
import logging
import logging.handlers
import os
import re
import socket
import subprocess
import sys
import threading
import time
import traceback
import urllib
import urllib2
from signal import SIGTERM
try: import simplejson as json
except ImportError: import json


project_id = 'TEST'
# project_id = 'PROD'
# server = 'http://127.0.0.1:9000/linux_monitor'
server = 'http://10.8.0.1:5001/linux_monitor'
version = '20170909'


filename = '/tmp/linux_monitor.log'
maxBytes = 1 * 1024 * 1024
backupCount = 1
level = logging.INFO

fmt = (
    '%(asctime)s,%(msecs)03d '
    '[%(name)s:%(funcName)s:%(lineno)d] '
    '%(levelname)-5s - %(message)s'
)
datefmt = '%Y-%m-%d %H:%M:%S'
formatter = logging.Formatter(fmt=fmt, datefmt=datefmt)

handler = logging.handlers.RotatingFileHandler(
    filename, maxBytes=maxBytes, backupCount=backupCount
)
handler.setFormatter(formatter)

logger = logging.getLogger()
logger.setLevel(level)
logger.addHandler(handler)



class Daemon:
    def __init__(self, pidfile, stdin=None, stdout=None, stderr=None):
        self.pidfile = pidfile
        self.stdin = stdin if stdin else '/dev/null'
        self.stdout = stdout if stdout else '/dev/null'
        self.stderr = stderr if stderr else '/dev/null'


    def daemonize(self):
        # Do the UNIX double-fork magic
        try:
            pid = os.fork()
            if pid > 0:
                # Exit first parent
                sys.exit(0)
        except OSError as e:
            msg = 'fork #1 failed: {0} ({1})\n'
            sys.stderr.write(msg.format(e.errno, e.strerror))
            sys.exit(1)

        # Decouple from parent environment
        os.chdir('/tmp')
        os.setsid()
        os.umask(0)

        # Do second fork
        try:
            pid = os.fork()
            if pid > 0:
                # Exit from second parent
                sys.exit(0)
        except OSError as e:
            msg = 'fork #2 failed: {0} ({1})\n'
            sys.stderr.write(msg.format(e.errno, e.strerror))
            sys.exit(1)

        # Redirect standard file descriptors
        sys.stdout.flush()
        sys.stderr.flush()
        f0 = open(self.stdin, 'r')
        f1 = open(self.stdout, 'a+')
        f2 = open(self.stderr, 'a+')
        os.dup2(f0.fileno(), sys.stdin.fileno())
        os.dup2(f1.fileno(), sys.stdout.fileno())
        os.dup2(f2.fileno(), sys.stderr.fileno())

        # Write pidfile, atexit (at exit)
        atexit.register(self.delpid)
        pid = str(os.getpid())
        open(self.pidfile, 'w+').write('{0}\n'.format(pid))


    def delpid(self):
        os.remove(self.pidfile)


    def start(self):
        # Check for a pidfile to see if the daemon already runs
        try:
            f = open(self.pidfile, 'r')
            pid = int(f.read().strip())
            f.close()
        except IOError:
            pid = None

        if pid:
            msg = (
                'The process(pid {0}, pidfile {1}) '
                'is already running...\n'
            )
            sys.stderr.write(msg.format(pid, self.pidfile))
            sys.exit(1)
        else:
            # Start the daemon
            self.daemonize()
            self.run()


    def stop(self):
        # Get the pid from the pidfile
        try:
            f = open(self.pidfile, 'r')
            pid = int(f.read().strip())
            f.close()
        except IOError:
            pid = None

        # Not an error in a restart
        if not pid:
            msg = 'The process is already stopped\n'
            sys.stderr.write(msg)
            return

        # Try killing the daemon process
        try:
            while 1:
                os.kill(pid, SIGTERM)
                time.sleep(0.1)
        except OSError as err:
            err = str(err)
            if err.find('No such process') > 0:
                if os.path.exists(self.pidfile):
                    os.remove(self.pidfile)
            else:
                print(str(err))
                sys.exit(1)


    def restart(self):
        self.stop()
        self.start()


    def status(self):
        try:
            f = open(self.pidfile, 'r')
            pid = int(f.read().strip())
            f.close()
        except IOError:
            pid = None

        if pid:
            msg = 'The process(pid %s, pidfile %s) is running...\n'
            sys.stderr.write(msg % (pid, self.pidfile))
            sys.exit(1)
        else:
            msg = 'The process is stopped\n'
            sys.stderr.write(msg)
            sys.exit(1)


    def run(self):
        '''
        You should override this method when you subclass Daemon.
        It will be called after the process has been daemonized
        by start() or restart(). '''



''' Host basic information '''
'''
from echo -n `hostname` | md5sum, may duplicated
return 2e5ec7f2963e9f4c66761fb151be33c7
'''
def get_id():
    return hashlib.md5(socket.gethostname()).hexdigest()


'''
from socket.gethostname()
return localhost
'''
def get_hostname():
    return socket.gethostname()


'''
from ifconfig, /sbin/ifconfig
return 172.17.1.1,192.168.2.109 // Exclude 127.0.0.1
'''
def get_ip_deprecated():
    p = subprocess.Popen(['ifconfig'], stdout=subprocess.PIPE, shell=True)
    ifconfig_res = p.communicate()
    pattern = re.compile(r'inet\s*\w*\S*:\s*(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})')
    ip_list = pattern.findall(ifconfig_res[0])
    ip_list.sort()
    try:
        ip_list.remove('127.0.0.1')
    except:
        logger.error(traceback.format_exc())

    return ','.join(ip_list)


'''
from ip, /sbin/ip -4 a
return 172.17.1.1,192.168.2.109 // Exclude 127.0.0.1
'''
def get_ip():
    p = subprocess.Popen('ip -family inet address', stdout=subprocess.PIPE, shell=True)
    if p.wait() == 0:
        ip_res = p.communicate()
        pattern = re.compile(r'inet\s*(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})')
        ip_list = pattern.findall(ip_res[0])
        ip_list.sort()
        try:
            ip_list.remove('127.0.0.1')
        except:
            logger.error(traceback.format_exc())
        return ','.join(ip_list)
    else:
        return 'UNKOWN IP'


''' Host static information '''
'''
from /etc/issue or /etc/centos-release
return Debian GNU/Linux 7 \n \l // man issue
'''
def get_os_type():
    ver = '/etc/centos-release' if os.path.isfile('/etc/centos-release') else '/etc/issue'
    f = open(ver, 'r')
    ostype = f.readline().strip()
    f.close()

    return ostype


'''
from uname -rm
return i686 // 3.2.0-4-686-pae i686 .. command lscpu is not universal
'''
def get_architecture():
    p = subprocess.Popen('getconf LONG_BIT |head -c -1; echo -n "-bit "; uname -rm', stdout=subprocess.PIPE, shell=True)
    if p.wait() == 0:
        arch_res = p.communicate()
        return arch_res[0].strip()
    else:
        return 'UNKOWN ARCH'


'''
from /proc/cpuinfo
return 2
logical CPU(s) = Socket(s) * Core(s) per socket * Thread(s) per core(HT)
'''
def get_cpu_processors():
    processors = 0
    f = open('/proc/cpuinfo', 'r')
    for line in f.readlines():
        if line.startswith('processor'):
            processors += 1
    f.close()

    return processors


'''
from /proc/meminfo
return 2 // GB
'''
def get_mem_size():
    f = open('/proc/meminfo', 'r')
    for line in f.readlines():
        if line.startswith('MemTotal:'):
            mem_size = line.split(' ')[-2]
            mem_size = '%.0f' % (round((int(mem_size) * 1.0) / (1024 * 1024)))
            break

    return mem_size


'''
from /proc/mounts
return 200 // GB
'''
def get_disk_size():
    disk_size = 0
    f = open('/proc/mounts', 'r')
    for line in f.readlines():
        if (line.startswith(r'/dev') and
            'chroot' not in line and
            'docker' not in line):
            line = line.split()
            mount_point = line[1]
            disk = os.statvfs(mount_point)
            total = disk.f_frsize * disk.f_blocks
            disk_size += round((total * 1.0) / (1024 * 1024 * 1024))
    f.close()

    return '%.0f' % disk_size


'''
from /proc/uptime
return 30855.26 // day
'''
def get_uptime():
    f = open('/proc/uptime', 'r')
    uptime = f.readline().split(' ')[0]
    f.close()

    return '%.2f' % (float(uptime) / 86400)


# Host dynamic information
'''
from /proc/loadavg
return 0.01,0.1,0.18 // 1m,5m,15m
'''
def get_loadavg():
    loadavg = os.getloadavg()

    return '%s,%s,%s' % (loadavg[0], loadavg[1], loadavg[2])


'''
from /proc/stat
return 2.08,0.00 // cpu_usage%,iowait%
cpu  9073694 130994 42043324 4582770963 36425573 3539 3876589 0 0
cpu  68755793 74824 39542355 10929030433 79957752 3218931 3929497 0
     user,    nice, system,  idle,       iowait,  irq,    softirq, steal, guest, guest_nice
     calculate 2~8
'''
def get_cpu_usage():
    f = open('/proc/stat', 'r')
    line = f.readline().split()
    user = line[1]
    nice = line[2]
    system = line[3]
    idle = line[4]
    iowait = line[5]
    irq = line[6]
    softirq = line[7]
    f.close()
    used = int(user) + int(nice) + int(system)
    total = int(user) + int(nice) + int(system) + int(idle) + int(iowait) + int(irq) + int(softirq)

    time.sleep(1)

    f = open('/proc/stat', 'r')
    line = f.readline().split()
    user2 = line[1]
    nice2 = line[2]
    system2 = line[3]
    idle2 = line[4]
    iowait2 = line[5]
    irq2 = line[6]
    softirq2 = line[7]
    f.close()
    used2 = int(user2) + int(nice2) + int(system2)
    total2 = int(user2) + int(nice2) + int(system2) + int(idle2) + int(iowait2) + int(irq2) + int(softirq2)

    cpu_usage = (int(used2) - int(used)) / (float(total2) - float(total))
    iowait_usage = (int(iowait2) - int(iowait)) / (float(total2) - float(total))

    return '%.2f,%.2f' % (cpu_usage * 100, iowait_usage * 100)


'''
from /proc/meminfo
return 3,23.15,0.00 // mem_total(G),mem_usage%,swap_total(G),swap_usage%
'''
def get_mem_usage():
    f = open('/proc/meminfo', 'r')
    for line in f.readlines():
        if line.startswith('MemTotal:'):
            memtotal = line.split(' ')[-2]
        elif line.startswith('MemFree:'):
            memfree = line.split(' ')[-2]
        elif line.startswith('Buffers:'):
            buffers = line.split(' ')[-2]
        elif line.startswith('Cached:'):
            cached = line.split(' ')[-2]
        elif line.startswith('SwapTotal:'):
            swaptotal = line.split(' ')[-2]
        elif line.startswith('SwapFree:'):
            swapfree = line.split(' ')[-2]
    f.close()

    mem_usage = (int(memtotal) - int(memfree) - int(buffers) - int(cached)) / float(memtotal) * 100
    swap_usage = (int(swaptotal) - int(swapfree)) / (float(swaptotal) + 0.1) * 100

    return '%.0f,%.2f,%.0f,%.2f' % (round((int(memtotal) * 1.0) / (1024 * 1024)), mem_usage,
                                    round((int(swaptotal) * 1.0) / (1024 * 1024)), swap_usage)


'''
from /proc/mounts
return /_82_13.36_5.46,/root_137_67.16_11.09 // mountPoint_diskTotal(G)_diskUsage%_inodeUsage%,..
struct statvfs {
  unsigned long  f_bsize;    /* filesystem block size */
  unsigned long  f_frsize;   /* fragment size */
  fsblkcnt_t     f_blocks;   /* size of fs in f_frsize units */
  fsblkcnt_t     f_bfree;    /* # free blocks */
  fsblkcnt_t     f_bavail;   /* # free blocks for unprivileged users */
  fsfilcnt_t     f_files;    /* # inodes */
  fsfilcnt_t     f_ffree;    /* # free inodes */
  fsfilcnt_t     f_favail;   /* # free inodes for unprivileged users */
  unsigned long  f_fsid;     /* filesystem ID */
  unsigned long  f_flag;     /* mount flags */
  unsigned long  f_namemax;  /* maximum filename length */
};
unsigned long f_frsize   Fundamental file system block size.
fsblkcnt_t    f_blocks   Total number of blocks on file system in units of f_frsize.
fsblkcnt_t    f_bfree    Total number of free blocks.
fsblkcnt_t    f_bavail   Number of free blocks available to non-privileged process.
statvfs.frsize * statvfs.f_blocks     # Size of filesystem in bytes
statvfs.frsize * statvfs.f_bfree      # Actual number of free bytes
statvfs.frsize * statvfs.f_bavail     # Number of free bytes that ordinary users are allowed to use (excl. reserved space)
disk.f_frsize = 4096(physical), sector = 512(logical)
'''
def get_disk_usage():
    disk_list = []

    f = open('/proc/mounts', 'r')
    for line in f.readlines():
        # Exclude bind auto mount
        if (line.startswith(r'/dev') and
            'chroot' not in line and
            'docker' not in line):
            line = line.split()
            mount_point = line[1]

            disk = os.statvfs(mount_point)
            total = disk.f_frsize * disk.f_blocks
            used = disk.f_frsize * (disk.f_blocks - disk.f_bfree)
            if disk.f_files != 0:
                inode_usage = ((disk.f_files - disk.f_ffree) * 1.0 / disk.f_files) * 100
            else:
                inode_usage = 0
            disk_list.append('%s_%.0f_%.2f_%.2f' % (mount_point, round((total * 1.0) / (1024 * 1024 * 1024)), (used * 1.0 / total) * 100, inode_usage))

    f.close()

    return ','.join(disk_list)


'''
from /proc/diskstats // (hd|sd|xvd)[a-z][0-9], xv is virtual drive
return 0,16,0 // read_rate(KB/s),write_rate(KB/s),current_requests
8       0 sda 476885 44795 85874852 3075176 345021 1106298 82592001 139788368 0 2654052 142863508
f3 85874852 # of sectors read
f4 3075176 # of  milliseconds spent reading
f7 82592001 # of sectors written
f8 139788368 # of milliseconds spent writing
f9 0 # of I/Os currently in progress
'''
def get_disk_io_rate():
    disk_type = 0

    regexp = re.compile(r'sd[a-z] ')
    regexp2 = re.compile(r'xvd[a-z] ')
    regexp3 = re.compile(r'xvd[a-z][0-9] ')
    regexp4 = re.compile(r'vd[a-z] ')
    regexp5 = re.compile(r'vd[a-z][0-9] ')
    f = open('/proc/diskstats', 'r')
    for line in f.readlines():
        if regexp.search(line) is not None:
            break
        elif regexp2.search(line) is not None:
            disk_type = 2
            break
        elif regexp3.search(line) is not None:
            disk_type = 3
            break
        elif regexp4.search(line) is not None:
            disk_type = 4
            break
        elif regexp5.search(line) is not None:
            disk_type = 5
            break
    f.close()

    if disk_type == 0:
        read_rate, write_rate, current_ios = parse_disk_io_rate(regexp)
    elif disk_type == 2:
        read_rate, write_rate, current_ios = parse_disk_io_rate(regexp2)
    elif disk_type == 3:
        read_rate, write_rate, current_ios = parse_disk_io_rate(regexp3)
    elif disk_type == 4:
        read_rate, write_rate, current_ios = parse_disk_io_rate(regexp4)
    elif disk_type == 5:
        read_rate, write_rate, current_ios = parse_disk_io_rate(regexp5)

    return '%d,%d,%d' % (read_rate, write_rate, current_ios)


def parse_disk_io_rate(regexp):
    rsectors = 0
    wsectors = 0
    rsectors2 = 0
    wsectors2 = 0
    current_ios = 0

    f = open('/proc/diskstats', 'r')
    for line in f.readlines():
        if regexp.search(line) is not None:
            line = line.split()
            rsectors += int(line[5])
            wsectors += int(line[9])
    f.close()
    time.sleep(1)
    f = open('/proc/diskstats', 'r')
    for line in f.readlines():
        if regexp.search(line) is not None:
            line = line.split()
            rsectors2 += int(line[5])
            wsectors2 += int(line[9])
            current_ios += int(line[11])
    f.close()

    read_rate = (rsectors2 - rsectors) * 512 / 1024
    write_rate = (wsectors2 - wsectors) * 512 / 1024

    return (read_rate, write_rate, current_ios)


'''
from /proc/net/dev
return 0.06,1,0.14,2 // receive_rate(KB/s),receive_packets,transmit_rate(KB/s),transmit_packets
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 6736440   14950    0    0    0     0          0         0  6736440   14950    0    0    0     0       0          0
 wlan0:  145644    1735    0    0    0 15990          0         0    13270      83   20    0    0     0       0          0
  eth0: 64299943  310617    0    0    0     0          0         0  8566720   65321    0    0    0     0       0          0
'''
def get_nic_io_rate():
    receive_bytes = 0
    transmit_bytes = 0
    receive_packets = 0
    transmit_packets = 0
    f = open('/proc/net/dev', 'r')
    for line in f.readlines():
        if 'Inter' in line or 'face' in line or 'lo:' in line:
            continue
        else:
            line = line.split(':')[-1].strip().split()
            receive_bytes += int(line[0])
            receive_packets += int(line[1])
            transmit_bytes += int(line[8])
            transmit_packets += int(line[9])
    f.close()

    time.sleep(1)
    receive_bytes2 = 0
    transmit_bytes2 = 0
    receive_packets2 = 0
    transmit_packets2 = 0
    f = open('/proc/net/dev', 'r')
    for line in f.readlines():
        if 'Inter' in line or 'face' in line or 'lo:' in line:
            continue
        else:
            line = line.split(':')[-1].strip().split()
            receive_bytes2 += int(line[0])
            receive_packets2 += int(line[1])
            transmit_bytes2 += int(line[8])
            transmit_packets2 += int(line[9])
    f.close()

    receive_bytes_rate = (int(receive_bytes2) - int(receive_bytes)) * 1.0 / 1024
    receive_packets_rate = int(receive_packets2) - int(receive_packets)
    transmit_bytes_rate = (int(transmit_bytes2) - int(transmit_bytes)) * 1.0 / 1024
    transmit_packets_rate = int(transmit_packets2) - int(transmit_packets)

    return '%.2f,%s,%.2f,%s' % (receive_bytes_rate, receive_packets_rate, transmit_bytes_rate, transmit_packets_rate)


'''
from /proc/net/sockstat
return 11,0 // inuse,timewait
'''
def get_tcp_sockets():
    inuse = 0
    tw = 0

    f = open('/proc/net/sockstat', 'r')
    for line in f.readlines():
        if line.startswith('TCP:'):
            line = line.split()
            inuse += int(line[2])
            tw += int(line[6])
    f.close()

    if os.path.exists('/proc/net/sockstat6'):
        f2 = open('/proc/net/sockstat6', 'r')
        for line in f2.readlines():
            if line.startswith('TCP6:'):
                line = line.split()
                inuse += int(line[2])
        f2.close()

    return '%s,%s' % (inuse, tw)


''' Common '''
'''
localtime also datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
'''
def get_current_local_time():
    return time.strftime('%Y-%m-%d %H:%M:%S')


def get_current_utc_time():
    return datetime.datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S")


'''
id   host_id
hn   hostname
ip   ip
os   os_type
arch architecture
nps  cpu_processors
upt  uptime
ht   current_utc_time(heart_time)
ver  version
pid  project_id
type sinfo static_info
return [{"type": "sinfo", "data": {"hn": "localhost", ..}}]
'''
def encode_static_info():
    json_array = []
    json_obj = {}
    static_info = {}
    static_info['id']   = get_id()
    static_info['hn']   = get_hostname()
    static_info['ip']   = get_ip()
    static_info['os']   = get_os_type()
    static_info['arch'] = get_architecture()
    static_info['nps']  = get_cpu_processors()
    static_info['ms']   = get_mem_size()
    static_info['ds']   = get_disk_size()
    static_info['upt']  = get_uptime()
    static_info['ht']   = get_current_local_time()
    static_info['ver']  = version
    static_info['pid']  = project_id
    json_obj['type'] = 'sinfo'
    json_obj['data'] = static_info
    json_array.append(json_obj)

    return json.dumps(json_array, encoding='utf-8')


'''
id   host_id
hn   hostname
ip   ip
ldg  loadavg
cpu  cpu_usage
mem  mem_usage
disk disk_usage
dio  disk_io_rate
nio  nic_io_rate
skt  tcp_sockets
ht   current_utc_time(heart_time)
pid  project_id
type dinfo dynamic info
return [{"type": "dinfo", "data": {"hn": "localhost", ..}}]
'''
def encode_dynamic_info():
    json_array = []
    json_obj = {}
    dynamic_info = {}
    dynamic_info['id']   = get_id()
    dynamic_info['hn']   = get_hostname()
    dynamic_info['ip']   = get_ip()
    dynamic_info['ldg']  = get_loadavg()
    dynamic_info['cpu']  = get_cpu_usage()
    dynamic_info['mem']  = get_mem_usage()
    dynamic_info['disk'] = get_disk_usage()
    dynamic_info['dio']  = get_disk_io_rate()
    dynamic_info['nio']  = get_nic_io_rate()
    dynamic_info['skt']  = get_tcp_sockets()
    dynamic_info['ht']   = get_current_local_time()
    dynamic_info['pid']  = project_id
    json_obj['type'] = 'dinfo'
    json_obj['data'] = dynamic_info
    json_array.append(json_obj)

    return json.dumps(json_array, encoding='utf-8')


def do_http_post(url, action, info, token):
    data = urllib.urlencode([('action', action), ('info', info), ('token', token)])
    request = urllib2.Request(url, data)
    response = urllib2.urlopen(request, timeout=20)
    http_code = response.code
    response.close()

    return http_code


def sys_print_out():
    print('HOST STATIC INFORMATION')
    print('ID')
    print('-- %s' % get_id())
    print('Hostname')
    print('-- %s' % get_hostname())
    print('IP')
    print('-- %s' % get_ip())
    print('OS Type')
    print('-- %s' % get_os_type())
    print('Architecture')
    print('-- %s' % get_architecture())
    print('CPU Processors')
    print('-- %s' % get_cpu_processors())
    print('Mem Size(G)')
    print('-- %s' % get_mem_size())
    print('Disk Size(G)')
    print('-- %s' % get_disk_size())
    print('Uptime(days)')
    print('-- %s' % get_uptime())
    print('Current UTC Time')
    print('-- %s' % get_current_local_time())
    print('')
    print('HOST DYNAMIC INFORMATION')
    print('Loadavg(1m,5m,15m)')
    print('-- %s' % get_loadavg())
    print('CPU Usage(cpu_usage%,iowait%)')
    print('-- %s' % get_cpu_usage())
    print('Mem Usage(mem_total(G),mem_usage%,swap_total(G),swap_usage%)')
    print('-- %s' % get_mem_usage())
    print('Disk Usage(mountPoint_diskTotal(G)_diskUsage%_inodeUsage%,..)')
    print('-- %s' % get_disk_usage().replace(',', '\n   '))
    print('Disk I/O Rate(read_rate(KB/s),write_rate(KB/s),current_requests)')
    print('-- %s' % get_disk_io_rate())
    print('NIC I/O Rate(receive_rate(KB/s),receive_packets,transmit_rate(KB/s),transmit_packets)')
    print('-- %s' % get_nic_io_rate())
    print('TCP Sockets(inuse,timewait)')
    print('-- %s' % get_tcp_sockets())
    print('')
    print('HOST STATIC INFORMATION')
    print(encode_static_info())
    print('')
    print('HOST DYNAMIC INFORMATION')
    print(encode_dynamic_info())


'''
Report host static info every 10 minutes
'''
def thread_10():
    while True:
        try:
            report_url = r'%s/api.php' % server
            encoded_static_info = encode_static_info()
            logger.info(encoded_static_info)
            do_http_post(report_url, 'report', encoded_static_info, '123456')
        except:
            logger.error(traceback.format_exc())
        time.sleep(600)


'''
Report host dynamic info every 1 minute
'''
def thread_1():
    while True:
        try:
            report_url = r'%s/api.php' % server
            encoded_dynamic_info = encode_dynamic_info()
            logger.info(encoded_dynamic_info)
            do_http_post(report_url, 'report', encoded_dynamic_info, '123456')
        except:
            logger.error(traceback.format_exc())
        time.sleep(60)


'''
Spawn two threads for reporting data
'''
def spawn_threads():
    threads = []
    thread10 = threading.Thread(target=thread_10)
    thread10.start()
    threads.append(thread10)

    thread1 = threading.Thread(target=thread_1)
    thread1.start()
    threads.append(thread1)

    for t in threads:
        t.join()


class MyDaemon(Daemon):
    def run(self):
        spawn_threads()


def main(argv):
    daemon = MyDaemon('/tmp/monitor_client.pid')

    usage = 'Usage: %s {start|stop|restart|status|test|version}' % argv[0]
    if len(argv) == 2:
        if argv[1] == 'start':
            daemon.start()
        elif argv[1] == 'stop':
            daemon.stop()
        elif argv[1] == 'restart':
            daemon.restart()
        elif argv[1] == 'status':
            daemon.status()
        elif argv[1] == 'test':
            sys_print_out()
        elif argv[1] == 'version':
            print(version)
        else:
            print(usage)
            sys.exit(2)
        sys.exit(0)
    else:
        print(usage)
        sys.exit(2)


if __name__ == '__main__':
    main(sys.argv)
