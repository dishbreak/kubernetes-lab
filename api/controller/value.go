package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type ValueService interface {
	SetValue(context.Context, int) error
	GetValue(context.Context) (int, error)
}

type ValueController struct {
	ValueService
}

func (v *ValueController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		val, err := v.GetValue(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("get failed: %s", err.Error())
			return
		}
		fmt.Fprintf(w, "%d", val)
	case http.MethodPost:
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		val, err := strconv.Atoi(string(buf))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = v.SetValue(ctx, val)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("set failed: %s", err.Error())
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func NewValueControllerDummy() *ValueController {
	return &ValueController{
		ValueService: &dummyValueService{},
	}
}

type dummyValueService struct {
	val int
}

func (v *dummyValueService) GetValue(ctx context.Context) (int, error) {
	return v.val, nil
}

func (v *dummyValueService) SetValue(ctx context.Context, value int) error {
	v.val = value
	return nil
}
