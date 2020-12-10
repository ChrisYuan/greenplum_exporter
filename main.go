package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	logger "github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
	"greenplum-exporter/collector"
	"net/http"
)

var (
	listenAddress         = kingpin.Flag("web.listen-address", "web endpoint").Default("0.0.0.0:5433").String()
	metricPath            = kingpin.Flag("web.telemetry-path", "path to expose metrics.").Default("/metrics").String()
	disableDefaultMetrics = kingpin.Flag("disableDefaultMetrics", "do not report default metrics(go metrics and process metrics)").Default("true").Bool()
	greenplumVersion      = kingpin.Flag("greenplumVersion", "greenplum Server Version, options: gposs5-open " +
		"source greenplum 5.x, gposs6-open source greenplum 6.x, gpdb5-pivotal greenplum 5.x, gpdb6-pivotal greenplum 6.x").Default("gposs6").String()
)

var gposs6Scrapers = map[collector.Scraper]bool{
	collector.NewUsersScraper():         true,
	collector.NewSegmentScraper6():      true,
	collector.NewMaxConnScraper():       true,
	collector.NewLocksScraper6():        true,
	collector.NewDatabaseSizeScraper():  true,
	collector.NewConnDetailScraper6():   true,
	collector.NewConnectionsScraper6():  true,
	collector.NewClusterStateScraper6(): true,
	collector.NewBgWriterStateScraper6():true,
}

var gposs5Scrapers = map[collector.Scraper]bool{
	collector.NewUsersScraper():         true,
	collector.NewSegmentScraper5():      true,
	collector.NewMaxConnScraper():       true,
	collector.NewLocksScraper5():        true,
	collector.NewDatabaseSizeScraper():  true,
	collector.NewConnDetailScraper5():   true,
	collector.NewConnectionsScraper5():  true,
	collector.NewClusterStateScraper5(): true,
	collector.NewBgWriterStateScraper5():true,
}

var gpdb6Scrapers = map[collector.Scraper]bool{
	collector.NewUsersScraper():         true,
	collector.NewSystemScraper():        true,
	collector.NewSegmentScraper6():      true,
	collector.NewQueryScraper():         true,
	collector.NewMaxConnScraper():       true,
	collector.NewLocksScraper6():        true,
	collector.NewDynamicMemoryScraper(): true,
	collector.NewDiskScraper():          true,
	collector.NewDatabaseSizeScraper():  true,
	collector.NewConnDetailScraper6():   true,
	collector.NewConnectionsScraper6():  true,
	collector.NewClusterStateScraper6(): true,
	collector.NewBgWriterStateScraper6():true,
}

var gpdb5Scrapers = map[collector.Scraper]bool{
	collector.NewUsersScraper():         true,
	collector.NewSystemScraper():        true,
	collector.NewSegmentScraper5():      true,
	collector.NewQueryScraper():         true,
	collector.NewMaxConnScraper():       true,
	collector.NewLocksScraper5():        true,
	collector.NewDynamicMemoryScraper(): true,
	collector.NewDiskScraper():          true,
	collector.NewDatabaseSizeScraper():  true,
	collector.NewConnDetailScraper5():   true,
	collector.NewConnectionsScraper5():  true,
	collector.NewClusterStateScraper5(): true,
	collector.NewBgWriterStateScraper5():true,
}

var gathers prometheus.Gatherers

func main() {
	kingpin.Version("1.0.0")
	kingpin.HelpFlag.Short('h')
	logger.AddFlags(kingpin.CommandLine)
	kingpin.Parse()

	var metricsHandleFunc http.HandlerFunc
	if *greenplumVersion == "gposs6" {
		metricsHandleFunc = newHandler(*disableDefaultMetrics, gposs6Scrapers)
	} else if *greenplumVersion == "gposs5" {
		metricsHandleFunc = newHandler(*disableDefaultMetrics, gposs5Scrapers)
	} else if *greenplumVersion == "gpdb6" {
		metricsHandleFunc = newHandler(*disableDefaultMetrics, gpdb6Scrapers)
	} else if *greenplumVersion == "gpdb5" {
		metricsHandleFunc = newHandler(*disableDefaultMetrics, gpdb5Scrapers)
	} else {
		metricsHandleFunc = newHandler(*disableDefaultMetrics, gposs6Scrapers)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(*metricPath, metricsHandleFunc)

	logger.Warnf("Greenplum exporter started and will listening on : %s", *listenAddress)
	logger.Error(http.ListenAndServe(*listenAddress, mux).Error())
}

func newHandler(disableDefaultMetrics bool, scrapers map[collector.Scraper]bool) http.HandlerFunc {
	registry := prometheus.NewRegistry()
	enabledScrapers := make([]collector.Scraper, 0, 16)

	for scraper, enable := range scrapers {
		if enable {
			enabledScrapers = append(enabledScrapers, scraper)
		}
	}

	greenplumCollector := collector.NewCollector(enabledScrapers)
	registry.MustRegister(greenplumCollector)

	if disableDefaultMetrics {
		gathers = prometheus.Gatherers{registry}
	} else {
		gathers = prometheus.Gatherers{registry, prometheus.DefaultGatherer}
	}

	handler := promhttp.HandlerFor(gathers, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	})

	return handler.ServeHTTP
}
