package collector

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	logger "github.com/prometheus/common/log"
	"time"
)

/**
 * 数据库锁信息抓取器
 */

const (
	//For GP5
	locksQuerySql5 = `
		SELECT pg_locks.pid
     , pg_database.datname
     , pg_stat_activity.usename
     , locktype
     , mode
     , pg_stat_activity.application_name
     , case
         when (current_query<>'<IDLE>') then 'idle'
         when (current_query<>'<IDLE>' and not waiting) then 'running'
         when (current_query<>'<IDLE>' and waiting) then 'waiting'
         else 'active'
       end state
     , CASE
           WHEN granted='f' THEN
               'wait_lock'
           WHEN granted='t' THEN
               'get_lock'
    END lock_satus
     , pg_stat_activity.current_query
     , coalesce(least(query_start,xact_start),'1970-01-01 00:00:00') start_time
     , count(*)::float
FROM pg_locks
         JOIN pg_database ON pg_locks.database=pg_database.oid
         JOIN pg_stat_activity on pg_locks.pid=pg_stat_activity.procpid
WHERE NOT pg_locks.pid=pg_backend_pid()
  AND pg_stat_activity.application_name<>'pg_statsinfod'
GROUP BY pg_locks.pid, pg_database.datname,pg_stat_activity.usename, locktype, mode,
         pg_stat_activity.application_name, state , lock_satus ,pg_stat_activity.current_query, start_time
ORDER BY start_time;
		`

	//For GP6
	locksQuerySql6 = `
		SELECT pg_locks.pid
			 , pg_database.datname
			 , pg_stat_activity.usename
			 , locktype
			 , mode
			 , pg_stat_activity.application_name
			 , state
			 , CASE
						WHEN granted='f' THEN
							'wait_lock'
						WHEN granted='t' THEN
							'get_lock'
					END lock_satus
			 , pg_stat_activity.query
     		 , coalesce(least(query_start,xact_start),'1970-01-01 00:00:00') start_time
			 , count(*)::float
		  FROM pg_locks
		  JOIN pg_database ON pg_locks.database=pg_database.oid
		  JOIN pg_stat_activity on pg_locks.pid=pg_stat_activity.pid
		WHERE NOT pg_locks.pid=pg_backend_pid()
		AND pg_stat_activity.application_name<>'pg_statsinfod'
		GROUP BY pg_locks.pid, pg_database.datname,pg_stat_activity.usename, locktype, mode,
		pg_stat_activity.application_name, state , lock_satus ,pg_stat_activity.query, start_time
		ORDER BY start_time
		`
)

var (
	locksDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "locks_table_detail"),
		"Table locks detail for greenplum database",
		[]string{"pid", "datname", "usename", "locktype", "mode", "application_name", "state", "lock_satus", "query"},
		nil,
	)
)

func NewLocksScraper6() Scraper {
	return &locksScraper6{}
}

func NewLocksScraper5() Scraper {
	return &locksScraper5{}
}

type locksScraper6 struct{}

type locksScraper5 struct{}

func (locksScraper6) Name() string {
	return "locks_scraper"
}

func (locksScraper5) Name() string {
	return "locks_scraper"
}

func (locksScraper6) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(locksQuerySql6)
	logger.Infof("Query Database: %s", locksQuerySql6)
	if err != nil {
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pid, datname, usename, locktype, mode, application_name, state, lock_satus, query string
		var startTime time.Time
		var count int64
		err = rows.Scan(&pid,
			&datname,
			&usename,
			&locktype,
			&mode,
			&application_name,
			&state,
			&lock_satus,
			&query,
			&startTime,
			&count)
		if err != nil {
			logger.Errorf("get metrics for scraper, error:%v", err.Error())
			return err
		}
		ch <- prometheus.MustNewConstMetric(locksDesc, prometheus.GaugeValue, float64(startTime.UTC().Unix()), pid, datname, usename, locktype, mode, application_name, state, lock_satus, query)
	}

	return nil
}

func (locksScraper5) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(locksQuerySql5)
	logger.Infof("Query Database: %s", locksQuerySql5)
	if err != nil {
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pid, datname, usename, locktype, mode, application_name, state, lock_satus, query string
		var startTime time.Time
		var count int64
		err = rows.Scan(&pid,
			&datname,
			&usename,
			&locktype,
			&mode,
			&application_name,
			&state,
			&lock_satus,
			&query,
			&startTime,
			&count)
		if err != nil {
			logger.Errorf("get metrics for scraper, error:%v", err.Error())
			return err
		}
		ch <- prometheus.MustNewConstMetric(locksDesc, prometheus.GaugeValue, float64(startTime.UTC().Unix()), pid, datname, usename, locktype, mode, application_name, state, lock_satus, query)
	}
	return nil
}