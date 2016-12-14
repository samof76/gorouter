## How to troubleshoot slow requests in Cloudfoundry

In this post we will discuss the following items

1. [Request flow from client to backend](#request_flow)
1. [How to find the component causing the delay](#components)
1. [Tips/tools on how to analyze network delays](#tips)

<a name='request_flow'></a>
### Request flow from client to backend
Let's consider a CF deployment with a Load Balancer/HAProxy deployed in front of the routers. 

In the following example we will use `curl` to connect to a deployed application.

`time curl -v http://app1.app_domain.com`

Sample response:
```
GET /v2/info HTTP/1.1
Host: api.app_domain.com
User-Agent: curl/7.43.0
Accept: */*

HTTP/1.1 200 OK
Content-Type: application/json;charset=utf-8
Date: Tue, 13 Dec 2016 23:28:05 GMT
Server: nginx
X-Content-Type-Options: nosniff
X-Vcap-Request-Id: c30fad28-4972-46eb-7da6-9d07dc79b109
Content-Length: 602
hello world!
real	0m0.707s
user	0m0.005s
sys	0m0.007s
```

>Note: [`time`](https://linux.die.net/man/1/time) is a linux utility that can display the time taken to execute the command.

![Image of req flow landing page]
(images/request_lifecycle.png)

<a name='components'></a>
### How to narrow down the cause of the delay

If there is a spike in latency there are multiple points where delay can occur

1. Client network (DNS delay/firewalls)
1. Load Balancer/HAProxy
1. Gorouter
1. Backend (Application/Internal component)

For this post lets assume Load Balancer and Client network are good citizens to focus our discussion.

#### Delay caused by application

Run `cf logs app1` for [live streaming of application logs](https://docs.cloudfoundry.org/adminguide/supporting-websockets.html) 

Sample application access log

```
app1.app_domain.com - [14/12/2016:00:31:32.348 +0000] "GET /hello HTTP/1.1" 200 0 60 "-" "HTTPClient/1.0 (2.7.1, ruby 2.3.3 (2016-11-21))" "10.0.4.207:20810" "10.0.48.67:61555" x_forwarded_for:"52.3.107.171" x_forwarded_proto:"http" vcap_request_id:"01144146-1e7a-4c77-77ab-49ae3e286fe9" response_time:120.00641734 app_id:"13ee085e-bdf5-4a48-aaaf-e854a8a975df" app_index:"0" x_b3_traceid:"3595985e7c34536a" x_b3_spanid:"3595985e7c34536a" x_b3_parentspanid:"-"
```

`[14/12/2016:00:31:32.348 +0000]` - This timestamp represents when Gorouter received the request.
`response_time:120.00641734` - This represents the response time for processing the request by gorouter.

Above information suggests that application is taking long time to process the request. To investigate this line of thought further we recommend adding additional logs in application.

#### Signs to look out for when delay is not caused by gorouter
  1. Network delay from specific application or specific instance of an application only.
  1. Network delay from a subset of applications.
    - Check the network between cell(on which delayed applications are deployed) and gorouter.

#### Delay caused by router.

Let's consider the access log for the request and corresponding application logs

Access Log Example:
```
app1.app_domain.com - [14/12/2016:00:31:32.348 +0000] "GET /hello HTTP/1.1" 200 0 60 "-" "HTTPClient/1.0 (2.7.1, ruby 2.3.3 (2016-11-21))" "10.0.4.207:20810" "10.0.48.67:61555" x_forwarded_for:"52.3.107.171" x_forwarded_proto:"http" vcap_request_id:"01144146-1e7a-4c77-77ab-49ae3e286fe9" response_time:0.211734 app_id:"13ee085e-bdf5-4a48-aaaf-e854a8a975df" > app_index:"0" x_b3_traceid:"3595985e7c34536a" x_b3_spanid:"3595985e7c34536a" x_b3_parentspanid:"-"
```
Possible Application Logs:
```
app1 received request at [14/12/2016:00:32:32.348 +0000] with "vcap_request_id": "01144146-1e7a-4c77-77ab-49ae3e286fe9"
app1 finished processing req at [14/12/2016:00:32:32.500 +0000] with "vcap_request_id": "01144146-1e7a-4c77-77ab-49ae3e286fe9"
```

Let get a timeline from these logs:
- gorouter recieved the request at: [14/12/2016:00:31:32.348 +0000]
- application recieved the reqquest at: [14/12/2016:00:32:32.348 +0000]
- application finished processing at: [14/12/2016:00:32:32.500 +0000]
- gorouter finished proccessing request at : [14/12/2016:00:32:32.510 +0000]

Gorouter took close to 60 sec to send the request to router. Let's accumulate further proof by SSH'ing into a router and run the following diagnostics.

1. Use `curl` to connect to the endpoint through gorouter (this avoid the LB, client network hop)
`time curl -H "Host: app1.app_domain.com" http://IP_GOROUTER_VM:80"`

1. Use [status endpoint](https://github.com/cloudfoundry/gorouter/tree/master#the-routing-table) to fetch the backend IP and port of application and run command 
`time curl http://APPLICATION_IP:APP_PORT"`. This will give us the time for contacting the application directly (avoiding gorouter proxy). 

Use this information to deduce one of the following
- Overall network delay between gorouter and cells
- Gorouter is under heavy load so it takes long time to process requests.

#### Signs to look out for when delay is caused by gorouter

1. Routers are under heavy load.
1. Huge network delay within the IAAS. Monitor IAAS specific metrics.
1. There is an application which takes long time to process requests. This can result in the spike of go routines in router.
1. Metrics is a good way to monitor the health of router. 
    - CPU utilization
    - Latency
    - number of request/sec

<a name='tips'></a>
#### Tips/tools on how to analyze network delays.
- Consider using pingdom app on a Cloudfoundry deployment to monitor the latency and uptime metrics.
- Monitor latency of all routers(If you are using datadog , do not get the average of all routers. Try to monitor them individually on the same graph).
- Use `tcpdump` to understand more about network delays of at TCP level. 
- Use access logs for application.
- Consider enabling access logs on the Load Balancer(for eg with AWS: http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/access-log-collection.html)
- If access log is not generated for a request it means that it did not reach router. Access logs are generated for every incoming request. 
