package rest

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"
)

type PushNotificationResource struct {
	Content           map[string]interface{} `json:"content"`
	ServiceAlias      string                 `json:"service_alias"`
	SubscriptionAlias string                 `json:"subscription_alias"`
}

func UniquePushNotificationId() string {
	// extracted from restapi randomUniqId
	var d [16]byte
	io.ReadFull(rand.Reader, d[:])
	return fmt.Sprintf("%x-%v", time.Now().Unix(), base64.URLEncoding.EncodeToString(d[:]))
}

func (pn PushNotificationResource) ToKeyValue() map[string]string {
	m := make(map[string]string, len(pn.Content)+2)
	m["service"] = pn.ServiceAlias
	m["subscriber"] = pn.SubscriptionAlias
	for k, v := range pn.Content {
		m[k] = v.(string)
	}
	return m
}

func (pn PushNotificationResource) ContentForProvider(providerType string) map[string]interface{} {
	return pn.Content[providerType].(map[string]interface{})
}
