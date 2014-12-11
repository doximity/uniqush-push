package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/rafaelbandeira3/uniqush-push/mysql"
	"github.com/rafaelbandeira3/uniqush-push/push"
	. "github.com/rafaelbandeira3/uniqush-push/rest"
	"github.com/uniqush/log"
	"net/http"
	"os"
)

type RestfulApi struct {
	router        *mux.Router
	loggers       []log.Logger
	legacyRestApi *RestAPI
	db            MySqlPushDb
}

func NewRestfulApi(db MySqlPushDb, loggers []log.Logger, legacyRestApi *RestAPI) *RestfulApi {
	api := new(RestfulApi)

	api.loggers = loggers

	api.router = mux.NewRouter()
	api.AddRoute("POST", "/push_service_providers", api.AddPushServiceProvider)
	api.AddRoute("DELETE", "/push_service_providers/{service_alias}/{service_type}", api.RemovePushServiceProvider)
	api.AddRoute("POST", "/subscribers", api.AddDeliveryPointToService)
	api.AddRoute("DELETE", "/subscribers/{subscription_alias}", RemoveDeliveryPointFromService)
	api.AddRoute("POST", "/push_notifications", api.PushNotification)

	api.legacyRestApi = legacyRestApi
	api.db = db

	return api
}

func (r *RestfulApi) Run(addr string, stopChan chan<- bool) {
	http.ListenAndServe(addr, r.router)
}

/* Routes */

func (rest *RestfulApi) AddPushServiceProvider(w http.ResponseWriter, r *http.Request) {
	resource := new(PushServiceProviderResource)
	readJson(r, resource)

	service, err := rest.db.FindServiceByAlias(resource.Alias)
	if err != nil {
		_, err = rest.db.InsertService(resource.Alias)

		if err != nil {
			w.WriteHeader(422)
			jsonError := JsonError{Error: fmt.Sprintf("Can't create service %", resource.Alias), GoError: err.Error()}
			respondJson(w, jsonError)
			return
		}

		service, err = rest.db.FindServiceByAlias(resource.Alias)
		if err != nil {
			w.WriteHeader(422)
			jsonError := JsonError{Error: fmt.Sprintf("Can't find service %v", resource.Alias), GoError: err.Error()}
			respondJson(w, jsonError)
			return
		}
	}

	accessKeys := make([]string, 0, 2)
	for _, v := range resource.ServiceAccessKeys() {
		accessKeys = append(accessKeys, v)
	}
	id, err := rest.db.InsertPushServiceProvider(service.Id, resource.Type, accessKeys)
	if err != nil {
		w.WriteHeader(422)
		jsonError := JsonError{Error: fmt.Sprintf("Can't create push service provider for %v", resource.Alias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	resource.Id = id

	respondJson(w, resource)
}

func (rest *RestfulApi) RemovePushServiceProvider(w http.ResponseWriter, r *http.Request) {
	//TODO this is not working
	vars := mux.Vars(r)
	alias := vars["service_alias"]
	service_type := vars["service_type"]

	rest.removePushServiceProviderOnLegacy(alias, service_type, w, r)
}

func (rest *RestfulApi) AddDeliveryPointToService(w http.ResponseWriter, r *http.Request) {
	resource := new(SubscriptionResource)
	readJson(r, resource)

	service, err := rest.db.FindServiceByAlias(resource.ServiceAlias)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		jsonError := JsonError{Error: fmt.Sprintf("Service %v not found", resource.ServiceAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	id, err := rest.db.InsertSubscription(service.Id, resource.Alias, resource.PushServiceProviderType, resource.DeviceKey)
	if err != nil {
		w.WriteHeader(422)
		jsonError := JsonError{Error: "Can't create subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	resource.Id = id

	respondJson(w, resource)
}

func RemoveDeliveryPointFromService(w http.ResponseWriter, r *http.Request) {

}

func (rest *RestfulApi) PushNotification(w http.ResponseWriter, r *http.Request) {
	resource := new(PushNotificationResource)
	readJson(r, resource)

	service, err := rest.db.FindServiceByAlias(resource.ServiceAlias)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		jsonError := JsonError{Error: fmt.Sprintf("Service %v not found", resource.ServiceAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	subscriptions, err := rest.db.FindAllSubscriptionsByAlias(resource.SubscriberAlias)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError := JsonError{Error: fmt.Sprintf("Can't load subscriptions for %v", resource.SubscriberAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	err = rest.db.FindPushServiceProvidersFor(&service)
	if err != nil {
		w.WriteHeader(422)
		jsonError := JsonError{Error: fmt.Sprintf("Can't find push service providers for %v", resource.ServiceAlias), GoError: err.Error()}
		respondJson(w, jsonError)
	}

	psm := push.GetPushServiceManager()
	notification := buildNotification(resource)
	for _, subs := range subscriptions {
		pushServiceProvider, found := service.ProviderOfType(subs.PushServiceProviderType)

		if !found {
			continue
		}

		psp, err := psm.BuildPushServiceProviderFromMap(pushServiceProvider.ToKeyValue())
		if err != nil {
			w.WriteHeader(422)
			jsonError := JsonError{Error: fmt.Sprintf("Can't initialize push service provider %v for %v", pushServiceProvider.Id, service.Alias), GoError: err.Error()}
			respondJson(w, jsonError)
			return
		}

		deliveryPoint, err := psm.BuildDeliveryPointFromMap(subs.ToKeyValue())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			jsonError := JsonError{Error: fmt.Sprintf("Can't initialize delivery point for subscription %v", subs.Id), GoError: err.Error()}
			respondJson(w, jsonError)
			return
		}

		deliveryPoints := make(chan *push.DeliveryPoint)
		pushResults := make(chan *push.PushResult)

		go func() {
			for _ = range pushResults {
			}
		}()
		go func() { psm.Push(psp, deliveryPoints, pushResults, notification) }()
		deliveryPoints <- deliveryPoint
		close(deliveryPoints)
	}

	respondJson(w, resource)
}

func buildNotification(resource *PushNotificationResource) *push.Notification {
	notie := push.NewEmptyNotification()
	for k, v := range resource.Content {
		notie.Data[k] = v
	}
	return notie
}

/* Legacy integration */

func (rest *RestfulApi) pushNotificationOnLegacy(pushNotification PushNotificationResource, w http.ResponseWriter, r *http.Request) {
	rest.legacyRestApi.pushNotification(UniquePushNotificationId(), pushNotification.ToKeyValue(), make(map[string][]string, 0), rest.legacyRestApi.loggers[LOGGER_PUSH], r.RemoteAddr)
}

func (rest *RestfulApi) subscribeOnLegacyApi(subs SubscriptionResource, w http.ResponseWriter, r *http.Request) {
	rest.legacyRestApi.changeSubscription(subs.ToKeyValue(), rest.legacyRestApi.loggers[LOGGER_SUB], r.RemoteAddr, true)
}

func (rest *RestfulApi) addPushServiceProviderOnLegacy(serv PushServiceProviderResource, w http.ResponseWriter, r *http.Request) {
	rest.legacyRestApi.changePushServiceProvider(serv.ToKeyValue(), rest.legacyRestApi.loggers[LOGGER_ADDPSP], r.RemoteAddr, true)
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
