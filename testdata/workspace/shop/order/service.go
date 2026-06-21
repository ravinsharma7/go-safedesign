package order

import (
	notification "example.com/missing-notification/notify"
	"example.com/payments/gateway"
	"example.com/shop/paymentclient"
)

type OrderRequest struct {
	OrderID string
}

type PaymentGateway interface {
	Charge(id string) bool
}

func PlaceOrder(id string) {
	ch := make(chan string, 1)
	go paymentclient.Authorize(id)
	defer close(ch)
	ch <- id
	<-ch
	notification.Notify(id)
	gateway.Charge(id)
}
