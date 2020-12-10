# Greenplum Exporter

使用go语言开发的Prometheus Greenplum监控数据采集器。

**项目地址：**

- Github: https://github.com/ChrisYuan/greenplum_exporter

**编译安装：**

- 下面以CentOS系统为例演示，其他平台编译方法类似

### 1.编译方法

#### 1.1 Go环境安装
```
wget https://gomirrors.org/dl/go/go1.14.12.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.14.12.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.io,direct
```

#### 1.2 Exporter编译
```
git clone https://github.com/ChrisYuan/greenplum_exporter
cd greenplum_exporter/ && make build
cd bin && ls -al
```

### 2.采集器使用
#### 2.1 将打包好的软件上传到服务器，通过以下命令行参数启动服务：
```
export GPDB_DATA_SOURCE_URL=postgres://gpadmin:password@192.168.0.100:5432/postgres?sslmode=disable
./greenplum_exporter --web.listen-address="0.0.0.0:5433" --web.telemetry-path="/metrics" --log.level=error --greenplumVersion=gposs6
```

服务启动后，访问监控指标的URL地址： *http://127.0.0.1:5433/metrics* 确定是否已经开始正常采集指标

**其中有几点需要注意：**
- 环境变量GPDB_DATA_SOURCE_URL指定了连接Greenplum数据库的连接串（请使用gpadmin账号连接postgres库），该连接串以postgres://为前缀，具体格式如下：
```
postgres://gpadmin:password@192.168.0.100:5432/postgres?sslmode=disable
postgres://[数据库连接账号，必须为gpadmin]:[账号密码，即gpadmin的密码]@[数据库的IP地址]:[数据库端口号]/[数据库名称，必须为postgres]?[参数名]=[参数值]&[参数名]=[参数值]
```
- --web.listen-address如果不定义，默认端口为5433。详见帮助
- --web.telemetry-path如果不定义，默认为/metrics。详见帮助
- --greenplumVersion如果不定义，默认为gposs6，其他选项还有：gposs5,gpdb6,gpdb5。详见帮助

**帮助：**

```
usage: greenplum_exporter [<flags>]

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long and --help-man).
      --web.listen-address="0.0.0.0:5433"
                               web endpoint
      --web.telemetry-path="/metrics"
                               path to expose metrics.
      --disableDefaultMetrics  do not report default metrics(go metrics and process metrics)
      --greenplumVersion="gposs6"
                               greenplum Server Version, options: gposs5-open source greenplum 5.x, gposs6-open source greenplum 6.x, gpdb5-pivotal greenplum 5.x,
                               gpdb6-pivotal greenplum 6.x
      --version                Show application version.
      --log.level="info"       Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
      --log.format="logger:stderr"
                               Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true"
```

#### 2.2 对应的Grafana json位于greenplum_exporter文件夹下，将对应的版本加载到Grafana并调整显示界面即可。

### 3.支持的监控指标

| No. | 指标名称	| 类型 | 标签组 |	度量单位 |	指标描述	| 数据源获取方法 |GP版本|
|:----:|:----|:----|:----|:----|:----|:----|:----|
|  1 | greenplum_cluster_state	| Gauge| version; master(master主机名)；standby(standby主机名) | boolean	| gp 可达状态 ?：1→ 可用;0→ 不可用 | SELECT count(\*) from gp_dist_random('gp_id'); select version(); SELECT hostname from gp_segment_configuration where content=-1 and role='p'; |ALL|
|  2 | greenplum_cluster_uptime | Gauge | - | int | 启动持续的时间 | select extract(epoch from now() - pg_postmaster_start_time()); |ALL|
|  3 | greenplum_cluster_sync | Gauge | - | int | Master同步Standby状态? 1→ 正常;0→ 异常 | SELECT count(*) from pg_stat_replication where state='streaming' |ALL|
|  4 | greenplum_cluster_max_connections | Gauge | - | int | 最大连接个数 | show max_connections; show superuser_reserved_connections; |ALL|
|  5 | greenplum_cluster_total_connections	| Gauge | - |	int |	当前连接个数	| select count(\*) total, count(\*) filter(where current_query='<IDLE>') idle, count(\*) filter(where current_query<>'<IDLE>') active, count(\*) filter(where current_query<>'<IDLE>' and not waiting) running, count(\*) filter(where current_query<>'<IDLE>' and waiting) waiting from pg_stat_activity where procpid <> pg_backend_pid(); |ALL|
|  6 | greenplum_cluster_idle_connections | Gauge| - | int |	idle连接数 | 同上 |ALL|
|  7 | greenplum_cluster_active_connections | Gauge | - | int | active query | 同上 |ALL|
|  8 | greenplum_cluster_running_connections	| Gauge |	- | int |	query executing | 同上 |ALL|
|  9 | greenplum_cluster_waiting_connections	| Gauge | - | int | query waiting execute | 同上 |ALL|
| 10 | greenplum_node_segment_status | Gauge | hostname; address; dbid; content; preferred_role; port; replication_port | int	| segment的状态status: 1(U)→ up; 0(D)→ down | select * from gp_segment_configuration; |ALL|
| 11 | greenplum_node_segment_role | Gauge | hostname; address; dbid; content; preferred_role; port; replication_port | int	| segment的role角色: 1(P)→ primary; 2(M)→ mirror | 同上 |ALL|
| 12 | greenplum_node_segment_mode | Gauge | hostname; address; dbid; content; preferred_role; port; replication_port | int | segment的mode：1(S)→ Synced; 2(R)→ Resyncing; 3(C)→ Change Tracking; 4(N)→ Not Syncing | 同上|ALL|
| 13 | greenplum_node_segment_disk_free_mb_size | Gauge | hostname | MB | segment主机磁盘空间剩余大小（MB) | SELECT dfhostname as segment_hostname,sum(dfspace)/count(dfspace)/(1024*1024) as segment_disk_free_gb from gp_toolkit.gp_disk_free GROUP BY dfhostname|ALL|
| 14 | greenplum_cluster_total_connections_per_client | Gauge | client | int | 每个客户端的total连接数 |select usename, count(*) total, count(*) filter(where current_query='<IDLE>') idle, count(*) filter(where current_query<>'<IDLE>') active from pg_stat_activity group by 1; |ALL|
| 15 | greenplum_cluster_idle_connections_per_client | Gauge | client |	int |	每个客户端的idle连接数 | 同上 |ALL|
| 16 | greenplum_cluster_active_connections_per_client | Gauge | client |	int |	每个客户端的active连接数 | 同上 |ALL|
| 17 | greenplum_cluster_total_online_user_count | Gauge	| - | int | 在线账号数 |	同上 |ALL|
| 18 | greenplum_cluster_total_client_count  | Gauge | - |	int |	当前所有连接的客户端个数 | 同上 |ALL|
| 19 | greenplum_cluster_total_connections_per_user | Gauge |	usename |	int |	每个账号的total连接数	| select client_addr, count(*) total, count(*) filter(where current_query='<IDLE>') idle, count(*) filter(where current_query<>'<IDLE>') active from pg_stat_activity group by 1; |ALL|
| 20 | greenplum_cluster_idle_connections_per_user | Gauge | usename | int | 每个账号的idle连接数 | 同上 |ALL|
| 21 | greenplum_cluster_active_connections_per_user | Gauge | usename | int | 每个账号的active连接数 | 同上 |ALL|
| 22 | greenplum_cluster_config_last_load_time_seconds | Gauge	| - | int | 系统配置加载时间 |	SELECT pg_conf_load_time()  |Only GPOSS6 and GPDB6
| 23 | greenplum_node_database_name_mb_size | Gauge | dbname | MB | 每个数据库占用的存储空间大小 |  SELECT dfhostname as segment_hostname,sum(dfspace)/count(dfspace)/(1024*1024) as segment_disk_free_gb from gp_toolkit.gp_disk_free GROUP BY dfhostname |ALL|
| 24 | greenplum_node_database_table_total_count | Gauge | dbname | - | 每个数据库内表的总数量 | SELECT count(*) as total from information_schema.tables where table_schema not in ('gp_toolkit','information_schema','pg_catalog');  |ALL|
| 25 | greenplum_exporter_total_scraped | Counter	| -| int | - | - |ALL|
| 26 | greenplum_exporter_total_error | Counter	| - | int	| - | - |ALL|
| 27 | greenplum_exporter_scrape_duration_second | Gauge	| - | int | - |	- |ALL|
| 28 | greenplum_server_users_name_list | Gauge	| - | int | 用户总数 |	SELECT usename from pg_catalog.pg_user; |ALL|
| 29 | greenplum_server_users_total_count | Gauge	| - | int | 用户明细 |	同上 |ALL|
| 30 | greenplum_server_locks_table_detail | Gauge	| pid;datname;usename;locktype;mode;application_name;state;lock_satus;query | int | 锁信息 |	 SELECT * from pg_locks |ALL|
| 31 | greenplum_server_database_hit_cache_percent_rate | Gauge	| - | float | 缓存命中率 |	select sum(blks_hit)/(sum(blks_read)+sum(blks_hit))*100 from pg_stat_database; |ALL|
| 32 | greenplum_server_database_transition_commit_percent_rate | Gauge	| - | float | 事务提交率 |	select sum(xact_commit)/(sum(xact_commit)+sum(xact_rollback))*100 from pg_stat_database; |ALL|

### 4.声明：

- 本项目在 https://github.com/tangyibo/greenplum_exporter 基础上进行了GPDB多个版本的适配、部分修正，在此对原作者表示感谢
- 如果您在使用过程中有任何疑问，可以在cn.greenplum.org/askgp上提问并@阿福，咨询类问题也可以直接在Greenplum官方技术微信群中@阿福
- 如果本项目有帮助到您，麻烦您**star一下**～