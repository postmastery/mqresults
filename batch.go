package main

import (
	"bytes"

	amqp "github.com/rabbitmq/amqp091-go"
)

type batch struct {
	bytes.Buffer
	count int
	tag   uint64
}

// Add message to batch. Assumes message is JSON object. A newline is added after
// each message to create newline delimited JSON.
func (b *batch) Add(msg amqp.Delivery) {
	body := bytes.TrimSpace(msg.Body)
	if len(body) != 0 {
		b.Write(body)
		if body[len(body)-1] != '\n' {
			b.WriteByte('\n')
		}
		b.count++
		b.tag = msg.DeliveryTag
	}
}

// Reset batch to empty but keep allocated buffer.
func (b *batch) Reset() {
	b.Buffer.Reset()
	b.count = 0
}
