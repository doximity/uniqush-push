package http

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/rafaelbandeira3/uniqush-push/mysql"
	. "github.com/rafaelbandeira3/uniqush-push/rest"
	"net/http"
	"os"
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

func (r *RestfulApi) WithMiddleware(h http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, h)
}

func (r *RestfulApi) AddRoute(method, route string, handlerFunc func(http.ResponseWriter, *http.Request)) {
	handler := http.HandlerFunc(handlerFunc)
	r.router.Handle(route, r.WithMiddleware(handler)).Methods(method)
}
