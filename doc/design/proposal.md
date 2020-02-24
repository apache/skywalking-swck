## Summary
The SkyWalking Cloud on Kubernetes is proposed in order to:

 * Managing and Monitoring 
 * Scaling backend cluster capacity up and down
 * Changing backend cluster configuration
 * Injecting configuration into the target cluster.
 * Securing traffic between target clusters and backend cluster, or between backend cluster with TLS certificate
## Motivation
If the user of SkyWalking decided to deploy it into Kubernetes, there’re some critical challenges for them. 

First of them is the complex of deployment, it doesn’t only mean the OAP server and storage cluster, but also include configuring target cluster to send data to backend. Then they might struggle to keep all of them reliable. The size of the data transferred is very big and the cost of data stored is very high. The user usually faces some problems, for instance, OAP server stuck, Elasticsearch cluster GC rate sharply increases, the system load of some OAP instances is much more than others, and etc.

With the help of CRDs and the Controller, we can figure out the above problems and give users a more pleasing experience when using SWCK.

## Proposal

### Production Design

I proposed two crucial components for SWCK, backend operator and target injector. The first one intends to solve the problems of the backend operation, and another focus on simplifying the configuration of the target cluster.

They should be built as two separate binary/image, then are installed according to user’s requirements. 

### Backend Operator

The operator might be a GO application that manages and monitors other components, for example, OAP pods, storage pods(ES, MySQL, and etc.), ingress/entry and configuration.

It should be capable of HA, performance, and scalability.

It should also have the following capabilities:

 * Defining CRDs for provisioning and configuring
 * Provisioning backend automatically
 * Splitting OAP instances according to their type(L1/L2), improving the ratio of them.
 * Performance tuning of OAP and storage.
 * Updating configuration dynamically, irrespectively it’s dynamic or not.
 * Upgrading mirror version seamlessly.
 * Health checking and failure recovery
 * Collecting and analyzing metrics and logs, abnormal detection
 * Horizontal scaling and scheduling tuning.
 * Loadbalancing input gPRC stream and GraphQL querying. 
 * Supporting externally hosted storage service.
 * Securing traffic

The above items should be accomplished in several versions/releases. The developer should sort the priority of them and grind the design.

### Target injector

The injector can inject agent lib and configuration into the target cluster automatically, enable/disable distributed tracing according to labels marked on resources or namespace.

It also integrates backend with service mesh platform, for example, Istio.

It should be a GO application and a GO lib to be invoked by swctl to generate pod YAMLs manually.

## Technology Selection

 * Development Language: GO
 * Operator dev tool: TBD
 * Building tool: Make(Docker for windows)
 * Installation: Helm3 chart
 * Repository: github.com/apache/skywalking-swck
 * CI: Github action
