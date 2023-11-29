package ping

import (
	"fmt"
	"net/http"

	cowsay "github.com/Code-Hex/Neo-cowsay/v2"
)

//do:api raw method=GET path=/ping
func Hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	content := fmt.Sprintf("Hello, %s!", name)
	cow, _ := cowsay.New(cowsay.Random())
	say, _ := cow.Say(content)
	fmt.Fprint(w, say)
}
