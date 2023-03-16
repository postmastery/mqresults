package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queue     = "results"
	interval  = time.Minute
	maxEvents = 500
	maxBytes  = 30 * 1024 * 1024 // 30MB
)

// http://www.rabbitmq.com/tutorials/tutorial-two-go.html
// http://godoc.org/github.com/streadway/amqp#example-Channel-Consume

var (
	rabbitmqAddress string
)

func main() {

	log.SetFlags(0)

	f := new(forwarder)
	f.userAgent = os.Args[0]
	f.contentType = "application/x-ndjson"

	flag.StringVar(&rabbitmqAddress, "rabbitmq-address", "amqp://guest:guest@localhost", "RabbitMQ connection URL")
	flag.StringVar(&f.url, "endpoint-url", "", "URL to post to")
	flag.StringVar(&f.authorization, "authorization", "", "Authorization header")
	flag.Parse()

	if f.url == "" {
		fmt.Fprintf(os.Stderr, "No endpoint URL specified. See %s -h for help.\n", path.Base(os.Args[0]))
		os.Exit(1)
	}

Reconnect:
	// connect to rabbitmq, wait if rabbitmq is not running yet
	var conn *amqp.Connection
	for {
		var err error
		if conn, err = amqp.Dial(rabbitmqAddress); err == nil {
			break
		}
		log.Printf("%v", err) // => "dial tcp 127.0.0.1:5672: getsockopt: connection refused"
		time.Sleep(interval)
	}
	defer conn.Close()

	//go func() {
	//    fmt.Printf("closing: %s", <-conn.NotifyClose(make(chan *amqp.Error)))
	//}()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	// ExchangeDeclare?

	// https://pkg.go.dev/github.com/rabbitmq/amqp091-go#Channel.QueueDeclarePassive
	q, err := ch.QueueDeclarePassive(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	// QueueBind? (to routing key)

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatal(err)
	}

	timer := time.NewTimer(interval)

	batch := new(batch)

	closed := conn.NotifyClose(make(chan *amqp.Error))

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	log.Printf("Listening on %s for queue %q", rabbitmqAddress, queue)

	// Read batch of messages, post them, and ack when done
ForLoop:
	for {
		select {
		case msg := <-msgs:
			// if channel was closed
			//if msg == nil {
			//	log.Printf("Channel closed")
			//	break ForLoop
			//}

			// content-type can be "application/json"
			// content-encoding can be "identity" or "gzip"

			// append record to buffer, grow if needed
			batch.Add(msg)

			// send if 500 records in buffer (limit for bigquery streaming inserts)
			if batch.count == 500 || batch.Len() >= maxBytes {
				send(f, batch, ch)
				timer.Reset(interval) // reset timer to zero
			}

		// Time interval elapsed
		case <-timer.C:
			// At least one message in buffer?
			if batch.Len() != 0 {
				send(f, batch, ch)
			} else {
				log.Printf("No messages yet") // show that agent is alive
			}
			timer.Reset(interval) // restart timer

		// RabbitMQ was stopped
		case err := <-closed:
			log.Printf("%v", err) // => Exception (320) Reason: "CONNECTION_FORCED - broker forced connection closure with reason 'shutdown'"
			if batch.Len() != 0 {
				send(f, batch, ch)
			}
			goto Reconnect

		// Interrupt/kill signal received
		case <-sigint:
			log.Printf("Signal received")
			break ForLoop
		}
	}
	if batch.Len() != 0 {
		send(f, batch, ch)
	}
}

func send(f *forwarder, batch *batch, acknowledger amqp.Acknowledger) {
	start := time.Now()
	err := f.post(batch)
	if err == nil {
		log.Printf("Posted %d events in %s", batch.count, time.Since(start).Round(time.Millisecond))
		// Acknowledge all messages received prior to (and including?) the delivery tag
		acknowledger.Ack(batch.tag, true)
	} else {
		log.Printf("Error posting events: %v", err)
		// Request redelivery of unacknowledged, delivered messages up to and
		// including the tag. see also http://www.rabbitmq.com/nack.html
		acknowledger.Nack(batch.tag, true, true)
	}
	batch.Reset()
	batch.count = 0
}
