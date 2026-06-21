package paymentclient

import "example.com/payments/gateway"

func Authorize(id string) {
	gateway.Charge(id)
}
