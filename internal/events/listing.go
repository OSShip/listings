package events

import (
	"context"

	"github.com/OSShip/utils/kafka"
)

type Publisher struct {
	producer *kafka.Producer
}

func New(brokers string) *Publisher {
	return &Publisher{producer: kafka.NewProducer(brokers, "listing.events")}
}

func (p *Publisher) Close() {
	p.producer.Close()
}

func (p *Publisher) PublishListingCreated(ctx context.Context, listingID string) error {
	return p.producer.Publish(ctx, "listing.created", map[string]string{"listing_id": listingID})
}

func (p *Publisher) PublishListingUpdated(ctx context.Context, listingID string) error {
	return p.producer.Publish(ctx, "listing.updated", map[string]string{"listing_id": listingID})
}
