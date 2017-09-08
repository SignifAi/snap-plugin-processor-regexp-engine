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
	yaml "gopkg.in/yaml.v2"
)

const (
	//Name of the plugin
	Name = "regexp-engine"
	//Version of the plugin
	Version = 1

	configSplitRegexp = "split"
	configParseRegexp = "parse"
	configAddTags     = "tags"
)

type Plugin struct {
}

type internalConfig struct {
	Parse    []*regexp.Regexp
	Split    []*regexp.Regexp
	Template *template.Template
}

// New() returns a new instance of the plugin
func New() *Plugin {
	p := &Plugin{}
	return p
}

// GetConfigPolicy returns the config policy
func (p *Plugin) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	return *policy, nil
}

// Process processes the data
func (p *Plugin) Process(metrics []plugin.Metric, cfg plugin.Config) ([]plugin.Metric, error) {
	// Configuration
	var err error
	var tagsTemplates *template.Template
	var internalCfg map[*regexp.Regexp]internalConfig = make(map[*regexp.Regexp]internalConfig)
	var splitRegexes, parseRegexes []*regexp.Regexp
	var mapRegex *regexp.Regexp
	var singletonList []plugin.Metric
	var didMatch bool
	var parsedMetrics, newMetrics []plugin.Metric
	var rawRegexCfg map[string]interface{} = make(map[string]interface{})

	for rawRegex, interfaceRegexCfg := range cfg {
		mapRegex, err = regexp.Compile(rawRegex)
		if err != nil {
			return nil, err
		}

		rawStringRegexCfg, ok := interfaceRegexCfg.(string)
		if ok {
			err = yaml.Unmarshal([]byte(rawStringRegexCfg), rawRegexCfg)
			if err != nil {
				return nil, err
			}
		}

		splitRegexesRaw, ok := rawRegexCfg[configSplitRegexp].([]interface{})
		if ok {
			splitRegexes, err = compileRegexes(splitRegexesRaw)
			if err != nil {
				return nil, fmt.Errorf("Couldn't compile split regexes: %v", splitRegexesRaw)
			}
		}

		parseRegexesRaw, ok := rawRegexCfg[configParseRegexp].([]interface{})
		if !ok {
			return nil, fmt.Errorf("Must specify parse regexps at least")
		}
		parseRegexes, err = compileRegexes(parseRegexesRaw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compile a regex: %v", err)
		}

		tagTemplatesRaw, ok := rawRegexCfg[configAddTags].(map[interface{}]interface{})
		if ok {
			tagsTemplates, err = compileTemplates(tagTemplatesRaw)
			if err != nil {
				return nil, err
			}
		}

		internalCfg[mapRegex] = internalConfig{
			Parse:    parseRegexes,
			Split:    splitRegexes,
			Template: tagsTemplates,
		}
	}

	if len(internalCfg) < 1 {
		return nil, fmt.Errorf("At least one match->parse block must be specified")
	}

	newMetrics = make([]plugin.Metric, 0)

MetricIter:
	for _, m := range metrics {
		didMatch = false
		for mustMatch, matchConfig := range internalCfg {
			testStr, ok := m.Data.(string)
			if !ok {
				warnFields := map[string]interface{}{
					"namespace": m.Namespace.Strings(),
					"data":      m.Data,
				}
				log.WithFields(warnFields).Warn("Match Phase: unexpected data type, plugin processes only strings")
				continue MetricIter
			}
			if mustMatch.FindStringSubmatch(testStr) != nil {
				didMatch = true
				if matchConfig.Split != nil {
					splitMetrics, err := splitMetric(m, matchConfig.Split)
					if err == nil {
						parsedMetrics, err = processMetrics(splitMetrics, matchConfig.Parse, mustMatch, matchConfig.Template)
						if err != nil {
							return nil, err
						}
					}
				} else {
					singletonList = make([]plugin.Metric, 1)
					singletonList = append(singletonList, m)
					parsedMetrics, err = processMetrics(singletonList, matchConfig.Parse, mustMatch, matchConfig.Template)
					if err != nil {
						return nil, err
					}
				}
				newMetrics = append(newMetrics, parsedMetrics...)
			}
		}

		// If we matched, we parsed
		// If we did not match, emit the "original"
		if !didMatch {
			newMetrics = append(newMetrics, m)
		}
	}

	return newMetrics, nil
}

func parse(message string, regexes []*regexp.Regexp) (map[string]string, error) {
	var fields map[string]string
	for _, regex := range regexes {
		match := regex.FindStringSubmatch(message)
		for i, name := range regex.SubexpNames() {
			if i > 0 && i <= len(match) {
				if fields == nil {
					fields = make(map[string]string, 0)
				}
				fields[name] = match[i]
			}
		}
	}
	return fields, nil
}

func compileRegexes(from []interface{}) ([]*regexp.Regexp, error) {
	var regexes []*regexp.Regexp
	for _, iexpr := range from {
		expr, ok := iexpr.(string)
		if !ok {
			return nil, fmt.Errorf("iexpr not a string but %T with value %v", iexpr, iexpr)
		}
		regex, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, regex)
	}
	return regexes, nil
}

func compileTemplates(templates map[interface{}]interface{}) (*template.Template, error) {
	rootTemplate := template.New("")
	for iTag, iTagTemplate := range templates {
		tag, ok := iTag.(string)
		if !ok {
			return nil, fmt.Errorf("Tag isn't a string, but a %T with value %v", iTag, iTag)
		}
		if tag == "" {
			// nope
			continue
		}
		newTagTemplate := rootTemplate.New(tag)
		if newTagTemplate == nil {
			return nil, fmt.Errorf("Couldn't create template for tag %v", tag)
		}
		tagTemplate, ok := iTagTemplate.(string)
		if !ok {
			return nil, fmt.Errorf("Template value %v was not a string but a %T", iTagTemplate, iTagTemplate)
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

func processMetrics(metrics []plugin.Metric, regexps []*regexp.Regexp, mustMatch *regexp.Regexp, tagsTemplates *template.Template) ([]plugin.Metric, error) {
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
		if mustMatch.FindStringSubmatch(logBlock) == nil {
			continue
		}

		newTags, err := parse(logBlock, regexps)
		if err != nil {
			warnFields := map[string]interface{}{
				"namespace":       n.Namespace.Strings(),
				"data":            n.Data,
				configParseRegexp: regexps,
			}
			log.WithFields(warnFields).Warn(err)
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
					"namespace": n.Namespace.Strings(),
					"data":      n.Data,
					"template":  tagsTemplates.DefinedTemplates(),
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
