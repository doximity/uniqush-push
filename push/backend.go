package push

import (
	"github.com/rafaelbandeira3/uniqush-push/mysql"
	"strconv"
)

type Backend struct {
	db mysql.MySqlPushDb
}

func NewBackend(db mysql.MySqlPushDb) *Backend {
	be := new(Backend)
	be.db = db
	return be
}

type Push struct {
	ServiceProvider *PushServiceProvider
	Notification    *Notification
	DeliveryPoints  chan *DeliveryPoint
	Results         chan *PushResult
}

func (push Push) Close() {
	close(push.DeliveryPoints)
}

func NewPush(provider *PushServiceProvider, notification *Notification) *Push {
	push := new(Push)
	push.DeliveryPoints = make(chan *DeliveryPoint)
	push.Results = make(chan *PushResult)
	push.ServiceProvider = provider
	push.Notification = notification
	return push
}

func (be Backend) Push(provider *PushServiceProvider, notification *Notification, deliveryPoints ...*DeliveryPoint) chan *PushResult {
	push := NewPush(provider, notification)
	go func() {
		for _, dp := range deliveryPoints {
			push.DeliveryPoints <- dp
		}
		push.Close()
	}()
	return be.Perform(push)
}

func (be Backend) Perform(push *Push) chan *PushResult {
	reportChannel := make(chan *PushResult)
	go be.processResults(push.Results, reportChannel)
	go GetPushServiceManager().Push(push.ServiceProvider, push.DeliveryPoints, push.Results, push.Notification)
	return reportChannel
}

func (be Backend) processResults(results chan *PushResult, report chan *PushResult) {
	for res := range results {
		if res.IsError() {
			res = be.processError(res)
		} else {
		}
		report <- res
	}
	close(report)
}

func (be Backend) processError(res *PushResult) *PushResult {
	switch res.Err.(type) {
	case *RetryError:
	case *PushServiceProviderUpdate:
	case *DeliveryPointUpdate:
		err := be.updateSubscription(res.Destination)
		if err != nil {
			res.Err = err
		}
	case *UnsubscribeUpdate:
	default:
	}
	return res
}

func (be Backend) updateSubscription(deliveryPoint *DeliveryPoint) error {
	id, err := strconv.Atoi(deliveryPoint.VolatileData["subscription_id"])
	if err == nil {
		err = be.db.UpdateSubscriptionDeviceKey(int64(id), deliveryPoint.VolatileData["regid"])
	}
	return err
}
