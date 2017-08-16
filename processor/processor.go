/*
http://www.apache.org/licenses/LICENSE-2.0.txt

Copyright 2016 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package processor

import (
	"fmt"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
)

const (
	//Name of the plugin
	Name = "regexp-o-matic"
	//Version of the plugin
	Version = 1

	configSplitRegexp  = "split_on"
	configParseRegexp = "regexps"
	configShouldEmit = "should_emit"
	configAddTags = "tags"

	// When to emit potentially-modified metrics
	shouldEmitAlways = "always" // no matter what
	shouldEmitOnAllSuccess = "all_success" // only if all match
	shouldEmitOnAnySuccess = "any_success" // only if at least one matches
)

type Plugin struct {
}

// New() returns a new instance of the plugin
func New() *Plugin {
	p := &Plugin{}
	return p
}

// GetConfigPolicy returns the config policy
func (p *Plugin) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	policy.AddNewStringRule([]string{""}, configShouldEmit, false, plugin.SetDefaultString(shouldEmitAlways))
	return *policy, nil
}

// Process processes the data
func (p *Plugin) Process(metrics []plugin.Metric, cfg plugin.Config) ([]plugin.Metric, error) {
	// Configuration
	var splitRegexes, parseRegexes []*regexp.Regexp
	splitRegexesRaw, ok := cfg[configSplitRegexp].([]string)
	if ok {
		splitRegexes, err := compileRegexes(splitRegexesRaw)
	}

	parseRegexesRaw, ok := cfg[configParseRegexp].([]string)
	if !ok {
		return nil, fmt.Errorf("Must specify parse regexps at least")
	}
	parseRegexes = compileRegexes(parseRegexesRaw)

	newMetrics := make([]plugin.Metric, 0)

	if splitRegexes != nil {
		var newMetrics []plugin.Metric
		for _, m := range metrics {
			newMetrics = append(newMetrics, ...splitMetric(m, splitRegexes))
		}
		metrics = newMetrics
	}

	for _, m := range metrics {
		logBlock, ok := m.Data.(string)
		if !ok {
			warnFields := map[string]interface{}{
				"namespace": m.Namespace.Strings(),
				"data":      m.Data,
			}
			log.WithFields(warnFields).Warn("unexpected data type, plugin processes only strings")
			continue
		}
		subBlocks, err := parse(logBlock, splitRgx)
		if err != nil {
			warnFields := map[string]interface{}{
				"namespace":       m.Namespace.Strings(),
				"data":            m.Data,
				configSplitRegexp: splitRgx,
			}
			log.WithFields(warnFields).Warn(err)
			continue
		}

		for _, block := range subBlocks {
			newMetric := m
			newMetric.Data = block
			newMetrics = append(newMetrics, newMetric)
		}
	}
	return newMetrics, nil
}

func getCheckConfigVar(cfg plugin.Config, cfgVarName string) (*regexp.Regexp, error) {
	expr, err := cfg.GetString(cfgVarName)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", cfgVarName, err)
	}
	rgx, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", cfgVarName, err)
	}
	return rgx, nil

}
func getCheckConfig(cfg plugin.Config) (*regexp.Regexp, error) {
	splitRgx, err := getCheckConfigVar(cfg, configSplitRegexp)
	if err != nil {
		return nil, err
	}

	return splitRgx, nil
}

func parse(message string, rgx *regexp.Regexp) ([]string, error) {
	subBlocks := rgx.Split(message, -1)
	return subBlocks, nil
}

func compileRegexes(from []string) ([]*regexp.Regexp, error) {
	regexes = []*regexp.Regexp
	for _, expr := range from {
		regex, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, regex)
	}
	return regexes, nil
}

func splitMetric(metric plugin.Metric, regexes []*regexp.Regexp) ([]plugin.Metric, error) {
	var metrics []plugin.Metric
	var workspace, product []string
	// Initialize the workspace
	origString, ok := metric.Data.(string)
	if !ok {
		return nil, fmt.Errorf("Metric to be split was not a string")
	}

	workspace = append(workspace, origString)

	// Work through it
	for _, regex := range regexes {
		for _, current := range workspace {
			splits := regex.Split(current, -1)
			product = append(product, ...splits)
		}
		workspace = product
		product = make([]string, 0)
	}

	// Finally, copy the metric over each split
	metrics = make([]plugin.Metric, len(workspace))
	for idx, split := range workspace {
		metrics[idx] = plugin.Metric{
			Namespace: metric.Namespace,
			Version: metric.Version,
			Config: metric.Config,
			Data: workspace[idx],
			Tags: metric.Tags,
			Timestamp: metric.Timestamp,
			Unit: metric.Unit,
			Description: metric.Description,
		}
	}

	// And return it
	return metrics, nil
}

