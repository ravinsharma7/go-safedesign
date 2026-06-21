module example.com/shop

go 1.26

require (
	example.com/missing-notification v0.0.0
	example.com/payments v0.0.0
)

replace example.com/missing-notification => ../missing-notification

replace example.com/payments => ../payments
