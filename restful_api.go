package main

import (
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/uniqush/log"
	"net/http"
	"os"
)

type RestfulApi struct {
	router        *mux.Router
	loggers       []log.Logger
	legacyRestApi *RestAPI
}

func (r *RestfulApi) WithMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, h)
}

func (r *RestfulApi) AddRoute(method, route string, handlerFunc func(http.ResponseWriter, *http.Request)) {

	handler := http.HandlerFunc(handlerFunc)
	r.router.Handle(route, r.WithMiddleware(handler)).Methods(method)
}

func NewRestfulApi(loggers []log.Logger, legacyRestApi *RestAPI) *RestfulApi {
	api := new(RestfulApi)

	api.loggers = loggers

	api.router = mux.NewRouter()
	api.AddRoute("POST", "/push_service_providers", AddPushServiceProvider)
	api.AddRoute("DELETE", "/push_service_providers", RemovePushServiceProvider)
	api.AddRoute("POST", "/subscribers", api.AddDeliveryPointToService)
	api.AddRoute("DELETE", "/subscribers", RemoveDeliveryPointFromService)
	api.AddRoute("POST", "/push", PushNotification)

	api.legacyRestApi = legacyRestApi

	return api
}

func (r *RestfulApi) Run(addr string, stopChan chan<- bool) {
	http.ListenAndServe(addr, r.router)
}

func AddPushServiceProvider(w http.ResponseWriter, r *http.Request) {
}

func RemovePushServiceProvider(w http.ResponseWriter, r *http.Request) {
}

func (rest *RestfulApi) AddDeliveryPointToService(w http.ResponseWriter, r *http.Request) {
	subscription := new(SubscriptionResource)
	readJson(r, subscription)

	rest.subscribeOnLegacyApi(*subscription, w, r)

	respondJson(w, subscription)
}

func RemoveDeliveryPointFromService(w http.ResponseWriter, r *http.Request) {
}

func PushNotification(w http.ResponseWriter, r *http.Request) {
}

func (rest *RestfulApi) subscribeOnLegacyApi(subs SubscriptionResource, w http.ResponseWriter, r *http.Request) {
	logLevel := log.LOGLEVEL_INFO
	weblogger := log.NewLogger(w, "[Subscribe]", logLevel)
	logger := log.MultiLogger(weblogger, rest.legacyRestApi.loggers[LOGGER_SUB])
	rest.legacyRestApi.changeSubscription(subs.ToKeyValue(), logger, r.RemoteAddr, true)
}

type SubscriptionResource struct {
	Alias                   string `json:"alias"`
	PushServiceProviderType string `json:"push_service_provider_type"`
	ServiceAlias            string `json:"service_alias"`
	DeviceKey               string `json:"device_key"`
}

func (subs SubscriptionResource) ToKeyValue() map[string]string {
	m := make(map[string]string, 4)
	m["service"] = subs.ServiceAlias
	m["subscriber"] = subs.Alias
	m["pushservicetype"] = subs.PushServiceProviderType
	m[subs.DeviceKeyName()] = subs.DeviceKey
	return m
}

func (subs SubscriptionResource) DeviceKeyName() string {
	if subs.PushServiceProviderType == "gcm" {
		return "regid"
	} else if subs.PushServiceProviderType == "apns" {
		return "devtoken"
	}
	return ""
}

func respondJson(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json; encoding=utf8")

	e := json.NewEncoder(w)
	if err := e.Encode(obj); err != nil {
		panic(err)
	}
}

func readJson(r *http.Request, obj interface{}) {
	d := json.NewDecoder(r.Body)
	if err := d.Decode(obj); err != nil {
		panic(err)
	}
}
