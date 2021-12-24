package localfile

import (
	"strings"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

type MetricFile struct {
	Name   string                        `json:"name"`
	Target string                        `json:"target,omitempty"`
	Units  healthautoexport.Units        `json:"units"`
	Data   []*healthautoexport.Datapoint `json:"data"`
}

func (f MetricFile) GetFileName() string {
	filename := f.Name + "_" + string(f.Units)
	filename = strings.ReplaceAll(filename, "/", "_")
	if f.Target != "" {
		filename = f.Target + "_" + filename
	}
	return filename + ".json"
}

func (f *MetricFile) FromMetric(metric *healthautoexport.Metric, target string) {
	f.Name = metric.Name
	f.Units = metric.Units
	f.Data = metric.Data
	f.Target = target
}
