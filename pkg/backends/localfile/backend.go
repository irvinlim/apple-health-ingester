package localfile

import (
	"os"
	"path"
	"sort"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

var (
	metricsPath string
)

// Backend LocalFile is used to store ingested metrics in the local filesystem
// as JSON files. It is not very performant as it would process all data at once
// to produce a sorted JSON output file. As such, it should only be used for
// debugging purposes.
//
// TODO(irvinlim): Handle workout data
type Backend struct {
	metrics map[string]*MetricFile
	mtx     sync.RWMutex
}

var _ backends.Backend = &Backend{}

func NewBackend() (*Backend, error) {
	backend := &Backend{}

	// Load metrics
	if metricsPath == "" {
		return nil, errors.New("--localfile.metricsPath is not set")
	}
	metrics, err := backend.loadMetrics()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load metrics from %v", metricsPath)
	}
	backend.metrics = metrics

	return backend, nil
}

func (b *Backend) Name() string {
	return "LocalFile"
}

// Write will take the incoming payload and merge the metrics with existing
// metric data, before writing it back to the filesystem.
func (b *Backend) Write(payload *healthautoexport.Payload, target string) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// Handle metrics.
	for _, metric := range payload.Data.Metrics {
		if err := b.handleMetric(metric, target); err != nil {
			return errors.Wrapf(err, "handle metric error for %v", metric.Name)
		}
	}

	return nil
}

func (b *Backend) handleMetric(metric *healthautoexport.Metric, target string) error {
	var metricFile MetricFile
	metricFile.FromMetric(metric, target)
	fileName := metricFile.GetFileName()
	updatedData := metricFile.Data

	// Merge with existing data if present
	existing, ok := b.metrics[fileName]
	if ok {
		// Merge data points by timestamp
		dataByTimestamp := make(map[healthautoexport.Time]*healthautoexport.Datapoint, len(existing.Data))
		for _, datum := range existing.Data {
			dataByTimestamp[*datum.Date] = datum
		}
		for _, datum := range metricFile.Data {
			dataByTimestamp[*datum.Date] = datum
		}

		// Convert back to slice and sort
		newData := make([]*healthautoexport.Datapoint, 0, len(dataByTimestamp))
		for _, datapoint := range dataByTimestamp {
			newData = append(newData, datapoint)
		}
		sort.Slice(newData, func(i, j int) bool {
			return newData[i].Date.Before(newData[j].Date.Time)
		})

		// Store merged data
		updatedData = newData
	}

	// Update data
	metricFile.Data = updatedData

	// Write back
	metricFilePath := path.Join(metricsPath, fileName)
	if err := b.writeMetricFile(metricFilePath, &metricFile); err != nil {
		return errors.Wrapf(err, "cannot write metrics to %v", metricFilePath)
	}

	return nil
}

func (b *Backend) loadMetrics() (map[string]*MetricFile, error) {
	output := make(map[string]*MetricFile)
	files, err := os.ReadDir(metricsPath)
	if err != nil {
		// Directory doesn't exist, simply return empty map.
		if os.IsNotExist(err) {
			return output, nil
		}

		return nil, errors.Wrapf(err, "cannot read dir")
	}

	for _, file := range files {
		metricFilePath := path.Join(metricsPath, file.Name())
		metricFile, err := b.loadMetricFile(metricFilePath)
		if err != nil {
			log.WithError(err).Warnf("could not read %v as metric file", metricFilePath)
			continue
		}
		output[metricFile.GetFileName()] = metricFile
	}

	return output, nil
}

func (b *Backend) loadMetricFile(name string) (*MetricFile, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open %v", name)
	}
	defer func() {
		_ = file.Close()
	}()

	var metricFile MetricFile
	dec := jsoniter.NewDecoder(file)
	if err := dec.Decode(&metricFile); err != nil {
		return nil, err
	}

	return &metricFile, nil
}

func (b *Backend) writeMetricFile(name string, metricFile *MetricFile) error {
	// Ensure directories exist
	dirname := path.Dir(name)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return errors.Wrapf(err, "cannot makedirs for %v", dirname)
	}

	// Write file
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "cannot open %v", name)
	}
	defer func() {
		_ = file.Close()
	}()

	// Encode as JSON
	enc := jsoniter.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(metricFile)
}

func init() {
	pflag.StringVar(&metricsPath, "localfile.metricsPath", "",
		"Output path to write metrics, with one metric per file. All data will be aggregated by timestamp. "+
			"Any existing data will be merged together.")
}
