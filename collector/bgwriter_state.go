package collector

import (
	"database/sql"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	logger "github.com/prometheus/common/log"
	"time"
)

// 参考地址：
// （1） https://zhmin.github.io/2019/11/27/postgresql-bg-writer/
// （2） https://zhmin.github.io/2019/11/24/postgresql-checkpoint/
const (
	//For GPOSS5,旧的pg内核没有相关字段，所以下面使用当前时间代替
	statBgwriterSql5 = ` SELECT checkpoints_timed, checkpoints_req, 0 as checkpoint_write_time, 0 as checkpoint_sync_time, buffers_checkpoint
 			, buffers_clean, maxwritten_clean, buffers_backend, 0 as buffers_backend_fsync, buffers_alloc, now() as stats_reset FROM pg_stat_bgwriter`
	//For GPOSS6
	statBgwriterSql6 = ` SELECT checkpoints_timed, checkpoints_req, checkpoint_write_time, checkpoint_sync_time, buffers_checkpoint
 			, buffers_clean, maxwritten_clean, buffers_backend, buffers_backend_fsync, buffers_alloc, stats_reset FROM pg_stat_bgwriter`
)

var (
	checkpointsTimedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_checkpoints_timed_total"),
		"Number of scheduled checkpoints that have been performed",
		nil,
		nil,
	)

	checkpointsReqDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_checkpoints_req_total"),
		"Number of requested checkpoints that have been performed",
		nil,
		nil,
	)

	checkpointWriteTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_checkpoint_write_time_seconds_total"),
		"Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk",
		nil,
		nil,
	)

	checkpointSyncTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_checkpoint_sync_time_seconds_total"),
		"Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk",
		nil,
		nil,
	)

	buffersCheckpointDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_buffers_checkpoint_total"),
		"Number of buffers written during checkpoints",
		nil,
		nil,
	)

	buffersCleanDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_buffers_clean_total"),
		"Number of buffers written by the background writer",
		nil,
		nil,
	)

	maxWrittenCleanDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_maxwritten_clean_total"),
		"Number of times the background writer stopped a cleaning scan because it had written too many buffers",
		nil,
		nil,
	)

	buffersBackendDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_buffers_backend_total"),
		"Number of buffers written directly by a backend",
		nil,
		nil,
	)

	buffersBackendFsyncDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_buffers_backend_fsync_total"),
		"Number of times a backend had to execute its own fsync call",
		nil,
		nil,
	)

	buffersAllocDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_buffers_alloc_total"),
		"Number of buffers allocated",
		nil,
		nil,
	)

	statsResetDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subSystemServer, "bgwriter_stats_reset_timestamp"),
		"Time at which these statistics were last reset",
		nil,
		nil,
	)
)

func NewBgWriterStateScraper6() Scraper {
	return &bgWriterStateScraper6{}
}

func NewBgWriterStateScraper5() Scraper {
	return &bgWriterStateScraper5{}
}

type bgWriterStateScraper6 struct{}

type bgWriterStateScraper5 struct{}

func (bgWriterStateScraper6) Name() string {
	return "bg_writer_state_scraper"
}

func (bgWriterStateScraper5) Name() string {
	return "bg_writer_state_scraper"
}

func (bgWriterStateScraper6) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(statBgwriterSql6)
	logger.Infof("Query Database: %s", statBgwriterSql6)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 0, "", "")
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var checkpointsTimedCounter, checkpointsReqCounter,
		buffersCheckpoint, buffersClean, maxWrittenClean,
		buffersBackend, buffersBackendFsync, buffersAlloc int64
		var checkpointWriteTime, checkpointSyncTime float64
		var statsReset time.Time
		err = rows.Scan(&checkpointsTimedCounter,
			&checkpointsReqCounter,
			&checkpointWriteTime,
			&checkpointSyncTime,
			&buffersCheckpoint,
			&buffersClean,
			&maxWrittenClean,
			&buffersBackend,
			&buffersBackendFsync,
			&buffersAlloc,
			&statsReset)
		if err != nil {
			logger.Errorf("get metrics for scraper, error:%v", err.Error())
			return err
		}

		ch <- prometheus.MustNewConstMetric(checkpointsTimedDesc, prometheus.CounterValue, float64(checkpointsTimedCounter))
		ch <- prometheus.MustNewConstMetric(checkpointsReqDesc, prometheus.CounterValue, float64(checkpointsReqCounter))
		ch <- prometheus.MustNewConstMetric(checkpointWriteTimeDesc, prometheus.CounterValue, float64(checkpointWriteTime/1000))
		ch <- prometheus.MustNewConstMetric(checkpointSyncTimeDesc, prometheus.CounterValue, float64(checkpointSyncTime/1000))
		ch <- prometheus.MustNewConstMetric(buffersCheckpointDesc, prometheus.CounterValue, float64(buffersCheckpoint))
		ch <- prometheus.MustNewConstMetric(buffersCleanDesc, prometheus.CounterValue, float64(buffersClean))
		ch <- prometheus.MustNewConstMetric(maxWrittenCleanDesc, prometheus.CounterValue, float64(maxWrittenClean))
		ch <- prometheus.MustNewConstMetric(buffersBackendDesc, prometheus.CounterValue, float64(buffersBackend))
		ch <- prometheus.MustNewConstMetric(buffersBackendFsyncDesc, prometheus.CounterValue, float64(buffersBackendFsync))
		ch <- prometheus.MustNewConstMetric(buffersAllocDesc, prometheus.CounterValue, float64(buffersAlloc))
		ch <- prometheus.MustNewConstMetric(statsResetDesc, prometheus.GaugeValue, float64(statsReset.UTC().Unix()))

		return nil
	}

	return errors.New("bgwriter not found")
}

func (bgWriterStateScraper5) Scrape(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(statBgwriterSql5)
	logger.Infof("Query Database: %s", statBgwriterSql5)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, 0, "", "")
		logger.Errorf("get metrics for scraper, error:%v", err.Error())
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var checkpointsTimedCounter, checkpointsReqCounter,
		buffersCheckpoint, buffersClean, maxWrittenClean,
		buffersBackend, buffersBackendFsync, buffersAlloc int64
		var checkpointWriteTime, checkpointSyncTime float64
		var statsReset time.Time
		err = rows.Scan(&checkpointsTimedCounter,
			&checkpointsReqCounter,
			&checkpointWriteTime,
			&checkpointSyncTime,
			&buffersCheckpoint,
			&buffersClean,
			&maxWrittenClean,
			&buffersBackend,
			&buffersBackendFsync,
			&buffersAlloc,
			&statsReset)
		if err != nil {
			logger.Errorf("get metrics for scraper, error:%v", err.Error())
			return err
		}

		ch <- prometheus.MustNewConstMetric(checkpointsTimedDesc, prometheus.CounterValue, float64(checkpointsTimedCounter))
		ch <- prometheus.MustNewConstMetric(checkpointsReqDesc, prometheus.CounterValue, float64(checkpointsReqCounter))
		ch <- prometheus.MustNewConstMetric(checkpointWriteTimeDesc, prometheus.CounterValue, float64(checkpointWriteTime/1000))
		ch <- prometheus.MustNewConstMetric(checkpointSyncTimeDesc, prometheus.CounterValue, float64(checkpointSyncTime/1000))
		ch <- prometheus.MustNewConstMetric(buffersCheckpointDesc, prometheus.CounterValue, float64(buffersCheckpoint))
		ch <- prometheus.MustNewConstMetric(buffersCleanDesc, prometheus.CounterValue, float64(buffersClean))
		ch <- prometheus.MustNewConstMetric(maxWrittenCleanDesc, prometheus.CounterValue, float64(maxWrittenClean))
		ch <- prometheus.MustNewConstMetric(buffersBackendDesc, prometheus.CounterValue, float64(buffersBackend))
		ch <- prometheus.MustNewConstMetric(buffersBackendFsyncDesc, prometheus.CounterValue, float64(buffersBackendFsync))
		ch <- prometheus.MustNewConstMetric(buffersAllocDesc, prometheus.CounterValue, float64(buffersAlloc))
		ch <- prometheus.MustNewConstMetric(statsResetDesc, prometheus.GaugeValue, float64(statsReset.UTC().Unix()))

		return nil
	}

	return errors.New("bgwriter not found")
}