[![CircleCI](https://img.shields.io/circleci/project/github/RedSparr0w/node-csgo-parser.svg)](https://circleci.com/gh/SignifAi/snap-plugin-processor-regexp-engine)
[![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://github.com/SignifAi/snap-plugin-processor-regexp-engine/blob/master/LICENSE)


# snap-plugin-processor-regexp-engine
Split, parse, and re-tag string metrics.

1. [Getting Started](#getting-started)
  * [System Requirements](#system-requirements)
  * [Installation](#installation)
  * [Configuration and Usage](#configuration-and-usage)
2. [Documentation](#documentation)
  * [Roadmap](#roadmap)
3. [Community Support](#community-support)
4. [Contributing](#contributing)
5. [License](#license-and-authors)
6. [Acknowledgements](#acknowledgements)

## Getting Started
### System Requirements 
* [golang 1.8+](https://golang.org/dl/) (needed only for building)

### Operating systems
All OSs currently supported by snap:
* Linux/amd64
* Darwin/amd64

### Installation
#### Download plugin binary:
You can get the pre-built binaries for your OS and architecture under the plugin's [release](https://github.com/SignifAi/snap-plugin-processor-regexp-engine/releases) page.  For Snap, check [here](https://github.com/intelsdi-x/snap/releases).


#### To build the plugin binary:
Fork https://github.com/SignifAi/snap-plugin-processor-regexp-engine

Clone repo into `$GOPATH/src/github.com/SignifAi/`:

```
$ git clone https://github.com/<yourGithubID>/snap-plugin-processor-regexp-engine.git
```


#### Building
The following provides instructions for building the plugin yourself if
you decided to download the source. We assume you already have a $GOPATH
setup for [golang development](https://golang.org/doc/code.html). The
repository utilizes [glide](https://github.com/Masterminds/glide) for
library management.

build:
  ```make```

testing:
  ```make test-small```

### Configuration and Usage
* Set up the [Snap framework](https://github.com/intelsdi-x/snap/blob/master/README.md#getting-started)

#### Load the Plugin
Once the framework is up and running, you can load the plugin.
```
$ snaptel plugin load snap-plugin-processor-regexp-engine
Plugin loaded
Name: regexp-engine
Version: 1
Type: processor
Signed: false
Loaded Time: Sat, 18 Mar 2017 13:28:45 PDT
```

## Documentation

This is a Snap processor plugin; metrics are passed in from the collector (or
from previous processors), modified here, and passed on "down the chain" to 
either more processors or publishers. This plugin can split one metric into
many by a regular expression, then enrich each of the split metrics with further
regular expressions and [golang templating](https://golang.org/pkg/text/template/). 

Upon receiving a metric, this plugin will:

1. Attempt to match the metric against a "gate" provided in configuration.
  a. If it matches, the plugin will process against that gate's configuration by:
    1. Splitting the metrics by the regexes provided in the 'split' section, in order
    2. Attempting to match each split against the original gate, filtering it out
       entirely on failure
    3. Parsing the regexes, using capture groups to capture parts of the string to
       store in the metric as tags
    4. Using golang templating against the metric as a whole to create or override
       further tags for the metric.
2. If _no_ gates match, the metric is simply passed "down the chain" as-is. 

If the metric matches more than one gate, it will be processed for each gate. 

Imagine a task manifest like:

```yaml
---
  version: 1
  schedule:
    type: "simple"
    interval: "3s"
  max-failures: 10
  workflow:
    collect:
      config:
        metric_name: featurelistfile
        cache_dir: /var/lib/snap/logcache
        log_dir: /var/log
        log_file: featurelistfile
        splitter_type: new-line
        collection_time: 2s
        metrics_limit: 1000
      metrics:
        /intel/logs/*: {}
      publish:
        - plugin_name: "regexp-engine"
          config:
            "^feature ([A-Za-z0-9]+)":
              split:
                - "\|"
              parse:
                - "^feature (?P<feature_name>)"
              tags:
                feature_index: "{{ .Tags.feature_name }}"
          publish:
            - plugin_name: "file"
              config:
                file: /tmp/logmetrics
```

And a list of metrics comes into the plugin like:

```json
[{
  "Name": "/intel/logs/featurelistfile/message",
  "Value": "feature 1|feature 2|feature 3",
  "Tags": {},
  ...
}]
```

The resulting metrics list will be passed down like:

```json
[{
  "Name": "/intel/logs/featurelistfile/message",
  "Value": "feature 1",
  "Tags": {
      "feature_name": "1",
      "feature_index": "1"
  },
  ...
},
{
  "Name": "/intel/logs/featurelistfile/message",
  "Value": "feature 2",
  "Tags": {
    "feature_name": "2",
    "feature_index": "2"
  },
  ...
},
{
  "Name": "/intel/logs/featurelistfile/message",
  "Value": "feature 3",
  "Tags": {
    "feature_name": "3",
    "feature_index": "3"
  },
  ...
}]
```

### Parse Details

#### Gating

The configuration is a yaml dict where the keys are regexp matches --
"gates" -- and the values are further dicts that express instructions
on handling a metric with data matching the key. Take this sample config
for instance:

```yaml
config:
  "^<[^>]+> .*$":
    parse:
      - "<(?<user>[^>]+)> some IRC message"
```

A metric with a value of "<zcarlson> test message" would gain a 
'user' tag of 'zcarlson', while a metric with a value of "%some_other_value%"
would pass through unprocessed.

The next few sections will instruct how to define the parsing of string
metrics that match this gate. 

#### Split phase

If you want to split the metrics based on a string (regexp), use the
'split' key with a list value. The list is a list of regular expressions
that will be used to split the string metrics. So for a 
config like this:

```yaml
config:
  ".*\\$.*\\|.*\\$.*":
    split:
      - '\\$'
      - '\\|'
    parse:
      - '.*'
```

And a metric string value like `map1|key1$value1|map2$key2|value2`, you
would get metrics with values "map1", "key1", "value1", "map2", "key2",
"value2". The splits here are applied in the order they are defined.

#### Match-again phase

If a metric was split, the gate match is attempted against the split
metric's value; if there is no match, the split metric is discarded. 

#### Parse phase

The `parse` key is required, and its value is also a list of regular
expressions, again applied _in order_. For instance:

```yaml
config:
  "instanceHostname\": \"([^\"]+)\"":
    parse:
      - 'instanceHostname\": \"(?P<hostname>[^\"]*)\"'
      - 'otherHostname\": \"(?P<hostname>[^\"]+)\"'
```

For a metric with a JSON-like value like `{"instanceHostname": "myhost1"}`, 
the `hostname` tag will be `myhost1`; however, if the JSON-value looked like:

`{"instanceHostname": "myhost1", "otherHostname": "differenthost"}`

The `hostname` tag would be "differenthost1". Remember, though, parse tags
only get set on match, so JSON like this:

`{"instanceHostname": "myhost1", "otherHostname": ""}`

Would still see `hostname` set to "myhost1" (because the otherHostname
value is empty and the regex requires at least one non-quote character). 

As you might expect, JSON like:

`{"instanceHostname": ""}`

Will end up setting the `hostname` tag to an empty string.

(If you want to do _just_ a split or template for some reason, you can
set parse to a list containing only a ".*", but parsing is the primary
intended use of the plugin)

#### Template phase

The metric is essentially filled out after the parse phase, but you can
do some additional processing/tag-setting with 
[golang templating](https://golang.org/pkg/text/template/) and the 
`template` key, whose value is another dict where the keys are tag names
and the values are golang templates (see the documentation linked) that
provide the intended values for a tag. For instance:


```
config:
  "instanceHostname\": \"(?P<host>[^\"]+)\":
    parse:
      - "instanceHostname\": \"(?P<host>[^\"]+)\""
      - "instanceHttpPort\": (?P<port>)"
    template:
      url: "http://{{ .Tags.host }}:{{ .Tags.port }}/"
```

### Roadmap

We keep working on more feature and will update the processor as needed.

## Community Support

Open an issue and we will respond.

## Contributing 

We love contributions!

The most immediately helpful way you can benefit this plug-in is by cloning the repository, adding some further examples and submitting a pull request.

## License
Released under the Apache 2.0 [License](LICENSE).

## Acknowledgements
* Author: [@SignifAi](https://github.com/SignifAi/)
* Info: www.signifai.io
