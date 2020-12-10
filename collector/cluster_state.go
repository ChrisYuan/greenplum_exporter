package collector

import (
	"database/sql"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	logger "github.com/prometheus/common/log"
	"time"
)

/**
 *  集群状态抓取器
 */

const (
	checkStateSql     = `SELECT count(1) from gp_dist_random('gp_id')`
	versionSql        = `select (select regexp_matches((select (select regexp_matches((select version()), 'Greenplum Database \d{1,}\.\d{1,}\.\d{1,}'))[1] as version), '\d{1,}\.\d{1,}\.\d{1,}'))[1];`
	masterNameSql     = `SELECT hostname from gp_segment_configuration where content=-1 and role='p'`
	standbyNameSql    = `SELECT hostname from gp_segment_configuration where content=-1 and role='m'`
	upTimeSql         = `select extract(epoch from now() - pg_postmaster_start_time())`
	syncSql           = `SELECT count(*) from pg_stat_replication where state='streaming'`
	//For GP5，旧的pg内核没有pg_conf_load_time函数，所以下面使用当前时间代替
	configLoadTimeSql5 = `SELECT now() `
	//For GP6
	configLoadTimeSql6 = `SELECT pg_conf_load_time() `
)

var (
	stateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "state"),
		"Whether the GreenPlum database is accessible",
		[]string{"version", "master", "standby"},
		nil,
	)

	upTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "uptime"),
		"Duration that the GreenPlum database have been started since last up in second",
		nil, nil,
	)

	syncDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "sync"),
		"Whether the GreenPlum master node is synchronizing to standby",
		nil,
		nil,
	)

	configLoadTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "config_last_load_time_seconds"),
		"Timestamp of the last configuration reload",
		nil,
		nil,
	)
)

func NewClusterStateScraper6() Scraper {
	return &clusterStateScraper6{}
}

func NewClusterStateScraper5() Scraper {
	return &clusterStateScraper5{}
}

type clusterStateScraper6 struct{}

type clusterStateScraper5 struct{}


func (clusterStateScraper6) Name() string {
	return "cluster_state_scraper"
}

func (clusterStateScraper5) Name() string {
	return "cluster_state_scraper"
}

func (clusterStateScraper6) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(checkStateSql)
	logger.Infof("Query Database: %s", checkStateSql)

	if err != nil {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 0, "", "")
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var count int
		err = rows.Scan(&count)
		if err != nil {
			ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 0, "", "")
			logger.Errorf("get metrics for scraper, error:%v", err.Error())
			return err
		}
	}

	version, errV := scrapeVersion(db)
	master, errM := scrapeMaster(db)
	standby, errX := scrapeStandby(db)
	upTime, errU := scrapeUpTime(db)
	sync, errW := scrapeSync(db)
	configLoadTime, errY := scrapeConfigLoadTime6(db)

	ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 1, version, master, standby)
	ch <- prometheus.MustNewConstMetric(upTimeDesc, prometheus.GaugeValue, upTime)
	ch <- prometheus.MustNewConstMetric(syncDesc, prometheus.GaugeValue, sync)
	ch <- prometheus.MustNewConstMetric(configLoadTimeDesc, prometheus.GaugeValue, float64(configLoadTime.UTC().Unix()))

	return combineErr(errM, errV, errU, errW, errX, errY)
}

func (clusterStateScraper5) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(checkStateSql)
	logger.Infof("Query Database: %s", checkStateSql)

	if err != nil {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 0, "", "")
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var count int
		err = rows.Scan(&count)
		if err != nil {
			ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 0, "", "")
			logger.Errorf("get metrics for scraper, error:%v", err.Error())
			return err
		}
	}

	version, errV := scrapeVersion(db)
	master, errM := scrapeMaster(db)
	standby, errX := scrapeStandby(db)
	upTime, errU := scrapeUpTime(db)
	sync, errW := scrapeSync(db)
	configLoadTime, errY := scrapeConfigLoadTime5(db)

	ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 1, version, master, standby)
	ch <- prometheus.MustNewConstMetric(upTimeDesc, prometheus.GaugeValue, upTime)
	ch <- prometheus.MustNewConstMetric(syncDesc, prometheus.GaugeValue, sync)
	ch <- prometheus.MustNewConstMetric(configLoadTimeDesc, prometheus.GaugeValue, float64(configLoadTime.UTC().Unix()))

	return combineErr(errM, errV, errU, errW, errX, errY)
}

func scrapeUpTime(db *sql.DB) (upTime float64, err error) {
	rows, err := db.Query(upTimeSql)
	logger.Infof("Query Database Up Time: %s", upTimeSql)

	if err != nil {
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&upTime)
		return
	}

	err = errors.New("start time of greenPlum not found")

	return
}

func scrapeVersion(db *sql.DB) (ver string, err error) {
	rows, err := db.Query(versionSql)
	logger.Infof("Query Database Version: %s", versionSql)

	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&ver)
		return
	}

	err = errors.New("greenPlum version not found")
	return
}

func scrapeMaster(db *sql.DB) (host string, err error) {
	rows, err := db.Query(masterNameSql)
	logger.Infof("Query Database Master Name: %s", masterNameSql)

	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&host)
		return
	}

	err = errors.New("hostname for master node not found")

	return
}

func scrapeStandby(db *sql.DB) (host string, err error) {
	rows, err := db.Query(standbyNameSql)
	logger.Infof("Query Database Standby Name: %s", standbyNameSql)

	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&host)
		return
	}

	return
}

func scrapeSync(db *sql.DB) (sync float64, err error) {
	rows, err := db.Query(syncSql)
	logger.Infof("Query Database Sync : %s", syncSql)

	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&sync)
		return
	}

	err = errors.New("greenPlum sync status not found")
	return
}

func scrapeConfigLoadTime6(db *sql.DB) (time time.Time, err error) {
	rows, err := db.Query(configLoadTimeSql6)
	logger.Infof("Query Database Config load Time : %s", configLoadTimeSql6)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&time)
		return
	}
	err = errors.New("greenPlum Config last load time not found")
	return
}

func scrapeConfigLoadTime5(db *sql.DB) (time time.Time, err error) {
	rows, err := db.Query(configLoadTimeSql5)
	logger.Infof("Query Database Config load Time : %s", configLoadTimeSql5)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&time)
		return
	}
	err = errors.New("greenPlum Config last load time not found")
	return
}