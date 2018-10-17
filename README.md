# kafka-health
simple binary to check Apache Kafka health, useful for Kubernetes Probes

## Build
Use the Makefile like : 
```
make clean
make
```
That will generate a binary file `kafka-health`

## Usage
```
Usage of ./kafka-health:
  -broker="localhost:9092": The comma separated list of brokers in the Kafka cluster including port
  -logLevel="warning": the log level to display
  -replicaLevel=2: Replication Level required to be OK
  -topics="": REQUIRED: limit the list of topics to be checked for replication
  ```

You can supply a comma-delimited list of topics, or the application will check all the topics of the kafka server.
ex:
`./kafka-health -replicaLevel=2 -logLevel=debug -topics=userevent`

The best usage is by creating a Centreon `check` or using it as a probe for a `Kubernetes` pod.
You can set `-replicaLevel=0` to only check that the topic exist, regardless of the replication status. This is useful to ensure Kafka is running, even if the topic is not ready to server.

### Kubernetes
As an example, install the `kafka-health` binary in your Kafka Image and add the probes to your `Deployment` : 
```
livenessProbe:
      exec:
        command:
         - ./kafka-health 
         - -replicaLevel=0 
         - -topics=userevent
      initialDelaySeconds: 5
      periodSeconds: 5

readinessProbe:
      exec:
        command:
         - ./kafka-health 
         - -replicaLevel=2 
         - -topics=userevent
      initialDelaySeconds: 5
      periodSeconds: 5
```