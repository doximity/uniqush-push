package mysql

import "database/sql"
import _ "github.com/go-sql-driver/mysql"

type MySqlPushDb struct {
	db *sql.DB
}

const (
	insertSubscription        = `INSERT INTO subscriptions (service_id, alias, push_service_provider_type, device_token) VALUES (?, ?, ?, ?)`
	insertService             = `INSERT INTO services (alias) VALUES (?)`
	insertPushServiceProvider = `INSERT INTO push_service_providers (service_id, type) VALUES (?, ?)`
	insertApnsAccessKeys      = `INSERT INTO apns_access_keys (push_service_provider_id, certificate, key) VALUES (?, ?, ?)`
	insertGcmAccessKeys       = `INSERT INTO gcm_access_keys (push_service_provider_id, project, key) VALUES (?, ?)`

	selectSubscription = `SELECT * FROM subscriptions WHERE alias = ?`
	selectService      = `SELECT * FROM services WHERE alias = ?`
)

func NewMySqlPushDb(url string) (MySqlPushDb, error) {
	var instance MySqlPushDb
	db, err := sql.Open("mysql", url)
	if err == nil {
		err = db.Ping()
	}

	instance.db = db

	return instance, err
}

type Service struct {
	Id    int    `db:"id"`
	Alias string `db:"alias"`
}

type PushServiceProvider struct {
	Id        int    `db:"id"`
	Type      string `db:"type"`
	ServiceId int    `db:"service_id"`
	Service   *Service
}

type Subscription struct {
	Id                      int    `db:"id"`
	Alias                   string `db:"alias"`
	ServiceId               int    `db:"service_id"`
	PushServiceProviderType string `db:"push_service_provider_type"`
	DeviceToken             string `db:"device_token"`
	Service                 *Service
}

func (db *MySqlPushDb) FindServiceByAlias(alias string) (Service, error) {
	var service Service

	err := db.db.QueryRow(selectService, alias).Scan(&service.Id, &service.Alias)

	return service, err
}

func (db *MySqlPushDb) FindSubscriptionByAlias(alias string) (Subscription, error) {
	var subscription Subscription

	err := db.db.QueryRow(selectSubscription, alias).Scan(&subscription)

	return subscription, err
}

func (db *MySqlPushDb) InsertSubscription(serviceId int, alias string, serviceType string, deviceKey string) (int64, error) {
	res, err := db.db.Exec(insertSubscription, serviceId, alias, serviceType, deviceKey)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, nil
}

func (db *MySqlPushDb) Close() {
	db.db.Close()
}

//func (db *MySqlPushDb) RemovePushServiceProviderFromService(service string, push_service_provider *PushServiceProvider) error {
//	return nil
//}
//
//func (db *MySqlPushDb) AddPushServiceProviderToService(serviceAlias string, push_service_provider *PushServiceProvider) error {
//	var err error
//	var service Service
//	return nil
//}
//
//func (db *MySqlPushDb) ModifyPushServiceProvider(psp *PushServiceProvider) error {
//	return nil
//}
//func (db *MySqlPushDb) AddDeliveryPointToService(service string, subscriber string, delivery_point *DeliveryPoint) (*PushServiceProvider, error) {
//	return nil, nil
//}
//func (db *MySqlPushDb) RemoveDeliveryPointFromService(service string, subscriber string, delivery_point *DeliveryPoint) error {
//	return nil
//}
//func (db *MySqlPushDb) ModifyDeliveryPoint(dp *DeliveryPoint) error {
//	return nil
//}
//func (db *MySqlPushDb) GetPushServiceProviderDeliveryPointPairs(service string, subscriber string) ([]PushServiceProviderDeliveryPointPair, error) {
//	return nil, nil
//}
//func (db *MySqlPushDb) FlushCache() error {
//	return nil
//}
