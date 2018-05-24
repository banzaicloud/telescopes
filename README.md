
[![CircleCI](https://circleci.com/gh/banzaicloud/cluster-recommender/tree/master.svg?style=shield)](https://circleci.com/gh/banzaicloud/cluster-recommender/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/cluster-recommender)](https://goreportcard.com/report/github.com/banzaicloud/cluster-recommender)
![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1651/badge)](https://bestpractices.coreinfrastructure.org/projects/1651)

# Cluster recommender

The Banzai Cloud cluster recommender is a standalone project in the [Pipeline](https://github.com/banzaicloud/pipeline) ecosystem.
It's main purpose is to recommend cluster instance types and full cluster layouts consisting EC2 spot or Google Cloud preemptible instances.

## Quick start

Building the project is as simple as running a go build command. The result is a statically linked executable binary.

```
go build .
```

The following options can be configured when starting the exporter (with defaults):

```
./cluster-recommender --help
Usage of ./cluster-recommender:
  -listen-address string
        The address to listen on for HTTP requests. (default ":9090")
  -log-level string
        log level (default "info")
  -product-info-renewal-interval duration
        Duration (in go syntax) between renewing the ec2 product info. Example: 2h30m (default 24h0m0s)
  -prometheus-address string
        http address of a Prometheus instance that has AWS spot price metrics via banzaicloud/spot-price-exporter. If empty, the recommender will use current spot prices queried directly from the AWS API.

```
 
## API calls

*For a complete OpenAPI 3.0 documentation, check out this [URL](https://editor.swagger.io/?url=https://raw.githubusercontent.com/banzaicloud/cluster-recommender/master/docs/openapi/recommender.yaml).*

#### `POST: api/v1/recommender/:provider/:region/cluster`

This endpoint returns a recommended cluster layout on a specific provider in a specific region, that contains on-demand and spot priced node pools.

**Request parameters:**

`sumCpu`: requested sum of CPUs in the cluster (approximately)

`sumMem`: requested sum of Memory in the cluster (approximately)

`minNodes`: minimum number of nodes in the cluster (optional)

`maxNodes`: maximum number of nodes in the cluster

`onDemandPct`: percentage of on-demand (regular) nodes in the cluster

`zones`: availability zones in the cluster - specifying multiple zones will recommend a multi-zone cluster

`sameSize`: signals if the resulting instance types should be similarly sized, or can be completely diverse


**`cURL` example**

```
curl -sX POST -d '{"sumCpu": 100, "sumMem":200, "sumGpu":0, "minNodes":10, "maxNodes":30, "sameSize":true, "onDemandPct":30, "zones":[]}' "localhost:9092/api/v1/recommender/ec2/eu-west-1/cluster" | jq .
```
**Sample response:**
```
{
  "Provider": "aws",
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

**1. How do I configure my AWS credentials with the project?**

The project is using the standard [AWS SDK for Go](https://aws.amazon.com/sdk-for-go/), so credentials can be configured via
environment variables, shared credential files and via AWS instance profiles. To learn more about that read the [Specifying Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) section of the SDK docs.

**2. Why do I see messages like `DEBU[0001] Getting available instance types from AWS API. [region=ap-northeast-2, memory=0.5]` when starting the recommender?**

After the recommender is started, it takes ~2-3 minutes to cache all the product information (like instance types) from AWS (in memory).
AWS is releasing new instance types and regions quite frequently and also changes on-demand pricing from time to time.
So it is necessary to keep this info up-to-date without needing to modify it manually every time something changes on the AWS side.
After the initial query, the recommender will parse this info from the AWS Pricing API once per day.
The frequency of this querying and caching is configurable with the `-product-info-renewal-interval` switch and is set to `24h` by default.

**3. What happens if the recommender cannot cache the AWS product info?**

If caching fails, the recommender will try to reach the AWS Pricing List API on the fly when a request is sent (and it will also cache the resulting information).
If that fails as well, the recommendation will return with an error.

**4. What kind of AWS permissions do I need to use the project?**

The recommender is querying the AWS [Pricing API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) to keep up-to-date info
about instance types, regions and on-demand pricing.
You'll need IAM access as described here in [example 11](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-permissions-ref.html#example-policy-pe-api) of the AWS IAM docs.

If you don't use Prometheus to track spot instance pricing, you'll need to be able to access the [spot price history](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotPriceHistory.html) from the AWS API as well with your IAM user.
It means giving permission to `ec2:DescribeSpotPriceHistory`.

**5. How are the spot prices determined?**

The spot prices can be queried from 2 different sources. You can use Prometheus with our [spot price exporter](https://github.com/banzaicloud/spot-price-exporter) configured,
or you can use the recommender without Prometheus.
In that case the current spot prices will be queried from the AWS API and that will be the base of the recommendation.

**6. What is the advantage of using Prometheus to determine spot prices?**

Prometheus is becoming the de-facto monitoring solution in the cloud native world, and it includes a time series database as well.
When using the Banzai Cloud [spot price exporter](https://github.com/banzaicloud/spot-price-exporter), spot price history will be collected as time series data and
can be queried for averages, maximums and predictions.
It gives a richer picture than relying on the current spot price that can be a spike, or on a downward or upward trend.
You can fine tune your query (with the `-prometheus-query` switch) if you want to change the way spot instance prices are scored.
By default the spot price averages of the last week are queried and instance types are sorted based on this score.

**7. What happens if my Prometheus server cannot be reached or if it doesn't have the necessary spot price metrics?**

If the recommender fails to reach the Prometheus query API, or it couldn't find proper metrics, it will fall back to querying the current spot prices from the AWS API.

**8. How is this project different from EC2 Spot Advisor and Spot Fleet?**

The recommender is similar to the EC2 Spot Advisor, it is also recommending different spot instance types for diverse clusters.
But the EC2 Spot Advisor has no externally available API, it is only available from the AWS Console, and it is only available to create Spot Fleets.
We wanted to build an independent solution where the recommendation can be used in an arbitrary way and it doesn't require Spot Fleets to work.
We are also keeping Kubernetes in mind - primarily to build our PaaS, [Pipeline](https://github.com/banzaicloud/pipeline) - and Kubernetes doesn't support
Spot Fleets out of the box for starting clusters (via Kubicorn, kops or any other tool) or for autoscaling, rather it uses standard Auto Scaling Groups for node pools
and that model fits the recommendation perfectly.
We also wanted to include on-demand instances to keep some part of the cluster completely safe.
And although EC2 is the only supported platform for now, we'd like to add support for Google Cloud and other providers as well.

**9. Will this project start instances on my behalf on my cloud provider?**

No, this project will never start instances. It only uses the cloud credentials to query region, instance type and pricing information.
The API response is a cluster description built from node pools of different instance types.
It is the responsibility of the user to start and manage the autoscaling groups based on the response.
The [Pipeline](https://github.com/banzaicloud/pipeline) and [Hollowtrees](https://github.com/banzaicloud/hollowtrees) projects are helping with that.

**10. How does the recommender decide which instance types to include in the recommendation?**

The recommender will list one node pool that contains on-demand (regular) instances.
The instance type of the on-demand node pool is decided based on price, and the CPU/memory ratio and the min/max cluster size in the request.
For the spot type node pools: all the instance types in the region are getting a price score - based on the Prometheus or AWS API info - and are sorted by that score.
Depending on the cluster's size the first N types are returned, and the number of instances are calculated to have about equal sized pools in terms of sum CPU/memory.

**11. Why do I see node pools with `SumNodes=0` in the recommendation?**

Those instance types are the next best recommendations after the node pools that contain instances in the response, but it's not needed to further diversify the cluster with them.
Because the response is only a recommendation and it won't start instances on the cloud provider, it is possible to fine tune the recommendation before creating a cluster.
It means that a user can remove recommended node pools (e.g.: because they don't want burstable instance types, like `t2`) and can also add new ones.
If they want to add new node pools (e.g. instead of a recommended one), it makes sense for them to include one of the 0-sized node pools and to increase the node count there.

**12. How are availability zones handled?**

Requested availability zones must be sent in the API request. When listing multiple zones, the response will contain a multi-zone recommendation,
and *all* node pools in the response are meant to span across multiple zones. Having different node pools in different zones are not supported.
Because spot prices can be different across availability zones, in this case the instance type price score is averaged across availability zones.

**13. Is there a Google Cloud implementation?**

Not yet, but we're planning to release it in the near future.

**14. There's no bid pricing on Google Cloud, what will the recommender take into account there?**

Even there's no bid pricing, Google Cloud can take your preemptible VMs away any time, and it still makes sense to diversify your node pools and minimize the risk
of losing all your instances at once. Second, Google Cloud VM types are also complicated - there are standard, high-memory, high-cpu instances in different sizes,
also special VM types, like shared-core and custom machine types, not to mention GPUs - so it makes sense to have a recommendation that takes these things into account as well.
Managing a long-running cluster built from preemptible instances is also a hard task, we're working on that as well as part of the [Hollowtrees](https://github.com/banzaicloud/hollowtrees) project.

**15. How is this project related to [Pipeline](https://github.com/banzaicloud/pipeline)?**

Pipeline is able to start clusters with multiple node pools. This API is used in the Pipeline UI and CLI to recommend a cluster setup and to make it easy for
a user to start a properly diversified spot instance based cluster. The recommender itself is not starting instances, it is the responsibility of Pipeline.
The recommendation can also be customized on the UI and CLI before sending the cluster create request to Pipeline.

**16. How is this project related to [Hollowtrees](https://github.com/banzaicloud/hollowtrees)**

This project is only capable of recommending a static cluster layout that can be used to start a properly diversified spot cluster.
But that is only one part of the whole picture: after the cluster is started it is still needed to be managed.
Spot instances can be taken away by the cloud provider or their price can change, so the cluster may need to be modified while running.
This maintenance work is done by Hollowtrees, that project is keeping the spot instance based cluster stable during its whole lifecycle.
When some spot instances are taken away, Hollowtrees can ask the recommender to find substitutes based on the current layout.

**17. What happens when the spot price of one of the instance types is rising after my cluster is running?**

It is out of the scope of this project, but [Hollowtrees](https://github.com/banzaicloud/hollowtrees) will be able to handle that situtation.
See the answer above for more information.

**18. Is this project production ready?**

Not yet. To make this project production ready, we need at least the following things:
 - cover the code base with unit tests
 - authentication on the API
 - API validations

**19. What is on the project roadmap for the near future?**

The first priority is to stabilize the API and to make it production ready (see above).
Other than that, these are the things we are planning to add soon:
 - GPU support
 - filters for instance type I/O performance, network performance
 - handle the sameSize switch to recommend similar types
 - Google Cloud Preemptible instances
