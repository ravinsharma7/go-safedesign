package paymentclient

func Await(ch <-chan string) string {
	select {
	case v := <-ch:
		return v
	default:
		return ""
	}
}
