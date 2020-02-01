package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Cart represents a shopping cart.
type Cart struct {
	Cases []*Case
}

// Case represents a case of beer.
type Case struct {
	Count int
	Beer  *Beer
	Price float64
}

// Beer represents a type of beer.
type Beer struct {
	Brand  string
	Name   string
	Ounces float64
}

// NewCart initializes a new shopping cart.
func NewCart() *Cart {
	return &Cart{}
}

// AddCase adds a case of beer to the shopping cart.
func (c *Cart) AddCase(beerCase *Case) {
	c.Cases = append(c.Cases, beerCase)
}

// Subtotal calculates the subtotal of the shopping cart.
func (c *Cart) Subtotal() float64 {
	var subtotal float64
	for _, beerCase := range c.Cases {
		subtotal += beerCase.Price
	}
	return subtotal
}

// ProcessPayment sends the total to an external payment api.
func ProcessPayment(paymentServer string, total float64) ([]byte, error) {
	b, _ := json.Marshal(total)
	resp, err := http.Post(paymentServer, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("payment server error: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// PlaceOrder places the order in the warehouse.
func (o *OrderHandler) PlaceOrder(ctx context.Context, cart *Cart) error {
	o.ProcessedOrders = append(o.ProcessedOrders, cart)
	return nil
}

// OrderHandler represents a concurrent order handler.
type OrderHandler struct {
	ProcessedOrders []*Cart
	messageChan     chan interface{}
}

var logger = logrus.WithFields(logrus.Fields{
	"component": "beer",
})

// startOrderHandler listens to the message channel and handles incoming orders.
func (o *OrderHandler) startOrderHandler(ctx context.Context) {
	for {
		msg, ok := <-o.messageChan
		if !ok {
			logger.Debug("message channel closed")
			return
		}

		cart, ok := msg.(*Cart)
		if ok {
			if err := o.PlaceOrder(ctx, cart); err != nil {
				logger.WithError(err).Error("error placing order")
				continue
			}
			logger.Info("successfully placed order")
			continue
		}

		logger.WithField("msg", msg).Errorf("received invalid message on message channel")
	}
}

// Subscription represents a shopping cart.
type Subscription struct {
	cart        *Cart
	interval    time.Duration
	messageChan chan interface{}
	mu          sync.Mutex
}

// GetCart safely retrieves the subscriptions shopping cart.
func (s *Subscription) GetCart() *Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cart
}

// SetCart safely sets the subscriptions shopping cart.
func (s *Subscription) SetCart(c *Cart) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cart = c
}

// GetInterval safely retrieves the subscriptions interval.
func (s *Subscription) GetInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

// SetInterval safely sets the subscriptions interval.
func (s *Subscription) SetInterval(t time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = t
}

// startSubscriptionTimer starts a timer and fires the cart to the
// order handler when the order is ready.
func (s *Subscription) startSubscriptionTimer(ctx context.Context) {
	ticker := time.NewTicker(s.GetInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.messageChan <- s.GetCart()
		}
	}
}

// FixtureBeer creates a Beer fixture for use in test.
func FixtureBeer(brand string, name string, ounces float64) *Beer {
	return &Beer{
		Brand:  brand,
		Name:   name,
		Ounces: ounces,
	}
}

// FixtureCase creates a Case fixture for use in test.
func FixtureCase(count int, beer *Beer, price float64) *Case {
	return &Case{
		Count: count,
		Beer:  beer,
		Price: price,
	}
}

// FixtureCart creates a Cart fixture for use in test.
func FixtureCart() *Cart {
	return &Cart{
		Cases: []*Case{FixtureCase(4, FixtureBeer("Duvel", "Tripel Hop", 11.0), 14)},
	}
}

func main() {}
