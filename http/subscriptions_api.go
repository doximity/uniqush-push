package http

import "net/http"
import "github.com/rafaelbandeira3/uniqush-push/rest"
import "github.com/rafaelbandeira3/uniqush-push/mysql"
import (
	"fmt"
)

type SubscriptionsApi struct {
	db mysql.MySqlPushDb
}

func (api *SubscriptionsApi) AddDeliveryPointToService(w http.ResponseWriter, r *http.Request) {
	resource := new(rest.SubscriptionResource)
	readJson(r, resource)

	service, err := api.db.FindServiceByAlias(resource.ServiceAlias)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		jsonError := rest.JsonError{Error: fmt.Sprintf("Service %v not found", resource.ServiceAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	id, err := api.db.UpsertSubscriptionFor(service, resource.Alias, resource.PushServiceProviderType, resource.DeviceKey)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't create subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	resource.Id = id

	respondJson(w, resource)
}

func (api *SubscriptionsApi) RemoveDeliveryPointFromService(w http.ResponseWriter, r *http.Request) {
	resource := new(rest.SubscriptionResource)
	readJson(r, resource)

	err := api.db.DeleteSubscriptionByDeviceKey(resource.Alias, resource.DeviceKey)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't destroy subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	w.WriteHeader(204)
}
