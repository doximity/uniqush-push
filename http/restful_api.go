package http

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/rafaelbandeira3/uniqush-push/mysql"
	"github.com/rafaelbandeira3/uniqush-push/push"
	. "github.com/rafaelbandeira3/uniqush-push/rest"
	"net/http"
	"os"
	"strconv"
)

type RestfulApi struct {
	router *mux.Router
	db     MySqlPushDb
	SubscriptionsApi
}

func NewRestfulApi(db MySqlPushDb) *RestfulApi {
	api := new(RestfulApi)

	api.router = mux.NewRouter()
	api.AddRoute("POST", "/push_service_providers", api.AddPushServiceProvider)
	api.AddRoute("DELETE", "/push_service_providers/{service_alias}/{service_type}", api.RemovePushServiceProvider)
	api.AddRoute("POST", "/subscribers", api.UpsertDeliveryPoint)
	api.AddRoute("POST", "/push_notifications", api.PushNotification)

	api.db = db
	api.SubscriptionsApi.db = db

	return api
}

func (r *RestfulApi) Run(addr string, stopChan chan<- bool) {
	err := http.ListenAndServe(addr, r.router)
	if err != nil {
		fmt.Println(err)
	}
	stopChan <- true
}

func (r RestfulApi) log(str string, st ...interface{}) {
	fmt.Println(fmt.Sprintf(str, st...))
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

	subscriptions, err := rest.db.FindAllSubscriptionsByAliasAndServiceId(resource.SubscriptionAlias, service.Id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError = JsonError{Error: fmt.Sprintf("Can't load subscriptions for %v", resource.SubscriptionAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	if len(subscriptions) == 0 {
		w.WriteHeader(http.StatusNotFound)
		jsonError = JsonError{Error: fmt.Sprintf("No subscriptions for %v in %v", resource.SubscriptionAlias, service.Alias)}
		respondJson(w, jsonError)
		return
	}

	fmt.Println("Subs", len(subscriptions))

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
			rest.log("Can't push to %v. No %v provider for %v.", subs.Alias, subs.PushServiceProviderType, service.Alias)
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
		deliveryPoint.VolatileData["subscription_id"] = strconv.Itoa(int(subs.Id))

		be := push.NewBackend(rest.db)
		results := be.Push(psp, notification, deliveryPoint)

		go func() {
			success := true
			for result := range results {
				if result.IsError() {
					success = false
					jsonError.AddError(result.Error())
					continue
				}
			}
			finished <- success
		}()
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

func (r *RestfulApi) WithMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, h)
}

func (r *RestfulApi) AddRoute(method, route string, handlerFunc func(http.ResponseWriter, *http.Request)) {

	handler := http.HandlerFunc(handlerFunc)
	r.router.Handle(route, r.WithMiddleware(handler)).Methods(method)
}
