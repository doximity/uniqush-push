package mysql

import "database/sql"
import _ "github.com/go-sql-driver/mysql"
import "fmt"

type MySqlPushDb struct {
	db *sql.DB
}

const (
	insertSubscription        = `INSERT INTO subscriptions (service_id, alias, push_service_provider_type, device_key) VALUES (?, ?, ?, ?)`
	insertService             = `INSERT INTO services (alias) VALUES (?)`
	insertPushServiceProvider = `INSERT INTO push_service_providers (service_id, type) VALUES (?, ?)`
	insertApnsAccessKeys      = `INSERT INTO apns_access_keys (push_service_provider_id, certificate_pem, key_pem) VALUES (?, ?, ?)`
	insertGcmAccessKeys       = `INSERT INTO gcm_access_keys (push_service_provider_id, project, api_key) VALUES (?, ?, ?)`

	selectSubscriptions         = `SELECT * FROM subscriptions WHERE alias = ? AND service_id = ?`
	findSubscriptionByDeviceKey = `SELECT id FROM subscriptions WHERE device_key = ?`
	selectService               = `SELECT * FROM services WHERE alias = ?`
	selectPushServiceProviders  = `SELECT psp.id, psp.type, psp.service_id, gcm.project, gcm.api_key, apns.certificate_pem, apns.key_pem FROM push_service_providers AS psp LEFT JOIN gcm_access_keys AS gcm ON gcm.push_service_provider_id = psp.id LEFT JOIN apns_access_keys AS apns ON apns.push_service_provider_id = psp.id WHERE psp.service_id = ?`

	selectSubscriptionsForPushServiceProvider = `SELECT * FROM subscriptions WHERE service_id = ? AND push_service_provider_type = ?`
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
	Id        int64  `db:"id"`
	Alias     string `db:"alias"`
	Providers []PushServiceProvider
}

func (serv Service) ProviderOfType(providerType string) (PushServiceProvider, bool) {
	for _, psp := range serv.Providers {
		if psp.Type == providerType {
			return psp, true
		}
	}
	return PushServiceProvider{}, false
}

type PushServiceProvider struct {
	Id         int64  `db:"id"`
	Type       string `db:"type"`
	ServiceId  int64  `db:"service_id"`
	Service    *Service
	AccessKeys map[string]string
}

func (psp PushServiceProvider) ToKeyValue() map[string]string {
	m := make(map[string]string, 4)
	m["service"] = psp.Service.Alias
	m["pushservicetype"] = psp.Type
	for k, v := range psp.AccessKeys {
		m[translateAccessKey(k)] = v
	}
	return m
}

func (psp PushServiceProvider) String() string {
	str := fmt.Sprintf("Id=%v Type=%v", psp.Id, psp.Type)
	for k, v := range psp.AccessKeys {
		str = fmt.Sprintf("%v AccessKey.%v=%v", str, k, v[:25])
	}
	return str
}

func translateAccessKey(column string) string {
	switch column {
	case "project":
		return "projectid"
	case "api_key":
		return "apikey"
	case "certificate_pem":
		return "cert"
	case "key_pem":
		return "key"
	default:
		return ""
	}
}

func translateDeviceKey(providerType string) string {
	switch providerType {
	case "gcm":
		return "regid"
	case "apns":
		return "devtoken"
	default:
		return ""
	}
}

type Subscription struct {
	Id                      int64  `db:"id"`
	Alias                   string `db:"alias"`
	ServiceId               int64  `db:"service_id"`
	PushServiceProviderType string `db:"push_service_provider_type"`
	DeviceKey               string `db:"device_key"`
	Enabled                 bool   `db:"enabled"`
	Service                 *Service
}

func (subs Subscription) ToKeyValue() map[string]string {
	keys := make(map[string]string)
	keys["pushservicetype"] = subs.PushServiceProviderType
	keys["subscriber"] = subs.Alias
	keys["service"] = "any"
	keys[translateDeviceKey(subs.PushServiceProviderType)] = subs.DeviceKey
	return keys
}

func (db *MySqlPushDb) FindPushServiceProvidersFor(service *Service) error {
	results := make([]PushServiceProvider, 0, 10)

	rows, err := db.db.Query(selectPushServiceProviders, service.Id)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		psp := new(PushServiceProvider)
		args := []interface{}{&psp.Id, &psp.Type, &psp.ServiceId}
		accessKeys := []*sql.NullString{new(sql.NullString), new(sql.NullString), new(sql.NullString), new(sql.NullString)}
		rawAccessKeys := map[string]*sql.NullString{
			"project":         accessKeys[0],
			"api_key":         accessKeys[1],
			"certificate_pem": accessKeys[2],
			"key_pem":         accessKeys[3]}
		for _, v := range accessKeys {
			args = append(args, v)
		}
		err := rows.Scan(args...)
		if err != nil {
			return err
		}
		psp.Service = service
		psp.AccessKeys = make(map[string]string, 2)
		for k, v := range rawAccessKeys {
			if v.Valid {
				psp.AccessKeys[k] = v.String
			}
		}
		results = append(results, *psp)
	}

	err = rows.Err()

	if err == nil {
		service.Providers = results
	}

	return err
}

func (db *MySqlPushDb) FindServiceByAlias(alias string) (Service, error) {
	var service Service

	err := db.db.QueryRow(selectService, alias).Scan(&service.Id, &service.Alias)

	return service, err
}

func (db *MySqlPushDb) UpdateSubscriptionDeviceKey(id int64, deviceKey string) error {
	ret, err := db.db.Exec("UPDATE subscriptions SET device_key = ? WHERE id = ?", id, deviceKey)
	log("UpdateSubscription(%v) %v", id, ret)
	return err
}

func log(str string, st ...interface{}) {
	fmt.Println("[db]", fmt.Sprintf(str, st...))
}

type SqlResult interface {
	Scan(dest ...interface{}) error
}

func ScanSubscription(scanner SqlResult) (*Subscription, error) {
	subs := new(Subscription)
	err := scanner.Scan(&subs.Id, &subs.ServiceId, &subs.Alias, &subs.PushServiceProviderType, &subs.DeviceKey, &subs.Enabled)

	return subs, err
}

func ScanService(scanner SqlResult) (*Service, error) {
	serv := new(Service)
	err := scanner.Scan(&serv.Id, &serv.Alias)

	return serv, err
}

func (db *MySqlPushDb) FindSubscription(id int64) (*Subscription, error) {
	subs, err := ScanSubscription(db.db.QueryRow("SELECT * FROM subscriptions WHERE id = ?", id))
	if err == nil {
		service, err := ScanService(db.db.QueryRow("SELECT * FROM services WHERE id = ?", subs.ServiceId))
		if err == nil {
			subs.Service = service
		}
	}
	return subs, err
}

func (db *MySqlPushDb) UpdateSubscription(id int64, enabled bool) error {
	_, err := db.db.Exec("UPDATE subscriptions SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

func (db *MySqlPushDb) DeleteSubscriptionByDeviceKey(alias string, deviceKey string) error {
	_, err := db.db.Exec("DELETE FROM subscriptions WHERE alias = ? AND device_key = ?", alias, deviceKey)
	return err
}

func (db *MySqlPushDb) FindAllSubscriptionsByAliasAndServiceId(alias string, serviceId int64) ([]Subscription, error) {
	results := make([]Subscription, 0, 10)

	rows, err := db.db.Query(selectSubscriptions, alias, serviceId)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		subs, err := ScanSubscription(rows)
		if err != nil {
			return results, err
		}

		results = append(results, *subs)
	}

	return results, err
}

func (db *MySqlPushDb) insert(stm string, values ...interface{}) (int64, error) {
	res, err := db.db.Exec(stm, values...)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, nil
}

func (db *MySqlPushDb) UpsertSubscriptionFor(service Service, alias string, serviceType string, deviceKey string) (int64, error) {
	var id int64
	err := db.db.QueryRow(findSubscriptionByDeviceKey, deviceKey).Scan(&id)

	if err == sql.ErrNoRows {
		return db.insert(insertSubscription, service.Id, alias, serviceType, deviceKey)
	}

	return id, err
}

func (db *MySqlPushDb) InsertService(alias string) (int64, error) {
	return db.insert(insertService, alias)
}

func (db *MySqlPushDb) InsertPushServiceProvider(serviceId int64, serviceType string, accessKeys []string) (int64, error) {
	id, err := db.insert(insertPushServiceProvider, serviceId, serviceType)
	if err != nil {
		return 0, err
	}

	var insertAccessKeys string
	if serviceType == "apns" {
		insertAccessKeys = insertApnsAccessKeys
	}
	if serviceType == "gcm" {
		insertAccessKeys = insertGcmAccessKeys
	}

	args := make([]interface{}, 3)
	args[0] = id
	args[1] = accessKeys[0]
	args[2] = accessKeys[1]
	_, err = db.insert(insertAccessKeys, args...)

	return id, err
}

func (db *MySqlPushDb) Close() {
	db.db.Close()
}
