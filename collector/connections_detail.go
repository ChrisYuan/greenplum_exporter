package collector

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	logger "github.com/prometheus/common/log"
)

/**
 *  连接详情抓取器
 */

const (
	//For GP5
	connectionsByUserSql5 = `select usename,
       							count(*) total,
       							count(*) filter(where current_query='<IDLE>') idle,
       							count(*) filter(where current_query<>'<IDLE>') active
								from pg_stat_activity group by 1;`
	connectionsByClientAddressSql5 = `select client_addr,
       									count(*) total,
       									count(*) filter(where current_query='<IDLE>') idle,
       									count(*) filter(where current_query<>'<IDLE>') active
										from pg_stat_activity where procpid <> pg_backend_pid() group by 1;`

	//For GP6
	connectionsByUserSql6 = `select usename,
                                      count(*) total,
                                      count(*) filter(where query='<IDLE>') idle,
                                      count(*) filter(where query<>'<IDLE>') active
                               from pg_stat_activity group by 1;`
	connectionsByClientAddressSql6 = `select client_addr,
                                               count(*) total,
                                               count(*) filter(where query='<IDLE>') idle,
                                               count(*) filter(where query<>'<IDLE>') active
                                        from pg_stat_activity where pid <> pg_backend_pid() group by 1;`
)

var (
	totalPerUserDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "total_connections_per_user"),
		"Total connections of specified database user",
		[]string{"usename"}, nil,
	)

	activePerUserDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "active_connections_per_user"),
		"Active connections of specified database user",
		[]string{"usename"}, nil,
	)

	idlePerUserDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "idle_connections_per_user"),
		"Idle connections of specified database user",
		[]string{"usename"}, nil,
	)

	totalPerClientDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "total_connections_per_client"),
		"Total connections of specified database user",
		[]string{"client"}, nil,
	)

	activePerClientDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "active_connections_per_client"),
		"Active connections of specified database user",
		[]string{"client"}, nil,
	)

	idlePerClientDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "idle_connections_per_client"),
		"Idle connections of specified database user",
		[]string{"client"}, nil,
	)

	totalCountClientDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "total_client_count"),
		"The total client count of greenplum database",
		nil, nil,
	)

	totalCountOnlineUsersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemCluster, "total_online_user_count"),
		"The total online user count of greenplum database",
		nil, nil,
	)
)

func NewConnDetailScraper6() Scraper {
	return connectionsDetailScraper6{}
}

func NewConnDetailScraper5() Scraper {
	return connectionsDetailScraper5{}
}

type connectionsDetailScraper6 struct{}

type connectionsDetailScraper5 struct{}

func (connectionsDetailScraper6) Name() string {
	return "connections_detail_scraper"
}

func (connectionsDetailScraper5) Name() string {
	return "connections_detail_scraper"
}

func (connectionsDetailScraper6) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	errU := scrapeLoadByUser6(db, ch)
	errC := scrapeLoadByClient6(db, ch)

	return combineErr(errC, errU)
}

func (connectionsDetailScraper5) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	errU := scrapeLoadByUser5(db, ch)
	errC := scrapeLoadByClient5(db, ch)

	return combineErr(errC, errU)
}

func scrapeLoadByUser6(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(connectionsByUserSql6)
	logger.Infof("Query Database: %s", connectionsByUserSql6)

	if err != nil {
		return err
	}

	defer rows.Close()

	errs := make([]error, 0)

	var totalOnlineUserCount int = 0
	for rows.Next() {
		var usename string
		var total, idle, active float64

		err = rows.Scan(&usename, &total, &idle, &active)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(totalPerUserDesc, prometheus.GaugeValue, total, usename)
		ch <- prometheus.MustNewConstMetric(idlePerUserDesc, prometheus.GaugeValue, idle, usename)
		ch <- prometheus.MustNewConstMetric(activePerUserDesc, prometheus.GaugeValue, active, usename)

		totalOnlineUserCount++
	}

	ch <- prometheus.MustNewConstMetric(totalCountOnlineUsersDesc, prometheus.GaugeValue, float64(totalOnlineUserCount))

	return combineErr(errs...)
}

func scrapeLoadByUser5(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(connectionsByUserSql5)
	logger.Infof("Query Database: %s", connectionsByUserSql5)

	if err != nil {
		return err
	}

	defer rows.Close()

	errs := make([]error, 0)

	var totalOnlineUserCount int = 0
	for rows.Next() {
		var usename string
		var total, idle, active float64

		err = rows.Scan(&usename, &total, &idle, &active)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(totalPerUserDesc, prometheus.GaugeValue, total, usename)
		ch <- prometheus.MustNewConstMetric(idlePerUserDesc, prometheus.GaugeValue, idle, usename)
		ch <- prometheus.MustNewConstMetric(activePerUserDesc, prometheus.GaugeValue, active, usename)

		totalOnlineUserCount++
	}

	ch <- prometheus.MustNewConstMetric(totalCountOnlineUsersDesc, prometheus.GaugeValue, float64(totalOnlineUserCount))

	return combineErr(errs...)
}

func scrapeLoadByClient6(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(connectionsByClientAddressSql6)

	if err != nil {
		return err
	}

	defer rows.Close()

	errs := make([]error, 0)

	var totalClientCount int = 0
	for rows.Next() {
		var client sql.NullString
		var total, idle, active float64

		err = rows.Scan(&client, &total, &idle, &active)

		if err != nil {
			errs = append(errs, err)
		}

		ch <- prometheus.MustNewConstMetric(totalPerClientDesc, prometheus.GaugeValue, total, client.String)
		ch <- prometheus.MustNewConstMetric(idlePerClientDesc, prometheus.GaugeValue, idle, client.String)
		ch <- prometheus.MustNewConstMetric(activePerClientDesc, prometheus.GaugeValue, active, client.String)

		totalClientCount++
	}

	ch <- prometheus.MustNewConstMetric(totalCountClientDesc, prometheus.GaugeValue, float64(totalClientCount))

	return combineErr(errs...)
}

func scrapeLoadByClient5(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(connectionsByClientAddressSql5)

	if err != nil {
		return err
	}

	defer rows.Close()

	errs := make([]error, 0)

	var totalClientCount int = 0
	for rows.Next() {
		var client sql.NullString
		var total, idle, active float64

		err = rows.Scan(&client, &total, &idle, &active)

		if err != nil {
			errs = append(errs, err)
		}

		ch <- prometheus.MustNewConstMetric(totalPerClientDesc, prometheus.GaugeValue, total, client.String)
		ch <- prometheus.MustNewConstMetric(idlePerClientDesc, prometheus.GaugeValue, idle, client.String)
		ch <- prometheus.MustNewConstMetric(activePerClientDesc, prometheus.GaugeValue, active, client.String)

		totalClientCount++
	}

	ch <- prometheus.MustNewConstMetric(totalCountClientDesc, prometheus.GaugeValue, float64(totalClientCount))

	return combineErr(errs...)
}