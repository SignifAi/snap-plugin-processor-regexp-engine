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

Once the task file has been created, you can create and watch the task.
```
$ snaptel task create -t tasks/signafai.yaml
Using task manifest to create task
Task created
ID: 72869b36-def6-47c4-9db2-822f93bb9d1f
Name: Task-72869b36-def6-47c4-9db2-822f93bb9d1f
State: Running

$ snaptel task list
ID                                       NAME
STATE     ...
72869b36-def6-47c4-9db2-822f93bb9d1f
Task-72869b36-def6-47c4-9db2-822f93bb9d1f    Running   ...
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
