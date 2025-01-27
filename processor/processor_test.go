// +build small

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
	"regexp"
	"testing"
	"time"

	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	. "github.com/smartystreets/goconvey/convey"
	yaml "gopkg.in/yaml.v2"
)

func TestProcessor(t *testing.T) {
	processor := New()
	Convey("Create processor", t, func() {
		Convey("So processor should not be nil", func() {
			So(processor, ShouldNotBeNil)
		})
		Convey("So processor should be of type statisticsProcessor", func() {
			So(processor, ShouldHaveSameTypeAs, &Plugin{})
		})
		Convey("processor.GetConfigPolicy should return a config policy", func() {
			configPolicy, err := processor.GetConfigPolicy()
			Convey("So config policy should be a plugin.ConfigPolicy", func() {
				So(configPolicy, ShouldHaveSameTypeAs, plugin.ConfigPolicy{})
			})
			Convey("So err should be nil", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestProcess(t *testing.T) {
	Convey("Test processing of metrics with correct configuration (all features)", t, func() {
		newPlugin := New()
		var matchMap map[string]interface{} = make(map[string]interface{})
		var splitRegexps []string
		splitRegexps = append(splitRegexps, `\|`)
		matchMap[configSplitRegexp] = splitRegexps
		var parseRegexps []string
		parseRegexps = append(parseRegexps, `^feature (?P<feature_name>[A-Za-z0-9]*)$`)
		matchMap[configParseRegexp] = parseRegexps
		config := plugin.Config{}
		var tagsTemplates map[string]string = make(map[string]string, 1)
		tagsTemplates["replaceme"] = "yay: {{ .Tags.feature_name }}"
		tagsTemplates["replaceme_old"] = "{{ .Tags.replaceme }}"
		matchMap[configAddTags] = tagsTemplates
		mmYaml, err := yaml.Marshal(matchMap)
		So(err, ShouldBeNil)
		config["^feature (?P<feature_name>[A-Za-z0-9]*)"] = string(mmYaml)
		// e.g.:
		// config:
		//  "^feature (?P<feature_name>[A-Za-z0-9]*":
		//    parse:
		//      - "^feature (?P<feature_name>[A-Za-z0-9]*"
		//    split:
		//      - "\|"
		//    template:
		//      replaceme: "yay: {{ .Tags.feature_name }}"
		//      replaceme_old: "{{ .Tags.replaceme }}"

		Convey("Testing with a sample", func() {
			logs := []string{
				`feature 1|feature 2|feature 3`,
			}
			mts := make([]plugin.Metric, 0)
			for i := range logs {
				mt := plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "logs", "metric", "log", "message"),
					Timestamp: time.Now(),
					Tags: map[string]string{
						"hello":     "world",
						"replaceme": "boo",
					},
					Data: logs[i],
				}
				mts = append(mts, mt)
			}

			metrics, err := newPlugin.Process(mts, config)

			So(err, ShouldBeNil)
			So(len(metrics), ShouldEqual, 3)
			for _, metric := range metrics {
				So(metric.Tags["hello"], ShouldEqual, "world")
				re, _ := regexp.Compile(parseRegexps[0])
				match := re.FindStringSubmatch(metric.Data.(string))
				So(metric.Tags["feature_name"], ShouldEqual, match[1])
				So(metric.Tags["replaceme"], ShouldEqual, "yay: "+match[1])
				So(metric.Tags["replaceme_old"], ShouldEqual, "boo")
			}

		})
	})

	Convey("Test processing of metrics that MATCH MULTIPLE GATES", t, func() {
		newPlugin := New()
		var matchMap map[string]interface{} = make(map[string]interface{})
		var splitRegexps []string
		splitRegexps = append(splitRegexps, `\|`)
		matchMap[configSplitRegexp] = splitRegexps
		var parseRegexps []string
		parseRegexps = append(parseRegexps, `^feature (?P<feature_name>[A-Za-z0-9]*)$`)
		matchMap[configParseRegexp] = parseRegexps
		config := plugin.Config{}
		var tagsTemplates map[string]string = make(map[string]string, 1)
		tagsTemplates["replaceme"] = "yay: {{ .Tags.feature_name }}"
		tagsTemplates["replaceme_old"] = "{{ .Tags.replaceme }}"
		matchMap[configAddTags] = tagsTemplates
		mmYaml, err := yaml.Marshal(matchMap)
		So(err, ShouldBeNil)
		config["^feature (?P<feature_name>[A-Za-z0-9]*)"] = string(mmYaml)
		config["^.eature (?P<feature_name>[A-Za-z0-9]*)"] = string(mmYaml)

		Convey("Testing with a sample", func() {
			logs := []string{
				`feature 1|feature 2|feature 3`,
			}
			mts := make([]plugin.Metric, 0)
			for i := range logs {
				mt := plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "logs", "metric", "log", "message"),
					Timestamp: time.Now(),
					Tags: map[string]string{
						"hello":     "world",
						"replaceme": "boo",
					},
					Data: logs[i],
				}
				mts = append(mts, mt)
			}

			metrics, err := newPlugin.Process(mts, config)

			So(err, ShouldBeNil)
			So(len(metrics), ShouldEqual, 6)
			for _, metric := range metrics {
				So(metric.Tags["hello"], ShouldEqual, "world")
				re, _ := regexp.Compile(parseRegexps[0])
				match := re.FindStringSubmatch(metric.Data.(string))
				So(metric.Tags["feature_name"], ShouldEqual, match[1])
				So(metric.Tags["replaceme"], ShouldEqual, "yay: "+match[1])
				So(metric.Tags["replaceme_old"], ShouldEqual, "boo")
			}

		})
	})

	Convey("Test split with one metric not matching", t, func() {
		newPlugin := New()
		var matchMap map[string]interface{} = make(map[string]interface{})
		var splitRegexps []string
		splitRegexps = append(splitRegexps, `\|`)
		matchMap[configSplitRegexp] = splitRegexps
		var parseRegexps []string
		parseRegexps = append(parseRegexps, `^feature (?P<feature_name>[A-Za-z0-9]*)$`)
		matchMap[configParseRegexp] = parseRegexps
		config := plugin.Config{}
		var tagsTemplates map[string]string = make(map[string]string, 1)
		tagsTemplates["replaceme"] = "yay: {{ .Tags.feature_name }}"
		tagsTemplates["replaceme_old"] = "{{ .Tags.replaceme }}"
		matchMap[configAddTags] = tagsTemplates
		mmYaml, err := yaml.Marshal(matchMap)
		So(err, ShouldBeNil)
		config["^feature (?P<feature_name>[A-Za-z0-9]*)"] = string(mmYaml)

		Convey("Testing with a sample", func() {
			logs := []string{
				`feature 1|antipattern 2|feature 3`,
			}
			mts := make([]plugin.Metric, 0)
			for i := range logs {
				mt := plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "logs", "metric", "log", "message"),
					Timestamp: time.Now(),
					Tags: map[string]string{
						"hello":     "world",
						"replaceme": "boo",
					},
					Data: logs[i],
				}
				mts = append(mts, mt)
			}

			metrics, err := newPlugin.Process(mts, config)

			So(err, ShouldBeNil)
			So(len(metrics), ShouldEqual, 2)
			for _, metric := range metrics {
				So(metric.Tags["hello"], ShouldEqual, "world")
				re, _ := regexp.Compile(parseRegexps[0])
				match := re.FindStringSubmatch(metric.Data.(string))
				So(metric.Tags["feature_name"], ShouldEqual, match[1])
				So(metric.Tags["replaceme"], ShouldEqual, "yay: "+match[1])
				So(metric.Tags["replaceme_old"], ShouldEqual, "boo")
			}

		})
	})

	Convey("Test parse with exactly zero matches at all", t, func() {
		newPlugin := New()
		var matchMap map[string]interface{} = make(map[string]interface{})
		var splitRegexps []string
		splitRegexps = append(splitRegexps, `\|`)
		matchMap[configSplitRegexp] = splitRegexps
		var parseRegexps []string
		parseRegexps = append(parseRegexps, `^feature (?P<feature_name>[A-Za-z0-9]*)$`)
		matchMap[configParseRegexp] = parseRegexps
		config := plugin.Config{}
		var tagsTemplates map[string]string = make(map[string]string, 1)
		tagsTemplates["replaceme"] = "yay: {{ .Tags.feature_name }}"
		tagsTemplates["replaceme_old"] = "{{ .Tags.replaceme }}"
		matchMap[configAddTags] = tagsTemplates
		mmYaml, err := yaml.Marshal(matchMap)
		So(err, ShouldBeNil)
		config["^feature (?P<feature_name>[A-Za-z0-9]*)"] = string(mmYaml)

		Convey("Testing with a sample", func() {
			logs := []string{
				`this must not match`,
			}
			mts := make([]plugin.Metric, 0)
			for i := range logs {
				mt := plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "logs", "metric", "log", "message"),
					Timestamp: time.Now(),
					Tags: map[string]string{
						"hello":     "world",
						"replaceme": "boo",
					},
					Data: logs[i],
				}
				mts = append(mts, mt)
			}

			metrics, err := newPlugin.Process(mts, config)

			So(err, ShouldBeNil)
			So(len(metrics), ShouldEqual, 1)
			for _, metric := range metrics {
				So(metric.Tags["hello"], ShouldEqual, "world")
			}

		})
	})

	Convey("Test processing of metrics with errors", t, func() {
		newPlugin := New()

		mts := []plugin.Metric{
			plugin.Metric{
				Namespace: plugin.NewNamespace("intel", "logs", "metric", "log", "message"),
				Timestamp: time.Now(),
				Tags:      make(map[string]string),
				Data:      `127.0.0.1 - - [07/Dec/2016:06:00:12 -0500] "GET /v3/users/fa2b2986c200431b8119035d4a47d420/projects HTTP/1.1" 200 446 21747 "-" "python-keystoneclient"`,
			},
		}

		Convey("Testing with missing configurable parameter", func() {
			Convey("Missing parse regexp", func() {
				config := plugin.Config{}
				metrics, err := newPlugin.Process(mts, config)
				So(err, ShouldNotBeNil)
				So(metrics, ShouldBeNil)
			})
		})
	})

	Convey("Test processing of metrics with unexpected log message", t, func() {
		newPlugin := New()
		config := plugin.Config{}
		var matchMap map[string]interface{} = make(map[string]interface{})
		var splitRegexps, parseRegexps []string
		splitRegexps = append(splitRegexps, `\|`)
		matchMap[configSplitRegexp] = splitRegexps
		parseRegexps = append(parseRegexps, `.*`)
		matchMap[configParseRegexp] = parseRegexps
		mmYaml, err := yaml.Marshal(matchMap)
		So(err, ShouldBeNil)
		config[".*"] = string(mmYaml)

		Convey("Unexpected metric data type", func() {
			mts := []plugin.Metric{
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "logs", "metric", "log", "message"),
					Timestamp: time.Now(),
					Tags:      make(map[string]string),
					Data:      123,
				},
			}
			metrics, err := newPlugin.Process(mts, config)
			So(err, ShouldBeNil)
			So(len(metrics), ShouldEqual, 0)
		})
	})
}
