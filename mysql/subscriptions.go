package mysql

import "database/sql"
import _ "github.com/go-sql-driver/mysql"
import "fmt"

const (
	insertSubscription          = `INSERT INTO subscriptions (service_id, device_key, alias, push_service_provider_type, subscription_key, enabled) VALUES (?, ?, ?, ?, ?, ?)`
	findSubscriptionByDeviceKey = `SELECT id FROM subscriptions WHERE device_key = ?`
)

type Subscription struct {
	Id                      int64  `db:"id"`
	DeviceKey               string `db:"device_key"`
	Alias                   string `db:"alias"`
	ServiceId               int64  `db:"service_id"`
	PushServiceProviderType string `db:"push_service_provider_type"`
	SubscriptionKey         string `db:"subscription_key"`
	Enabled                 bool   `db:"enabled"`
	Service                 *Service
}

func (subs Subscription) ToKeyValue() map[string]string {
	keys := make(map[string]string)
	keys["pushservicetype"] = subs.PushServiceProviderType
	keys["subscriber"] = subs.Alias
	keys["service"] = "any"
	keys[translateSubscriptionKey(subs.PushServiceProviderType)] = subs.SubscriptionKey
	return keys
}

func (db *MySqlPushDb) UpsertSubscription(subs Subscription) (int64, error) {
	var id int64

	values := []interface{}{subs.Service.Id, subs.DeviceKey, subs.Alias, subs.PushServiceProviderType, subs.SubscriptionKey, subs.Enabled}

	err := db.db.QueryRow(findSubscriptionByDeviceKey, subs.DeviceKey).Scan(&id)

	if err == sql.ErrNoRows {
		return db.insert(insertSubscription, values...)
	}

	setters := "service_id = ?, device_key = ?, alias = ?, push_service_provider_type = ?, subscription_key = ?"
	stmt := fmt.Sprintf("UPDATE subscriptions SET %v WHERE id = ?", setters)

	_, err = db.db.Exec(stmt, values...)

	return id, err
}
