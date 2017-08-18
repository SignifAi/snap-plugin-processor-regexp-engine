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
	"bytes"
	"fmt"
	"regexp"
	"text/template"

	log "github.com/Sirupsen/logrus"
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
)

const (
	//Name of the plugin
	Name = "regexp-o-matic"
	//Version of the plugin
	Version = 1

	configSplitRegexp = "split_on"
	configParseRegexp = "regexps"
	configShouldEmit  = "should_emit"
	configAddTags     = "tags"

	// When to emit potentially-modified metrics
	shouldEmitAlways       = "always"      // no matter what
	shouldEmitOnAllSuccess = "all_success" // only if all match
	shouldEmitOnAnySuccess = "any_success" // only if at least one matches
	shouldEmitOnNoSuccess  = "no_success"  // only if _none_ match (grep -v mode)
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
	var err error
	var shouldEmit string
	var tagsTemplates *template.Template
	splitRegexesRaw, ok := cfg[configSplitRegexp].([]string)
	if ok {
		splitRegexes, err = compileRegexes(splitRegexesRaw)
		if err != nil {
			return nil, fmt.Errorf("Couldn't compile split regexes: %v", splitRegexesRaw)
		}
	}

	parseRegexesRaw, ok := cfg[configParseRegexp].([]string)
	if !ok {
		return nil, fmt.Errorf("Must specify parse regexps at least")
	}
	parseRegexes, err = compileRegexes(parseRegexesRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compile a regex: %v", err)
	}

	tagTemplatesRaw, ok := cfg[configAddTags].(map[string]string)
	if ok {
		tagsTemplates, err = compileTemplates(tagTemplatesRaw)
	}

	newMetrics := make([]plugin.Metric, 0)
	shouldEmit, ok = cfg[configShouldEmit].(string)
	if !ok {
		// Default is always
		shouldEmit = shouldEmitAlways
	}

	if shouldEmit != shouldEmitAlways && shouldEmit != shouldEmitOnAnySuccess && shouldEmit != shouldEmitOnAllSuccess {
		return nil, fmt.Errorf("%v should be one of '%v', '%v' or '%v'", configShouldEmit, shouldEmitAlways, shouldEmitOnAnySuccess, shouldEmitOnAllSuccess)
	}

	if splitRegexes != nil {
		for _, m := range metrics {
			splitMetrics, err := splitMetric(m, splitRegexes)
			if err == nil {
				parsedMetrics, err := processMetrics(splitMetrics, parseRegexes, shouldEmit, tagsTemplates)
				if err == nil {
					newMetrics = append(newMetrics, parsedMetrics...)
				} else {
					return nil, err
				}
			}
		}
	} else {
		newMetrics, err = processMetrics(metrics, parseRegexes, shouldEmit, tagsTemplates)
		if err != nil {
			return nil, err
		}
	}

	return newMetrics, nil
}

func parse(message string, regexes []*regexp.Regexp) (map[string]string, int, error) {
	var fields map[string]string
	var matchCnt int = 0
	for _, regex := range regexes {
		match := regex.FindStringSubmatch(message)
		if match != nil {
			matchCnt++
		}
		for i, name := range regex.SubexpNames() {
			if i > 0 && i <= len(match) {
				if fields == nil {
					fields = make(map[string]string, 0)
				}
				fields[name] = match[i]
			}
		}
	}
	return fields, matchCnt, nil
}

func compileRegexes(from []string) ([]*regexp.Regexp, error) {
	var regexes []*regexp.Regexp
	for _, expr := range from {
		regex, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, regex)
	}
	return regexes, nil
}

func compileTemplates(templates map[string]string) (*template.Template, error) {
	rootTemplate := template.New("")
	for tag, tagTemplate := range templates {
		if tag == "" {
			// nope
			continue
		}
		newTagTemplate := rootTemplate.New(tag)
		if newTagTemplate == nil {
			return nil, fmt.Errorf("Couldn't create template for tag %v", tag)
		}
		newTagTemplate, err := newTagTemplate.Parse(tagTemplate)
		if err != nil {
			return nil, err
		}
	}
	return rootTemplate, nil
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
			product = append(product, splits...)
		}
		workspace = product
		product = make([]string, 0)
	}

	// Finally, copy the metric over each split
	metrics = make([]plugin.Metric, len(workspace))
	for idx, split := range workspace {
		metrics[idx] = plugin.Metric{
			Namespace:   metric.Namespace,
			Version:     metric.Version,
			Config:      metric.Config,
			Data:        split,
			Tags:        metric.Tags,
			Timestamp:   metric.Timestamp,
			Unit:        metric.Unit,
			Description: metric.Description,
		}
	}

	// And return it
	return metrics, nil
}

func processMetrics(metrics []plugin.Metric, regexps []*regexp.Regexp, shouldEmit string, tagsTemplates *template.Template) ([]plugin.Metric, error) {
	var newMetrics []plugin.Metric
	for _, n := range metrics {
		logBlock, ok := n.Data.(string)
		if !ok {
			warnFields := map[string]interface{}{
				"namespace": n.Namespace.Strings(),
				"data":      n.Data,
			}
			log.WithFields(warnFields).Warn("unexpected data type, plugin processes only strings")
			continue
		}
		newTags, matchCnt, err := parse(logBlock, regexps)
		if err != nil {
			warnFields := map[string]interface{}{
				"namespace":       n.Namespace.Strings(),
				"data":            n.Data,
				configParseRegexp: regexps,
			}
			log.WithFields(warnFields).Warn(err)
			continue
		}

		if (shouldEmit == shouldEmitOnAnySuccess && matchCnt == 0) || (shouldEmit == shouldEmitOnAllSuccess && matchCnt < len(regexps) || (shouldEmit == shouldEmitOnNoSuccess && matchCnt > 0)) {
			continue
		}

		if newTags != nil || tagsTemplates != nil {
			// Because we've split the metric,
			// there's a chance we're using the
			// same tags pointer. So if we need
			// to merge from this one split, we
			// need to create a whole new tags
			// map.
			oldTags := n.Tags
			n.Tags = make(map[string]string)

			for nf_key, nf_value := range oldTags {
				n.Tags[nf_key] = nf_value
			}

			for nf_key, nf_value := range newTags {
				n.Tags[nf_key] = nf_value
			}

		}

		// Tags templating here
		if tagsTemplates != nil {
			newTags, err := executeTemplates(n, tagsTemplates)
			if err != nil {
				warnFields := map[string]interface{}{
					"namespace":       n.Namespace.Strings(),
					"data":            n.Data,
					"template":        tagsTemplates.DefinedTemplates(),
				}
				log.WithFields(warnFields).Warn(err)
				continue
			}
			for nf_key, nf_value := range newTags {
				n.Tags[nf_key] = nf_value
			}
		}
		newMetrics = append(newMetrics, n)
	}
	return newMetrics, nil
}

func executeTemplates(metric plugin.Metric, template *template.Template) (map[string]string, error) {
	var results map[string]string = make(map[string]string)
	var execBuffer *bytes.Buffer = bytes.NewBufferString("")
	var err error

	for _, tpl := range template.Templates() {
		if tpl.Name() == "" {
			// nope
			continue
		}

		err = tpl.Execute(execBuffer, metric)
		if err != nil {
			return nil, err
		}
		results[tpl.Name()] = execBuffer.String()
		execBuffer.Reset()
	}
	return results, nil
}