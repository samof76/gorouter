## How to troubleshoot slow requests in Cloudfoundry

In this post we will discuss the following items

1. Request flow from client to backend
1. How to find the component causing the delay
1. Common network delay problems we have encountered
  1. Network delay across all applications
  1. Network delay from specific application or specific instance of an application.
  1. Application took longer than 15min to handle requests.
1. Tips/tools on how to analyze network delay in Gorouter.

#### Request flow from client to backend
To simplify we will consider production scenario where you have Load Balancer/HAProxy deployed in front of routers. In the following we will use `Curl` to connect to an application deployed in Cloudfoundry.

`time curl -v http://app1.app_domain.com`
Sample response:
> GET /v2/info HTTP/1.1
> Host: api.app_domain.com
> User-Agent: curl/7.43.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: application/json;charset=utf-8
< Date: Tue, 13 Dec 2016 23:28:05 GMT
< Server: nginx
< X-Content-Type-Options: nosniff
< X-Vcap-Request-Id: c30fad28-4972-46eb-7da6-9d07dc79b109
< Content-Length: 602
< hello world!
real	0m0.707s
user	0m0.005s
sys	0m0.007s

>Note: `time` is simple utility in linux that can provide the time required to execute the command.



(image: images/Draw.jpeg)

#### How to find the point of delay in request flow

If there is a spike in latency there are multiple points where latency can occur
1. Client network (DNS delay/firewalls)
1. Load Balancer
1. Gorouter
1. Backend (Application/Internal component)

In this case lets consider Load Balancer, Client network are good citizens and there is a delay with either Gorouter or application. How do we find out who is the villain (Muhaha :D) ?
For the following cases we will gorouter access logs as our source of information.

1. Delay caused by application

Run `cf logs app1` for live streaming of application logs (considering websocket is configured properly)

Sample application access log:

```
app1.app_domain.com - [14/12/2016:00:31:32.348 +0000] "GET /hello HTTP/1.1" 200 0 60 "-" "HTTPClient/1.0 (2.7.1, ruby 2.3.3 (2016-11-21))" "10.0.4.207:20810" "10.0.48.67:61555" x_forwarded_for:"52.3.107.171" x_forwarded_proto:"http" vcap_request_id:"01144146-1e7a-4c77-77ab-49ae3e286fe9" response_time:120.00641734 app_id:"13ee085e-bdf5-4a48-aaaf-e854a8a975df" app_index:"0" x_b3_traceid:"3595985e7c34536a" x_b3_spanid:"3595985e7c34536a" x_b3_parentspanid:"-"
```

1. `[14/12/2016:00:31:32.348 +0000]` - This represents the time Gorouter received the request.
1. `response_time:120.00641734` - This represents the response time for processing the request.

Above information suggests that application is taking long time to process the request. To investigate this line of thought further we recommend adding additional logs in application.

#### Signs for delay caused by application we have encountered
1. Network delay from specific application or specific instance of an application only.
1. Network delay from set of applications.
  - Check the network between cell(on which delayed applications are deployed) and gorouter.

1. Delay caused by router.

For this use case consider the access log for a request and corresponding application logs
```
app1.app_domain.com - [14/12/2016:00:31:32.348 +0000] "GET /hello HTTP/1.1" 200 0 60 "-" "HTTPClient/1.0 (2.7.1, ruby 2.3.3 (2016-11-21))" "10.0.4.207:20810" "10.0.48.67:61555" x_forwarded_for:"52.3.107.171" x_forwarded_proto:"http" vcap_request_id:"01144146-1e7a-4c77-77ab-49ae3e286fe9" response_time:0.211734 app_id:"13ee085e-bdf5-4a48-aaaf-e854a8a975df" app_index:"0" x_b3_traceid:"3595985e7c34536a" x_b3_spanid:"3595985e7c34536a" x_b3_parentspanid:"-"

app1 received req at [14/12/2016:00:32:32.348 +0000]
app1 finished processing req at [14/12/2016:00:32:32.500 +0000]
```

Let get a timeline from these logs:
- gorouter rec the req at: [14/12/2016:00:31:32.348 +0000]
- app rec the req at: [14/12/2016:00:32:32.348 +0000]
- app finished processing at: [14/12/2016:00:32:32.500 +0000]
- gorouter finished the req at : [14/12/2016:00:32:32.510 +0000]

Gorouter took close to 60 sec to send the req to router. What went wrong? How can I check if something is not right with gorouter
 - Metrics is a good way to monitor the health of router. for eg: CPU utilization, number of go routines , latency, num of req /sec

To further diagnose lets SSH into the router and get more data.
- Lets `Curl` the endpoint through gorouter (this avoid the LB, client network hop)
`time curl -H "Host: app1.app_domain.com" http://IP_GOROUTER_VM:80"`
- Use status endpoint details to fetch the backend IP and port of application and try this
`time curl http://APPLICATION_IP:APP_PORT"`. This will give us the time for contacting the application directly (avoiding gorouter proxy)  

Use this information to deduce whether one of the following
- overall network delay between gorouter and cells
- gorouter is under heavy load(use metrics to monitor this)

#### Signs for delay caused by router
  1. Network delays across all applications. Overall latency of gorouter is high. Common reasons why gorouter can get into this state
    - Routers are under heavy load.
    - Huge network delay within the IAAS. Monitor IAAS specific metrics.
    - There is an application taking long time to process requests which results in the spike of go routines(each request in router spins up new go routine) in router.

#### Tips/tools on how to analyze network delay in Gorouter.
- Consider using pingdom app on a Cloudfoundry deployment to monitor the latency and uptime metrics.
- Monitor latency of all routers(If you are datadog , do not get the aggregate of all routers. Try to get monitor them individually on the same graph).
- Use of `tcpdump` to understand more about network delays of IAAS.
- Use access logs for application.
- Consider enabling access logs on the Load Balancer(for eg with AWS: http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/access-log-collection.html)
