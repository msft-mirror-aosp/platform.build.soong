// Copyright 2020 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"io/ioutil"
	"runtime"
	"sort"

	"github.com/google/blueprint/metrics"
	"google.golang.org/protobuf/proto"

	soong_metrics_proto "android/soong/ui/metrics/metrics_proto"
)

var soongMetricsOnceKey = NewOnceKey("soong metrics")

type SoongMetrics struct {
	Modules  int
	Variants int
}

func readSoongMetrics(config Config) (SoongMetrics, bool) {
	soongMetrics, ok := config.Peek(soongMetricsOnceKey)
	if ok {
		return soongMetrics.(SoongMetrics), true
	} else {
		return SoongMetrics{}, false
	}
}

func init() {
	RegisterSingletonType("soong_metrics", soongMetricsSingletonFactory)
}

func soongMetricsSingletonFactory() Singleton { return soongMetricsSingleton{} }

type soongMetricsSingleton struct{}

func (soongMetricsSingleton) GenerateBuildActions(ctx SingletonContext) {
	metrics := SoongMetrics{}
	ctx.VisitAllModules(func(m Module) {
		if ctx.PrimaryModule(m) == m {
			metrics.Modules++
		}
		metrics.Variants++
	})
	ctx.Config().Once(soongMetricsOnceKey, func() interface{} {
		return metrics
	})
}

func collectMetrics(config Config, eventHandler *metrics.EventHandler) *soong_metrics_proto.SoongBuildMetrics {
	metrics := &soong_metrics_proto.SoongBuildMetrics{}

	soongMetrics, ok := readSoongMetrics(config)
	if ok {
		metrics.Modules = proto.Uint32(uint32(soongMetrics.Modules))
		metrics.Variants = proto.Uint32(uint32(soongMetrics.Variants))
	}

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	metrics.MaxHeapSize = proto.Uint64(memStats.HeapSys)
	metrics.TotalAllocCount = proto.Uint64(memStats.Mallocs)
	metrics.TotalAllocSize = proto.Uint64(memStats.TotalAlloc)

	for _, event := range eventHandler.CompletedEvents() {
		perfInfo := soong_metrics_proto.PerfInfo{
			Description: proto.String(event.Id),
			Name:        proto.String("soong_build"),
			StartTime:   proto.Uint64(uint64(event.Start.UnixNano())),
			RealTime:    proto.Uint64(event.RuntimeNanoseconds()),
		}
		metrics.Events = append(metrics.Events, &perfInfo)
	}
	mixedBuildsInfo := soong_metrics_proto.MixedBuildsInfo{}
	mixedBuildEnabledModules := make([]string, 0, len(config.mixedBuildEnabledModules))
	for module, _ := range config.mixedBuildEnabledModules {
		mixedBuildEnabledModules = append(mixedBuildEnabledModules, module)
	}

	mixedBuildDisabledModules := make([]string, 0, len(config.mixedBuildDisabledModules))
	for module, _ := range config.mixedBuildDisabledModules {
		mixedBuildDisabledModules = append(mixedBuildDisabledModules, module)
	}
	// Sorted for deterministic output.
	sort.Strings(mixedBuildEnabledModules)
	sort.Strings(mixedBuildDisabledModules)

	mixedBuildsInfo.MixedBuildEnabledModules = mixedBuildEnabledModules
	mixedBuildsInfo.MixedBuildDisabledModules = mixedBuildDisabledModules
	metrics.MixedBuildsInfo = &mixedBuildsInfo

	return metrics
}

func WriteMetrics(config Config, eventHandler *metrics.EventHandler, metricsFile string) error {
	metrics := collectMetrics(config, eventHandler)

	buf, err := proto.Marshal(metrics)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(absolutePath(metricsFile), buf, 0666)
	if err != nil {
		return err
	}

	return nil
}
