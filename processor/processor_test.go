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
	"testing"
	"time"

	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	. "github.com/smartystreets/goconvey/convey"
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
	Convey("Test processing of metrics with correct configuration", t, func() {
		newPlugin := New()
		config := plugin.Config{}
		config[configSplitRegexp] = defaultSplitRegexp

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
						"hello": "world",
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
			Convey("Missing split regexp", func() {
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
		config[configSplitRegexp] = defaultSplitRegexp

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
