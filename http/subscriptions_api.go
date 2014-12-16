package http

import "net/http"
import "github.com/rafaelbandeira3/uniqush-push/rest"
import "github.com/rafaelbandeira3/uniqush-push/mysql"
import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"strconv"
)

type SubscriptionsApi struct {
	db mysql.MySqlPushDb
}

func (api *SubscriptionsApi) AddDeliveryPointToService(w http.ResponseWriter, r *http.Request) {
	resource := MustGetSubscriptionResource(w, r)
	if resource == nil {
		return
	}

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

	err = api.UpdateResourceState(id, resource)
	if err != nil {
		w.WriteHeader(500)
		jsonError := rest.JsonError{Error: "Updated but can't select subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	respondJson(w, resource)
}

func (api *SubscriptionsApi) RemoveDeliveryPointFromService(w http.ResponseWriter, r *http.Request) {
	resource := MustGetSubscriptionResource(w, r)
	if resource == nil {
		return
	}

	err := api.db.DeleteSubscriptionByDeviceKey(resource.Alias, resource.DeviceKey)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't destroy subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	w.WriteHeader(204)
}

func (api *SubscriptionsApi) UpdateDeliveryPoint(w http.ResponseWriter, r *http.Request) {
	resource := MustGetSubscriptionResource(w, r)
	vars := mux.Vars(r)

	id, _ := strconv.Atoi(vars["id"])
	err := api.db.UpdateSubscription(int64(id), resource.Enabled)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't update subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	err = api.UpdateResourceState(int64(id), resource)
	if err != nil {
		w.WriteHeader(500)
		jsonError := rest.JsonError{Error: "Updated but can't select subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	respondJson(w, resource)
}

func (api *SubscriptionsApi) UpdateResourceState(id int64, resource *rest.SubscriptionResource) error {
	subs, err := api.db.FindSubscription(id)
	if err != nil {
		return err
	}
	resource.Id = subs.Id
	resource.Alias = subs.Alias
	resource.PushServiceProviderType = subs.PushServiceProviderType
	resource.DeviceKey = subs.DeviceKey
	resource.Enabled = subs.Enabled
	resource.ServiceAlias = subs.Service.Alias
	return err
}

func GetSubscriptionResource(r *http.Request) (*rest.SubscriptionResource, error) {
	resource := new(rest.SubscriptionResource)
	var parsed map[string]interface{}
	readJson(r, &parsed)

	fmt.Println(parsed)

	if parsed["enabled"] != nil {
		enabled, ok := parsed["enabled"].(bool)
		if !ok {
			enabledInt, ok := parsed["enabled"].(float64)

			if !ok {
				return resource, fmt.Errorf("Invalid value for 'enabled'")
			}
			enabled = enabledInt != 0
		}
		parsed["enabled"] = enabled
	}

	raw, err := json.Marshal(parsed)
	if err != nil {
		return resource, err
	}

	err = json.Unmarshal(raw, resource)
	if err != nil {
		return resource, err
	}

	return resource, nil
}

func MustGetSubscriptionResource(w http.ResponseWriter, r *http.Request) *rest.SubscriptionResource {
	resource, err := GetSubscriptionResource(r)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't parse subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return nil
	}
	return resource
}
