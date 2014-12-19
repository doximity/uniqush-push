package rest

type PushServiceProviderResource struct {
	Id      int64             `json:"id"`
	Alias   string            `json:"alias"`
	Type    string            `json:"type"`
	Sandbox bool              `json:"sandbox"`
	Access  map[string]string `json:"access"`
}

func (serv PushServiceProviderResource) ToKeyValue() map[string]string {
	m := make(map[string]string, 4)
	m["service"] = serv.Alias
	m["pushservicetype"] = serv.Type
	if serv.Sandbox {
		m["sandbox"] = "true"
	} else {
		m["sandbox"] = "false"
	}
	for k, v := range serv.ServiceAccessKeys() {
		m[k] = v
	}
	return m
}

func (serv PushServiceProviderResource) ServiceAccessKeys() map[string]string {
	keys := make(map[string]string)
	if serv.Type == "gcm" {
		keys["projectid"] = serv.Access["projectid"]
		keys["apikey"] = serv.Access["apikey"]
	}
	if serv.Type == "apns" {
		keys["cert"] = serv.Access["cert"]
		keys["key"] = serv.Access["key"]
	}
	return keys
}
