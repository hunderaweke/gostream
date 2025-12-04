package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ() (*RabbitMQ, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		return nil, fmt.Errorf("error connecting to rabbitmq: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error getting connection channel: %v", err)
	}
	_, err = ch.QueueDeclare(
		"video_encoding_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error declaring video encoding queue: %v", err)
	}
	return &RabbitMQ{Conn: conn, Channel: ch}, nil
}

type VideoMessage struct {
	VideoID  string `json:"video_id,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

func (r *RabbitMQ) PublishVideoUploaded(videoId, filePath string) error {
	msg := VideoMessage{
		VideoID:  videoId,
		FilePath: filePath,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error parsing video message: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = r.Channel.PublishWithContext(ctx, "", "video_encoding_queue", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return fmt.Errorf("error publishing the event %v", err)
	}
	return nil
}
func (r *RabbitMQ) Close() {
	r.Channel.Close()
	r.Conn.Close()
}
