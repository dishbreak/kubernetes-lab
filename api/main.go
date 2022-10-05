package main

import (
	"log"
	"net/http"

	"github.com/dishbreak/value-api/controller"
)

func main() {
	valueC := controller.NewValueControllerDummy()
	http.Handle("/value", valueC)

	log.Println("ready to listen")
	http.ListenAndServe(":8080", nil)
}
