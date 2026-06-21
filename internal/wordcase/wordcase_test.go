package wordcase

import (
	"reflect"
	"testing"
)

func TestSplitWordsHandlesIdentifierStyles(t *testing.T) {
	got := SplitWords("HTTPServer_UserID/payment-client/SCREAMING_SNAKE")
	want := []string{"http", "server", "user", "id", "payment", "client", "screaming", "snake"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("words = %#v, want %#v", got, want)
	}
}
