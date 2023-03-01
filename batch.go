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

func (b *batch) Add(msg amqp.Delivery) {
	body := bytes.TrimSpace(msg.Body)
	if len(body) != 0 {
		b.Write(body)
		// Newline delimited JSON
		if body[len(body)-1] != '\n' {
			b.WriteByte('\n')
		}
		b.count++
		b.tag = msg.DeliveryTag
	}
}
