# Cluster recommender

Cluster (spot) instance recommender is a building block of the [Hollowtrees](https://github.com/banzaicloud/hollowtrees) project. When the HT engine is launching, changing or reconciling existing spot instances it does based on a recommendation. When EC2 is terminating the spot instances usually it does so based on increased demand for the same instancy type - thus relaunching a new bid for the same type it's usually not filled. The spot-recommender `recommends` similar spot instance types based on the following flow

1. Spot recommender is asking the AWS EC2 API for available instances in the given datacenter/region with similar `cpu` and `memory` parameters.
2. Fetches the current spot price for the available instance types per datacenter/region
3. It computes a normalized `cost` and `stability` score and assigns to these instances 
4. The engine periodically fetches and recomputes these and puts the result in an internal cache 
5. The API fetches the results from the cache, thus no need to recompute it every time it's needed (note that every second matters as EC2 notifies 2 minutes before the instances are terminated and HT to gracefully drain and launch a new node)

### Quick start

Building the project is as simple as running a go build command. The result is a statically linked executable binary.

```
go build .
```

The following options can be configured when starting the exporter (with defaults):

```
./spot-recommender --help
Usage of ./spot-recommender:
  -cache-instance-types string
        Recommendations are cached for these instance types (comma separated list) (default "m4.xlarge,m5.xlarge,c5.xlarge")
  -listen-address string
        The address to listen on for HTTP requests. (default ":9090")
  -log-level string
        log level (default "info")
  -reevaluation-interval duration
        Time (in seconds) between reevaluating the recommendations (default 1m0s)
  -region string
        AWS region where the recommender should work (default "eu-west-1")
 ```
 
 ### API calls 
 
 ```
 curl -s -X GET "localhost:9090/api/v1/recommender/<region>?baseInstanceType=<instance-type>[&availabilityZones=<comma-separated-list-of-az>]" | jq .

curl -s "localhost:9090/api/v1/recommender/eu-west-1?baseInstanceType=m5.xlarge&availabilityZones=eu-west-1a" | jq .
```

The response of the API call is similar to the one below:

```
{
  "eu-west-1a": [
    {
      "InstanceTypeName": "c5.2xlarge",
      "CurrentPrice": "0.161300",
      "AvgPriceFor24Hours": "0.0",
      "OnDemandPrice": "0.3840000000",
      "SuggestedBidPrice": "0.3840000000",
      "CostScore": "0.000000",
      "StabilityScore": "0.0"
    },
    {
      "InstanceTypeName": "t2.xlarge",
      "CurrentPrice": "0.060500",
      "AvgPriceFor24Hours": "0.0",
      "OnDemandPrice": "0.2016000000",
      "SuggestedBidPrice": "0.2016000000",
      "CostScore": "1.000000",
      "StabilityScore": "0.0"
    },
    {
      "InstanceTypeName": "m5.xlarge",
      "CurrentPrice": "0.082600",
      "AvgPriceFor24Hours": "0.0",
      "OnDemandPrice": "0.2140000000",
      "SuggestedBidPrice": "0.2140000000",
      "CostScore": "0.780754",
      "StabilityScore": "0.0"
    },
    {
      "InstanceTypeName": "m4.xlarge",
      "CurrentPrice": "0.064200",
      "AvgPriceFor24Hours": "0.0",
      "OnDemandPrice": "0.2220000000",
      "SuggestedBidPrice": "0.2220000000",
      "CostScore": "0.963294",
      "StabilityScore": "0.0"
    }
  ]
}
```
       
### Future improvements

Spot recommender is part of the [Hollowtrees](https://github.com/banzaicloud/hollowtrees) project and as such a buldsing block of the [Pipeline](https://github.com/banzaicloud/pipeline) PaaS. HT and Pipeline has a good understanding of the workloads and the scheduled pods and their resource requirements.

* The workload it's translated to resource requirements (from the k8s scheduler queue) - and EC2 instances might not be re-launched based on a near match (we launch either smaller or larger instances based on pods resource needs and the spot market price)
* Recommender is building an `opt-in` ML model based on metrics collected by Prometheus
* Closely watches the spot market price and trigger reconcile and alert actions within a configurable threshold 
