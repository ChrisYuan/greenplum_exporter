package collector

import (
	"database/sql"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	logger "github.com/prometheus/common/log"
)

/**
 *  连接数量抓取器
 */

const (
	//For GP5
	connectionsSql5 = `select
    					count(*) total,
    					count(*) filter(where current_query='<IDLE>') idle,
    					count(*) filter(where current_query<>'<IDLE>') active,
    					count(*) filter(where current_query<>'<IDLE>' and not waiting) running,
    					count(*) filter(where current_query<>'<IDLE>' and waiting) waiting
						from pg_stat_activity where procpid <> pg_backend_pid();`

	//For GP6
	connectionsSql6 = `select
                         count(*) total, 
                         count(*) filter(where query='<IDLE>') idle, 
                         count(*) filter(where query<>'<IDLE>') active,
                         count(*) filter(where query<>'<IDLE>' and not waiting) running,
                         count(*) filter(where query<>'<IDLE>' and waiting) waiting
                         from pg_stat_activity where pid <> pg_backend_pid();`
)

var (
	currentConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "total_connections"),
		"Current connections of GreenPlum cluster at scrape time",
		nil, nil,
	)

	idleConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "idle_connections"),
		"Idle connections of GreenPlum cluster at scape time",
		nil, nil,
	)

	activeConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "active_connections"),
		"Active connections of GreenPlum cluster at scape time",
		nil, nil,
	)

	runningConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "running_connections"),
		"Running sql count of GreenPlum cluster at scape time",
		nil, nil,
	)

	queuingConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "waiting_connections"),
		"Waiting sql count of GreenPlum cluster at scape time",
		nil, nil,
	)
)

func NewConnectionsScraper6() Scraper {
	return &connectionsScraper6{}
}

func NewConnectionsScraper5() Scraper {
	return &connectionsScraper5{}
}

type connectionsScraper6 struct{}

type connectionsScraper5 struct{}

func (connectionsScraper6) Name() string {
	return "connections_scraper"
}

func (connectionsScraper5) Name() string {
	return "connections_scraper"
}

func (connectionsScraper6) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(connectionsSql6)
	logger.Infof("Query Database: %s",connectionsSql6)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var total, idle, active, running, waiting float64
		err = rows.Scan(&total, &idle, &active, &running, &waiting)
		if err != nil {
			return err
		}
		ch <- prometheus.MustNewConstMetric(currentConnDesc, prometheus.GaugeValue, total)
		ch <- prometheus.MustNewConstMetric(idleConnDesc, prometheus.GaugeValue, idle)
		ch <- prometheus.MustNewConstMetric(activeConnDesc, prometheus.GaugeValue, active)
		ch <- prometheus.MustNewConstMetric(runningConnDesc, prometheus.GaugeValue, running)
		ch <- prometheus.MustNewConstMetric(queuingConnDesc, prometheus.GaugeValue, waiting)
		return nil
	}

	return errors.New("connections not found")
}

func (connectionsScraper5) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(connectionsSql5)
	logger.Infof("Query Database: %s",connectionsSql5)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var total, idle, active, running, waiting float64
		err = rows.Scan(&total, &idle, &active, &running, &waiting)
		if err != nil {
			return err
		}
		ch <- prometheus.MustNewConstMetric(currentConnDesc, prometheus.GaugeValue, total)
		ch <- prometheus.MustNewConstMetric(idleConnDesc, prometheus.GaugeValue, idle)
		ch <- prometheus.MustNewConstMetric(activeConnDesc, prometheus.GaugeValue, active)
		ch <- prometheus.MustNewConstMetric(runningConnDesc, prometheus.GaugeValue, running)
		ch <- prometheus.MustNewConstMetric(queuingConnDesc, prometheus.GaugeValue, waiting)
		return nil
	}

	return errors.New("connections not found")
}