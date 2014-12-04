package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/uniqush/log"
	"net/http"
	"os"
)

type RestfulApi struct {
	router  *mux.Router
	loggers []Logger
}

func (r *RestfulApi) WithMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, h)
}

func (r *RestfulApi) AddRoute(method, route string, handlerFunc func(http.ResponseWriter, *http.Request)) {

	handler := http.HandlerFunc(handlerFunc)
	r.router.Handle(route, r.WithMiddleware(handler)).Methods(method)
}

func NewRestfulApi(loggers []Logger) *RestfulApi {
	api := new(RestfulApi)

	api.loggers = loggers

	api.router = mux.NewRouter()
	api.AddRoute("POST", "/push_service_providers", AddPushServiceProvider)
	api.AddRoute("DELETE", "/push_service_providers", RemovePushServiceProvider)
	api.AddRoute("POST", "/subscribers", AddDeliveryPointToService)
	api.AddRoute("DELETE", "/subscribers", RemoveDeliveryPointFromService)
	api.AddRoute("POST", "/push", PushNotification)

	return api
}

func (r *RestfulApi) Run(addr string, stopChan chan<- bool) {
	http.ListenAndServe(addr, r.router)
}

func AddPushServiceProvider(w http.ResponseWriter, r *http.Request) {
}
func RemovePushServiceProvider(w http.ResponseWriter, r *http.Request) {
}
func AddDeliveryPointToService(w http.ResponseWriter, r *http.Request) {
}
func RemoveDeliveryPointFromService(w http.ResponseWriter, r *http.Request) {
}
func PushNotification(w http.ResponseWriter, r *http.Request) {
}
