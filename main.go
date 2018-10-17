package main

import (
	"os"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/namsral/flag"
	"github.com/sirupsen/logrus"
)

var (
	logLevel     = flag.String("logLevel", logrus.WarnLevel.String(), "the log level to display")
	broker       = flag.String("broker", "localhost:9092", "The comma separated list of brokers in the Kafka cluster including port")
	topics       = flag.String("topics", "", "REQUIRED: limit the list of topics to be checked for replication")
	replicaLevel = flag.Int("replicaLevel", 2, "Replication Level required to be OK")
	version      = "no version set"
)

func main() {
	flag.Parse()

	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.JSONFormatter{})
	myLogLevel, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		myLogLevel = logrus.WarnLevel
	}
	logrus.SetLevel(myLogLevel)

	// Output to stdout instead of the default stderr
	logrus.SetOutput(os.Stdout)

	logrus.WithFields(logrus.Fields{
		"version": version,
		"brokers": *broker}).Info("starting app")

	// split brokers and topics
	brokersList := strings.Split(*broker, ",")
	topicsList := strings.Split(*topics, ",")

	// init (custom) config, enable errors and notifications
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Version = sarama.V1_0_0_0

	// init consumer
	client, err := sarama.NewClient(brokersList, config)
	if err != nil {
		logrus.Fatalf("Failed to start sarama client: %s", err)
	}
	defer client.Close()

	// get the list of topics
	// if none provided, get the list from Kafka
	if len(topicsList) == 1 && topicsList[0] == "" {
		topicsList, err = client.Topics()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err,
			}).Fatal("Error Listing Topics")
		}
	}

	// debug the list of topics to check
	logrus.WithFields(logrus.Fields{
		"topics": topicsList,
		"len":    len(topicsList),
	}).Debug("topic list generated")

	// parse all topics for replication
	for _, topic := range topicsList {
		partitions, err := client.Partitions(topic)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"err":   err,
				"topic": topic,
			}).Fatal("Error Listing Partitions")
		}
		// parse each partition and get replication status
		for _, partition := range partitions {
			replicas, err := client.Replicas(topic, partition)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"topic":     topic,
					"partition": partition,
				}).Fatal("Error listing partitions")
			}

			logrus.Debug("found topic", "topic", topic, "partition", partition, "replica", replicas)

			// exit with error if replication not OK
			if *replicaLevel > 0 && len(replicas) != *replicaLevel {
				logrus.WithFields(logrus.Fields{
					"topic":     topic,
					"partition": partition,
				}).Fatalf("topics %s:%d is not fully replicated", topic, partition)
			}
		}
	}
}
