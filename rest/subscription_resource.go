package rest

type SubscriptionResource struct {
	Id                      int64  `json:"id"`
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
