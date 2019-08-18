package main

import "fmt"
import "net/http"
import "encoding/json"
import "os"

func handler(w http.ResponseWriter, r *http.Request) {
	var v interface{}
	d := json.NewDecoder(r.Body)
	err := d.Decode(&v)
	if err != nil {
		fmt.Println("error: ", err)
		w.WriteHeader(500)
	} else {
		fmt.Printf("%#v\n", v)
		w.WriteHeader(200)
	}
}
func main() {
	apiRoute := "/"
	if v := os.Getenv("API_ROUTE"); len(v) > 0 {
		apiRoute = v
	}
	http.HandleFunc(apiRoute, handler)
	err := http.ListenAndServe(":9000", nil)
	fmt.Println("error: ", err)
}
