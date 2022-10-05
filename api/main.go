package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dishbreak/value-api/controller"
)

const (
	useRedis = "USE_REDIS_BACKEND"
)

func main() {
	var valueC *controller.ValueController
	if os.Getenv(useRedis) != "" {
		valueC = controller.NewValueControllerRedis()
	} else {
		valueC = controller.NewValueControllerDummy()
	}
	http.Handle("/value", valueC)

	log.Println("ready to listen")
	http.ListenAndServe(":8080", nil)
}
