package http

import "net/http"
import "github.com/rafaelbandeira3/uniqush-push/rest"
import "github.com/rafaelbandeira3/uniqush-push/mysql"
import (
	"encoding/json"
	"fmt"
)

type SubscriptionsApi struct {
	db mysql.MySqlPushDb
}

func MapToSubscription(resource *rest.SubscriptionResource) *mysql.Subscription {
	subs := new(mysql.Subscription)
	subs.Id = resource.Id
	subs.DeviceKey = resource.DeviceKey
	subs.Alias = resource.Alias
	subs.PushServiceProviderType = resource.PushServiceProviderType
	subs.SubscriptionKey = resource.SubscriptionKey
	subs.Enabled = resource.Enabled
	return subs
}

func (api *SubscriptionsApi) UpsertDeliveryPoint(w http.ResponseWriter, r *http.Request) {
	resource := MustGetSubscriptionResource(w, r)

	serv, err := api.db.FindServiceByAlias(resource.ServiceAlias)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		jsonError := rest.JsonError{Error: fmt.Sprintf("Service %v not found", resource.ServiceAlias), GoError: err.Error()}
		respondJson(w, jsonError)
		return
	}

	subs := MapToSubscription(resource)
	subs.Service = &serv
	id, err := api.db.UpsertSubscription(*subs)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't update subscription", GoError: err.Error()}
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

func (api *SubscriptionsApi) UpdateResourceState(id int64, resource *rest.SubscriptionResource) error {
	subs, err := api.db.FindSubscription(id)
	if err != nil {
		return err
	}
	resource.Id = subs.Id
	resource.DeviceKey = subs.DeviceKey
	resource.Alias = subs.Alias
	resource.PushServiceProviderType = subs.PushServiceProviderType
	resource.SubscriptionKey = subs.SubscriptionKey
	resource.Enabled = subs.Enabled
	resource.ServiceAlias = subs.Service.Alias
	return err
}

func ParseSubscriptionResource(r *http.Request) (*rest.SubscriptionResource, error) {
	resource := new(rest.SubscriptionResource)
	var parsed map[string]interface{}
	readJson(r, &parsed)

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
	resource, err := ParseSubscriptionResource(r)
	if err != nil {
		w.WriteHeader(422)
		jsonError := rest.JsonError{Error: "Can't parse subscription", GoError: err.Error()}
		respondJson(w, jsonError)
		return nil
	}
	return resource
}
