
[![CircleCI](https://circleci.com/gh/banzaicloud/telescopes/tree/master.svg?style=shield)](https://circleci.com/gh/banzaicloud/telescopes/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/telescopes)](https://goreportcard.com/report/github.com/banzaicloud/telescopes)
![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1651/badge)](https://bestpractices.coreinfrastructure.org/projects/1651)

*Telescopes is a world-class left that offers lots of surfing flexibility and allows entry to a flawless, steady tube section. It's predictable and rarely pinches, is deeper than other reefs and handles the constant traffic of intermediates and experts alike.*

*Telescopes is a cluster instance types and full cluster layout recommender consisting of on-demand and EC2 spot or Google Cloud preemptible instances. Based on predefined resource requirements as CPU, memory, GPU, network, etc it recommends a diverse set of cost optimized node pools.*

# Telescopes

The `Banzai Cloud Telescopes` is a cluster recommender application; its main purpose is to recommend cluster instance types and full cluster layouts consisting EC2 spot or Google Cloud preemptible instances. The application operates on cloud provider product information retrieved from the [Productinfo](https://github.com/banzaicloud/productinfo) application.

`Banzai Cloud Telescopes` exposes a rest API for accepting `recommendation requests`


## Quick start

Building the project is as simple as running a go build command. The result is a statically linked executable binary.

```
go build .
```

The application can be started with the following arguments:

```
Usage of ./telescopes:
      --dev-mode                     development mode, if true token based authentication is disabled, false by default
      --help                         print usage
      --listen-address string        the address where the server listens to HTTP requests. (default ":9090")
      --log-format string            log format
      --log-level string             log level (default "info")
      --metrics-address string       the address where internal metrics are exposed (default ":9900")
      --metrics-enabled              internal metrics are exposed if enabled
      --productinfo-address string   the address of the Product Info service to retrieve attribute and pricing info [format=scheme://host:port/basepath] (default "http://localhost:9090/api/v1")
      --tokensigningkey string       The token signing key for the authentication process
      --vault-address string         The vault address for authentication token management (default ":8200")
```

> We have recently added Oauth2 (bearer) token based authentication to `telescopes` which is enabled by default. In order for this to work, the application needs to be connected to a component (eg.: [Banzai Cloud Pipeline ](http://github.com/banzaicloud/pipeline)) capable to emit the `bearer token` The connection is made through a `vault` instance (which' address must be specified by the --vault-address flag) The --token-signing-key also must be specified in this case (this is a string secret that is shared with the token emitter component)

*The authentication can be switched off by starting the application in development mode (--dev-mode flag) - please note that other functionality can also be affected!*

For more information on how to set up `Banzai Cloud Pipeline` instance for using it for authentication (emitting bearer tokens) please check the following documents:
* https://github.com/banzaicloud/pipeline/blob/master/docs/github-app.md
* https://github.com/banzaicloud/pipeline/blob/master/docs/pipeline-howto.md

## API calls

*For a complete OpenAPI 3.0 documentation, check out this [URL](https://editor.swagger.io/?url=https://raw.githubusercontent.com/banzaicloud/telescopes/master/api/openapi-spec/recommender.yaml).*


#### `POST: api/v1/recommender/:provider/:service/:region/cluster`

This endpoint returns a recommended cluster layout on a specific provider in a specific region, that contains on-demand and spot priced node pools.

**Request parameters:**

`sumCpu`: requested sum of CPUs in the cluster (approximately)

`sumMem`: requested sum of Memory in the cluster (approximately)

`minNodes`: minimum number of nodes in the cluster (optional)

`maxNodes`: maximum number of nodes in the cluster

`onDemandPct`: percentage of on-demand (regular) nodes in the cluster

`allowBurst`: signals whether burst type instances are allowed or not in the recommendation (defaults to true)

`zones`: availability zones in the cluster - specifying multiple zones will recommend a multi-zone cluster

`sameSize`: signals if the resulting instance types should be similarly sized, or can be completely diverse

`allowBurst`: are burst instances allowed in recommendation

`networkPerf`: networkPerf specifies the network performance category

`excludes`: excludes is a blacklist - a list with vm types to be excluded from the recommendation

`includes`: includes is a whitelist - a list with vm types to be contained in the recommendation



**`cURL` example**

```
curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ8.eyJhdWQiOiJodHRwczovL3BpcGVsaW5lLmJhbnphaWNsb3VkLmNvbSIsImp0aSI6IjUxMWE1ODQyLWYxMmUtNDk1NC04YTg2LTVjNmUyOWRmZTg5YiIsImlhdCI6MTUyODE5MTM0MSwiaXNzIjoiaHR0cHM6Ly9iYW56YWljbG91ZC5jb20vIiwic3ViIjoiMSIsInNjb3BlIjoiYXBpOmludm9rZSIsInR5cGUiOiJ1c2VyIiwidGV4dCI6ImxwdXNrYXMifQ.azhx0MbuLp7vQ1XmwPYrOqFG5vWZVh-hkzmHig8nnvs' \POST -d '{"sumCpu": 100, "sumMem":200, "sumGpu":0, "minNodes":10, "maxNodes":30, "sameSize":true, "onDemandPct":30, "zones":[]}' "localhost:9090/api/v1/recommender/amazon/compute/eu-central-1/cluster" | jq .
```
**Sample response:**
```
{
  "Provider": "amazon",
  "zones": [
      "eu-west-1a",
      "eu-west-1b",
      "eu-west-1c",
  ],
  "NodePools": [
    {
      "VmType": {
        "Type": "c5.xlarge",
        "AvgPrice": 0.07325009009008994,
        "OnDemandPrice": 0.19200000166893005,
        "Cpus": 4,
        "Mem": 8,
        "Gpus": 0
      },
      "SumNodes": 8,
      "VmClass": "regular"
    },
    {
      "VmType": {
        "Type": "m1.xlarge",
        "AvgPrice": 0.03789999999999985,
        "OnDemandPrice": 0.3790000081062317,
        "Cpus": 4,
        "Mem": 15,
        "Gpus": 0
      },
      "SumNodes": 4,
      "VmClass": "spot"
    },
    {
      "VmType": {
        "Type": "m2.2xlarge",
        "AvgPrice": 0.05499999999999986,
        "OnDemandPrice": 0.550000011920929,
        "Cpus": 4,
        "Mem": 34.20000076293945,
        "Gpus": 0
      },
      "SumNodes": 4,
      "VmClass": "spot"
    },
    {
      "VmType": {
        "Type": "m2.4xlarge",
        "AvgPrice": 0.10999999999999972,
        "OnDemandPrice": 1.100000023841858,
        "Cpus": 8,
        "Mem": 68.4000015258789,
        "Gpus": 0
      },
      "SumNodes": 2,
      "VmClass": "spot"
    },
...
  ]
}
```

## FAQ

**1. Will this project start instances on my behalf on my cloud provider?**

No, this project will never start instances. The API response is a cluster description built from node pools of different instance types.It is the responsibility of the user to start and manage the autoscaling groups based on the response. The [Pipeline](https://github.com/banzaicloud/pipeline) and [Hollowtrees](https://github.com/banzaicloud/hollowtrees) projects are helping with that.

**2. How does the recommender decide which instance types to include in the recommendation?**

The recommender will list one node pool that contains on-demand (regular) instances.
The instance type of the on-demand node pool is decided based on price, and the CPU/memory ratio and the min/max cluster size in the request.
For the spot type node pools: all the instance types in the region are getting a price score - based on the Prometheus or AWS API info - and are sorted by that score.
Depending on the cluster's size the first N types are returned, and the number of instances are calculated to have about equal sized pools in terms of sum CPU/memory.

**3. Why do I see node pools with `SumNodes=0` in the recommendation?**

Those instance types are the next best recommendations after the node pools that contain instances in the response, but it's not needed to further diversify the cluster with them.
Because the response is only a recommendation and it won't start instances on the cloud provider, it is possible to fine tune the recommendation before creating a cluster.
It means that a user can remove recommended node pools (e.g.: because they don't want burstable instance types, like `t2`) and can also add new ones.
If they want to add new node pools (e.g. instead of a recommended one), it makes sense for them to include one of the 0-sized node pools and to increase the node count there.

**4. How are availability zones handled?**

Requested availability zones must be sent in the API request. When listing multiple zones, the response will contain a multi-zone recommendation,
and *all* node pools in the response are meant to span across multiple zones. Having different node pools in different zones are not supported.
Because spot prices can be different across availability zones, in this case the instance type price score is averaged across availability zones.

**5. How is this project different from EC2 Spot Advisor and Spot Fleet?**

The most important difference is that Telescopes is working across different cloud providers, instead of locking in to AWS.
Otherwise the recommender is similar to the EC2 Spot Advisor, it is also recommending different spot instance types to have diverse clusters.
But the EC2 Spot Advisor has no externally available API, it is only available from the AWS Console, and it is only available to create Spot Fleets.
We wanted to build an independent solution where the recommendation can be used in an arbitrary way and it doesn't require Spot Fleets to work.
We are also keeping Kubernetes in mind - primarily to build our PaaS, [Pipeline](https://github.com/banzaicloud/pipeline) - and Kubernetes doesn't support
Spot Fleets out of the box for starting clusters (via Kubicorn, kops or any other tool) or for autoscaling, rather it uses standard Auto Scaling Groups for node pools
and that model fits the recommendation perfectly.
We also wanted to include on-demand instances to keep some part of the cluster completely safe.

**6. There's no bid pricing on Google Cloud, what will the recommender take into account there?**

Even there's no bid pricing, Google Cloud can take your preemptible VMs away any time, and it still makes sense to diversify your node pools and minimize the risk
of losing all your instances at once. Second, Google Cloud VM types are also complicated - there are standard, high-memory, high-cpu instances in different sizes,
also special VM types, like shared-core and custom machine types, not to mention GPUs - so it makes sense to have a recommendation that takes these things into account as well.
Managing a long-running cluster built from preemptible instances is also a hard task, we're working on that as well as part of the [Hollowtrees](https://github.com/banzaicloud/hollowtrees) project.

**7. How is this project related to [Pipeline](https://github.com/banzaicloud/pipeline)?**

Pipeline is able to start clusters with multiple node pools. This API is used in the Pipeline UI and CLI to recommend a cluster setup and to make it easy for
a user to start a properly diversified spot instance based cluster. The recommender itself is not starting instances, it is the responsibility of Pipeline.
The recommendation can also be customized on the UI and CLI before sending the cluster create request to Pipeline.

Pipeline also provides the bearer token to be used for authentication when accessing the telescope API.

**8. Can the authentication be disabled from the telescopes API?**

Authentication is enabled by default on the API. It *is* however possible to disable it by starting the application in development mode. (just start the app with the `--dev-mode` flag)

Beware that (unrelated) behavior of the application may be affected in this mode (logging for example)
It's not recommended to use the application in production with this flag!

**9. How is this project related to [Hollowtrees](https://github.com/banzaicloud/hollowtrees)**

This project is only capable of recommending a static cluster layout that can be used to start a properly diversified spot cluster.
But that is only one part of the whole picture: after the cluster is started it is still needed to be managed.
Spot instances can be taken away by the cloud provider or their price can change, so the cluster may need to be modified while running.
This maintenance work is done by Hollowtrees, that project is keeping the spot instance based cluster stable during its whole lifecycle.
When some spot instances are taken away, Hollowtrees can ask the recommender to find substitutes based on the current layout.

**10. What happens when the spot price of one of the instance types is rising after my cluster is running?**

It is out of the scope of this project, but [Hollowtrees](https://github.com/banzaicloud/hollowtrees) will be able to handle that situtation.
See the answer above for more information.

**11. Is this project production ready?**

Almost there. We are using this already internally and plan to GA it soon.

**12. What is on the project roadmap for the near future?**

The first priority is to stabilize the API and to make it production ready (see above).
Other than that, these are the things we are planning to add soon:
 - GPU support
 - filters for instance type I/O performance
 - handle the sameSize switch to recommend similar types

### License

Copyright (c) 2017-2018 [Banzai Cloud, Inc.](https://banzaicloud.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
