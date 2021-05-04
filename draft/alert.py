import datetime
import sqlite3

sending_list = []

conn = sqlite3.connect('./lnxmon.db')
conn.row_factory = sqlite3.Row
cursor = conn.cursor()
sql = '''
    SELECT
        project_id,
        host_id,
        max_host_dynamic_info_id,
        hostname,
        ip,
        cpu_processors
    FROM tb_host_static_info
'''
cursor.execute(sql)
result = cursor.fetchall()
for row in result:
    project_id = row['project_id']
    host_id = row['host_id']
    max_host_dynamic_info_id = row['max_host_dynamic_info_id']
    hostname = row['hostname']
    ip = row['ip']
    cpu_processors = row['cpu_processors']
    hostname_ip = '{}:{}'.format(hostname, ip)
    sql2 = '''
        SELECT loadavg,disk_usage,heart_time
        FROM tb_host_dynamic_info_{}
        WHERE rowid=?
    '''
    sql2 = sql2.format(project_id)
    t = (max_host_dynamic_info_id,)
    cursor.execute(sql2, t)
    result = cursor.fetchone()
    loadavg = result['loadavg']
    disk_usage = result['disk_usage']
    heart_time = result['heart_time']

    heart_time= datetime.datetime.strptime(
        heart_time, '%Y-%m-%d %H:%M:%S'
    )
    expected_heart_time = (
        datetime.datetime.now() - datetime.timedelta(minutes=10)
    )
    if heart_time < expected_heart_time:
        sending_list.append('heart_time::{}::{}'.format(
            hostname_ip, heart_time
        ))

    loadavg_1m, loadavg_5m, loadavg_15m = loadavg.split(',')
    if float(loadavg_15m) > float(cpu_processors) * 2:
        sending_list.append('loadavg::{}::{}::{}'.format(
            hostname_ip, cpu_processors, loadavg
        ))

    disks = disk_usage.split(',')
    for disk in disks:
        _, _, disk_usage2, inode_usage = disk.split('_')
        if float(disk_usage2) > 90 or float(inode_usage) > 90:
            sending_list.append('disk_usage::{}::{}'.format(
                hostname_ip, disk
            ))

cursor.close()
conn.close()

print(list(set(sending_list)))
