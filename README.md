# Cluster recommender

The Banzai Cloud cluster recommender is a standalone project in the [Pipeline](https://github.com/banzaicloud/pipeline) ecosystem.
It's main purpose is to recommend cluster instance types and full cluster layouts consisting EC2 spot or Google Cloud preemptible instances.

### Quick start

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
 
### API calls
 
To query the recommender API you must specify the approximate sum of the CPUs and Memory requested for the cluster and also the minimum and maximum number of nodes
you'd like to have. You can set the percentage of on-demand nodes in your cluster as well.
```
curl -sX POST -d '{"provider":"ec2", "sumCpu": 100, "sumMem":200, "sumGpu":0, "minNodes":10, "maxNodes":30, "sameSize":true, "onDemandPct":30, "zones":[]}' "localhost:9092/api/v1/recommender/ec2/eu-west-1/cluster" | jq .
```
Sample response:
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

### FAQ

1. How do I configure my AWS credentials with the project?

The project is using the standard [AWS SDK for Go](https://aws.amazon.com/sdk-for-go/), so credentials can be configured via
environment variables, shared credential files and via AWS instance profiles. To learn more about that read the [Specifying Credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) section of the SDK docs.

2. Why do I see messages like `DEBU[0001] Getting available instance types from AWS API. [region=ap-northeast-2, memory=0.5]` when starting the recommender?

After the recommender is started, it takes ~2-3 minutes to cache all the product information (like instance types) from AWS (in memory).
AWS is releasing new instance types and regions quite frequently and also changes on-demand pricing from time to time.
So it is necessary to keep this info up-to-date without needing to modify it manually every time something changes on the AWS side.
After the initial query, the recommender will parse this info from the AWS Pricing API once per day (configurable).

3. What kind of AWS permissions do I need to use the project?

The recommender is querying the AWS [Pricing API](https://aws.amazon.com/blogs/aws/aws-price-list-api-update-new-query-and-metadata-functions/) to keep up-to-date info
about instance types, regions and on-demand pricing.
You'll need IAM access as described here in [example 11](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-permissions-ref.html#example-policy-pe-api) of the AWS IAM docs.

If you don't use Prometheus to track spot instance pricing, you'll need to be able to access the [spot price history](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotPriceHistory.html) from the AWS API as well with your IAM user.
It means giving permission to `ec2:DescribeSpotPriceHistory`.

4. How are the spot prices determined?

The spot prices can be queried from 2 different sources. You can use Prometheus with our [spot price exporter](https://github.com/banzaicloud/spot-price-exporter) configured,
or you can use the recommender without Prometheus.
In that case the current spot prices will be queried from the AWS API and that will be the base of the recommendation.

5. What is the advantage of using Prometheus to determine spot prices?

Prometheus is becoming the de-facto monitoring solution in the cloud native world, and it includes a time series database as well.
When using the Banzai Cloud [spot price exporter](https://github.com/banzaicloud/spot-price-exporter), spot price history will be collected as time series data and
can be queried for averages, maximums and predictions.
It gives a richer picture than relying on the current spot price that can be a spike, or on a downward or upward trend.

6. How is this project different from EC2 Spot Advisor and Spot Fleet?

... built with Kubernetes clusters in mind ...
... can be used with standard auto scaling groups ...
... on demand percentage ...

7. Will this project start instances on my behalf on my cloud provider?

No, this project will never start instances. It only uses the cloud credentials to query region, instance type and pricing information.
The API response is a cluster description built from node pools of different instance types.
It is the responsibility of the user to start and manage the autoscaling groups based on the response.
The [Pipeline](https://github.com/banzaicloud/pipeline) and [Hollowtrees](https://github.com/banzaicloud/hollowtrees) projects are helping with that.

8. How does the recommender decide which instance types to include in the recommendation?

9. How are availability zones handled?

10. Is there a Google Cloud implementation?

No, but we're planning to release it in the near future.

11. There's no bid pricing on Google Cloud, what will the recommender take into account there?

12. How is this project related to [Pipeline](https://github.com/banzaicloud/pipeline)?

13. How is this project related to [Hollowtrees](https://github.com/banzaicloud/hollowtrees)

14. What happens when the spot price of one of the instance types is rising after my cluster is running?

Not the scope of this project, but check out Hollowtrees.

15. Is this project production ready?

Not yet. To make this project production ready, we need at least the following things:
 - cover the code base with unit tests
 - authentication on the API
 - API validations

16. What is on the project roadmap for the near future?

The first priority is to stabilize the API and make it production ready (see above).
Other than that, these are the things we will add soon:
 - GPU support
 - filters for instance type I/O performance, network performance
 - handle the sameSize switch to recommend similar types
 - Google Cloud Preemptible instances