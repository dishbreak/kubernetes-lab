package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v9"
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

func NewValueControllerRedis() *ValueController {
	return &ValueController{
		ValueService: &redisValueService{
			rc: redis.NewClient(&redis.Options{
				Addr:     "redis:6379",
				DB:       0,
				Password: "",
			}),
		},
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

type redisValueService struct {
	rc *redis.Client
}

const (
	valueKey = "my-value"
)

func (v *redisValueService) GetValue(ctx context.Context) (int, error) {
	status := v.rc.Get(ctx, valueKey)
	if err := status.Err(); err != nil {
		return -1, err
	}
	return status.Int()
}

func (v *redisValueService) SetValue(ctx context.Context, value int) error {
	status := v.rc.Set(ctx, valueKey, value, time.Duration(0))
	return status.Err()
}
