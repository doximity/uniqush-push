package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/rafaelbandeira3/uniqush-push/mysql"
	"github.com/rafaelbandeira3/uniqush-push/push"
	. "github.com/rafaelbandeira3/uniqush-push/rest"
	"net/http"
	"os"
)

type RestfulApi struct {
	router *mux.Router
	db     MySqlPushDb
}

func NewRestfulApi(db MySqlPushDb) *RestfulApi {
	api := new(RestfulApi)

	api.router = mux.NewRouter()
	api.AddRoute("POST", "/push_service_providers", api.AddPushServiceProvider)
	api.AddRoute("DELETE", "/push_service_providers/{service_alias}/{service_type}", api.RemovePushServiceProvider)
	api.AddRoute("POST", "/subscribers", api.AddDeliveryPointToService)
	api.AddRoute("DELETE", "/subscribers/{subscription_alias}", RemoveDeliveryPointFromService)
	api.AddRoute("POST", "/push_notifications", api.PushNotification)

	api.db = db

	return api
}

func (r *RestfulApi) Run(addr string, stopChan chan<- bool) {
	err := http.ListenAndServe(addr, r.router)
	if err != nil {
		fmt.Println(err)
	}
	stopChan <- true
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
	var jsonError JsonError
	finished := make(chan bool)
	resource := new(PushNotificationResource)
	readJson(r, resource)

	service, err := rest.db.FindServiceByAlias(resource.ServiceAlias)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		jsonError = JsonError{Error: fmt.Sprintf("Service %v not found", resource.ServiceAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	subscriptions, err := rest.db.FindAllSubscriptionsByAliasAndServiceId(resource.SubscriberAlias, service.Id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError = JsonError{Error: fmt.Sprintf("Can't load subscriptions for %v", resource.SubscriberAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	err = rest.db.FindPushServiceProvidersFor(&service)
	if err != nil {
		w.WriteHeader(422)
		jsonError = JsonError{Error: fmt.Sprintf("Can't find push service providers for %v", resource.ServiceAlias), GoError: err.Error()}
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
			jsonError = JsonError{Error: fmt.Sprintf("Can't initialize push service provider %v for %v", pushServiceProvider.Id, service.Alias), GoError: err.Error()}
			respondJson(w, jsonError)
			return
		}

		deliveryPoint, err := psm.BuildDeliveryPointFromMap(subs.ToKeyValue())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			jsonError = JsonError{Error: fmt.Sprintf("Can't initialize delivery point for subscription %v", subs.Id), GoError: err.Error()}
			respondJson(w, jsonError)
			return
		}

		deliveryPoints := make(chan *push.DeliveryPoint)
		pushResults := make(chan *push.PushResult)

		go func() {
			jsonError := JsonError{}
			success := true
			for result := range pushResults {
				if result.IsError() {
					if success {
						success = false
					}
					jsonError.AddError(result.Error())
				}
			}
			finished <- success
		}()
		go func() { psm.Push(psp, deliveryPoints, pushResults, notification) }()
		deliveryPoints <- deliveryPoint
		close(deliveryPoints)
	}

	if <-finished {
		respondJson(w, resource)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		respondJson(w, jsonError)
	}
}

func buildNotification(resource *PushNotificationResource) *push.Notification {
	notie := push.NewEmptyNotification()
	for k, v := range resource.Content {
		notie.Data[k] = v
	}
	return notie
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
