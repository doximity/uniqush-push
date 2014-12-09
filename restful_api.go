package main

import (
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/rafaelbandeira3/uniqush-push/rest"
	"github.com/uniqush/log"
	"net/http"
	"os"
)

type RestfulApi struct {
	router        *mux.Router
	loggers       []log.Logger
	legacyRestApi *RestAPI
}

func NewRestfulApi(loggers []log.Logger, legacyRestApi *RestAPI) *RestfulApi {
	api := new(RestfulApi)

	api.loggers = loggers

	api.router = mux.NewRouter()
	api.AddRoute("POST", "/push_service_providers", api.AddPushServiceProvider)
	api.AddRoute("DELETE", "/push_service_providers/{service_alias}/{service_type}", api.RemovePushServiceProvider)
	api.AddRoute("POST", "/subscribers", api.AddDeliveryPointToService)
	api.AddRoute("DELETE", "/subscribers/{subscription_alias}", RemoveDeliveryPointFromService)
	api.AddRoute("POST", "/push_notifications", api.PushNotification)

	api.legacyRestApi = legacyRestApi

	return api
}

func (r *RestfulApi) Run(addr string, stopChan chan<- bool) {
	http.ListenAndServe(addr, r.router)
}

/* Routes */

func (rest *RestfulApi) AddPushServiceProvider(w http.ResponseWriter, r *http.Request) {
	resource := new(PushServiceProviderResource)
	readJson(r, resource)

	rest.addPushServiceProviderOnLegacy(*resource, w, r)

	respondJson(w, resource)
}

func (rest *RestfulApi) RemovePushServiceProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alias := vars["service_alias"]
	service_type := vars["service_type"]

	rest.removePushServiceProviderOnLegacy(alias, service_type, w, r)
}

func (rest *RestfulApi) AddDeliveryPointToService(w http.ResponseWriter, r *http.Request) {
	subscription := new(SubscriptionResource)
	readJson(r, subscription)

	rest.subscribeOnLegacyApi(*subscription, w, r)

	respondJson(w, subscription)
}

func RemoveDeliveryPointFromService(w http.ResponseWriter, r *http.Request) {

}

func (rest *RestfulApi) PushNotification(w http.ResponseWriter, r *http.Request) {
	pushNotification := new(PushNotificationResource)
	readJson(r, pushNotification)

	rest.pushNotificationOnLegacy(*pushNotification, w, r)

	respondJson(w, pushNotification)
}

/* Legacy integration */

func (rest *RestfulApi) pushNotificationOnLegacy(pushNotification PushNotificationResource, w http.ResponseWriter, r *http.Request) {
	logLevel := log.LOGLEVEL_INFO
	weblogger := log.NewLogger(w, "[Push]", logLevel)
	logger := log.MultiLogger(weblogger, rest.legacyRestApi.loggers[LOGGER_PUSH])
	rest.legacyRestApi.pushNotification(UniquePushNotificationId(), pushNotification.ToKeyValue(), make(map[string][]string, 0), logger, r.RemoteAddr)
}

func (rest *RestfulApi) subscribeOnLegacyApi(subs SubscriptionResource, w http.ResponseWriter, r *http.Request) {
	logLevel := log.LOGLEVEL_INFO
	weblogger := log.NewLogger(w, "[Subscribe]", logLevel)
	logger := log.MultiLogger(weblogger, rest.legacyRestApi.loggers[LOGGER_SUB])
	rest.legacyRestApi.changeSubscription(subs.ToKeyValue(), logger, r.RemoteAddr, true)
}

func (rest *RestfulApi) addPushServiceProviderOnLegacy(serv PushServiceProviderResource, w http.ResponseWriter, r *http.Request) {
	logLevel := log.LOGLEVEL_INFO
	weblogger := log.NewLogger(w, "[PushServiceProvider]", logLevel)
	logger := log.MultiLogger(weblogger, rest.legacyRestApi.loggers[LOGGER_SUB])
	rest.legacyRestApi.changePushServiceProvider(serv.ToKeyValue(), logger, r.RemoteAddr, true)
}

func (rest *RestfulApi) removePushServiceProviderOnLegacy(alias, service_type string, w http.ResponseWriter, r *http.Request) {

}

/* Utils */

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

func (r *RestfulApi) WithMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, h)
}

func (r *RestfulApi) AddRoute(method, route string, handlerFunc func(http.ResponseWriter, *http.Request)) {

	handler := http.HandlerFunc(handlerFunc)
	r.router.Handle(route, r.WithMiddleware(handler)).Methods(method)
}
