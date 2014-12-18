package http

import (
	"fmt"
	"github.com/rafaelbandeira3/uniqush-push/push"
	. "github.com/rafaelbandeira3/uniqush-push/rest"
	"net/http"
	"strconv"
)

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
	for idx, subs := range subscriptions {
		last := idx+1 == len(subscriptions)
		pushServiceProvider, found := service.ProviderOfType(subs.PushServiceProviderType)

		if !found {
			if last {
				go func() { finished <- true }()
			}
			rest.log("Can't push to %v. No %v provider for %v.", subs.Alias, subs.PushServiceProviderType, service.Alias)
			continue
		}

		content := resource.ContentForProvider(subs.PushServiceProviderType)
		if len(content) == 0 {
			if last {
				go func() { finished <- true }()
			}
			rest.log("No content for %v, ignoring", subs.PushServiceProviderType)
			continue
		}
		notification := buildNotification(content)

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

func buildNotification(content map[string]interface{}) *push.Notification {
	notie := push.NewEmptyNotification()
	for k, v := range content {
		if v != nil {
			switch v.(type) {
			case string:
				notie.Data[k] = v.(string)
			case int64:
				str := strconv.Itoa(int(v.(int64)))
				notie.Data[k] = str
			}
		}
	}
	return notie
}
