// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package secret

import (
	"context"

	"fmt"

	logutil "github.com/boz/go-logutil"

	"github.com/boz/kcache"

	"github.com/boz/kcache/client"

	"github.com/boz/kcache/filter"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
) // This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

var (
	ErrInvalidType = fmt.Errorf("invalid type")
	adapter        = _adapter{}
)

type Event interface {
	Type() kcache.EventType
	Resource() *v1.Secret
}

type CacheReader interface {
	Get(ns string, name string) (*v1.Secret, error)
	List() ([]*v1.Secret, error)
}

type CacheController interface {
	Cache() CacheReader
	Ready() <-chan struct{}
}

type Subscription interface {
	CacheController
	Events() <-chan Event
	Close()
	Done() <-chan struct{}
}

type Publisher interface {
	Subscribe() Subscription
	SubscribeWithFilter(filter.Filter) FilterSubscription
	Clone() Controller
	CloneWithFilter(filter.Filter) FilterController
}

type Controller interface {
	CacheController
	Publisher
	Done() <-chan struct{}
	Close()
}

type FilterSubscription interface {
	Subscription
	Refilter(filter.Filter)
}

type FilterController interface {
	Controller
	Refilter(filter.Filter)
}

type Handler interface {
	OnInitialize([]*v1.Secret)
	OnCreate(*v1.Secret)
	OnUpdate(*v1.Secret)
	OnDelete(*v1.Secret)
}

type HandlerBuilder interface {
	OnInitialize(func([]*v1.Secret)) HandlerBuilder
	OnCreate(func(*v1.Secret)) HandlerBuilder
	OnUpdate(func(*v1.Secret)) HandlerBuilder
	OnDelete(func(*v1.Secret)) HandlerBuilder
	Create() Handler
}

type _adapter struct{}

func (_adapter) adaptObject(obj metav1.Object) (*v1.Secret, error) {
	if obj, ok := obj.(*v1.Secret); ok {
		return obj, nil
	}
	return nil, ErrInvalidType
}

func (a _adapter) adaptList(objs []metav1.Object) ([]*v1.Secret, error) {
	var ret []*v1.Secret
	for _, orig := range objs {
		adapted, err := a.adaptObject(orig)
		if err != nil {
			continue
		}
		ret = append(ret, adapted)
	}
	return ret, nil
}

func newCache(parent kcache.CacheReader) CacheReader {
	return &cache{parent}
}

type cache struct {
	parent kcache.CacheReader
}

func (c *cache) Get(ns string, name string) (*v1.Secret, error) {
	obj, err := c.parent.Get(ns, name)
	switch {
	case err != nil:
		return nil, err
	case obj == nil:
		return nil, nil
	default:
		return adapter.adaptObject(obj)
	}
}

func (c *cache) List() ([]*v1.Secret, error) {
	objs, err := c.parent.List()
	if err != nil {
		return nil, err
	}
	return adapter.adaptList(objs)
}

type event struct {
	etype    kcache.EventType
	resource *v1.Secret
}

func wrapEvent(evt kcache.Event) (Event, error) {
	obj, err := adapter.adaptObject(evt.Resource())
	if err != nil {
		return nil, err
	}
	return event{evt.Type(), obj}, nil
}

func (e event) Type() kcache.EventType {
	return e.etype
}

func (e event) Resource() *v1.Secret {
	return e.resource
}

type subscription struct {
	parent kcache.Subscription
	cache  CacheReader
	outch  chan Event
}

func newSubscription(parent kcache.Subscription) *subscription {
	s := &subscription{
		parent: parent,
		cache:  newCache(parent.Cache()),
		outch:  make(chan Event, kcache.EventBufsiz),
	}
	go s.run()
	return s
}

func (s *subscription) run() {
	defer close(s.outch)
	for pevt := range s.parent.Events() {
		evt, err := wrapEvent(pevt)
		if err != nil {
			continue
		}
		select {
		case s.outch <- evt:
		default:
		}
	}
}

func (s *subscription) Cache() CacheReader {
	return s.cache
}

func (s *subscription) Ready() <-chan struct{} {
	return s.parent.Ready()
}

func (s *subscription) Events() <-chan Event {
	return s.outch
}

func (s *subscription) Close() {
	s.parent.Close()
}

func (s *subscription) Done() <-chan struct{} {
	return s.parent.Done()
}

func NewController(ctx context.Context, log logutil.Log, cs kubernetes.Interface, ns string) (Controller, error) {
	client := NewClient(cs, ns)
	return BuildController(ctx, log, client)
}

func BuildController(ctx context.Context, log logutil.Log, client client.Client) (Controller, error) {
	parent, err := kcache.NewController(ctx, log, client)
	if err != nil {
		return nil, err
	}
	return newController(parent), nil
}

func newController(parent kcache.Controller) *controller {
	return &controller{parent, newCache(parent.Cache())}
}

type controller struct {
	parent kcache.Controller
	cache  CacheReader
}

func (c *controller) Close() {
	c.parent.Close()
}

func (c *controller) Ready() <-chan struct{} {
	return c.parent.Ready()
}

func (c *controller) Done() <-chan struct{} {
	return c.parent.Done()
}

func (c *controller) Cache() CacheReader {
	return c.cache
}

func (c *controller) Subscribe() Subscription {
	return newSubscription(c.parent.Subscribe())
}

func (c *controller) SubscribeWithFilter(f filter.Filter) FilterSubscription {
	return newFilterSubscription(c.parent.SubscribeWithFilter(f))
}

func (c *controller) Clone() Controller {
	return newController(c.parent.Clone())
}

func (c *controller) CloneWithFilter(f filter.Filter) FilterController {
	return newFilterController(c.parent.CloneWithFilter(f))
}

type filterController struct {
	controller
	filterParent kcache.FilterController
}

func newFilterController(parent kcache.FilterController) FilterController {
	return &filterController{
		controller:   controller{parent, newCache(parent.Cache())},
		filterParent: parent,
	}
}

func (c *filterController) Refilter(f filter.Filter) {
	c.filterParent.Refilter(f)
}

type filterSubscription struct {
	subscription
	filterParent kcache.FilterSubscription
}

func newFilterSubscription(parent kcache.FilterSubscription) FilterSubscription {
	return &filterSubscription{
		subscription: *newSubscription(parent),
		filterParent: parent,
	}
}

func (s *filterSubscription) Refilter(f filter.Filter) {
	s.filterParent.Refilter(f)
}

func NewMonitor(publisher Publisher, handler Handler) kcache.Monitor {
	phandler := kcache.NewHandlerBuilder().
		OnInitialize(func(objs []metav1.Object) {
			aobjs, _ := adapter.adaptList(objs)
			handler.OnInitialize(aobjs)
		}).
		OnCreate(func(obj metav1.Object) {
			aobj, _ := adapter.adaptObject(obj)
			handler.OnCreate(aobj)
		}).
		OnUpdate(func(obj metav1.Object) {
			aobj, _ := adapter.adaptObject(obj)
			handler.OnUpdate(aobj)
		}).
		OnDelete(func(obj metav1.Object) {
			aobj, _ := adapter.adaptObject(obj)
			handler.OnDelete(aobj)
		}).Create()

	switch obj := publisher.(type) {
	case *controller:
		return kcache.NewMonitor(obj.parent, phandler)
	case *filterController:
		return kcache.NewMonitor(obj.parent, phandler)
	default:
		panic(fmt.Sprintf("Invalid publisher type: %T is not a *controller", publisher))
	}
}

func NewHandlerBuilder() HandlerBuilder {
	return &handlerBuilder{}
}

type handler struct {
	onInitialize func([]*v1.Secret)
	onCreate     func(*v1.Secret)
	onUpdate     func(*v1.Secret)
	onDelete     func(*v1.Secret)
}

type handlerBuilder handler

func (hb *handlerBuilder) OnInitialize(fn func([]*v1.Secret)) HandlerBuilder {
	hb.onInitialize = fn
	return hb
}

func (hb *handlerBuilder) OnCreate(fn func(*v1.Secret)) HandlerBuilder {
	hb.onCreate = fn
	return hb
}

func (hb *handlerBuilder) OnUpdate(fn func(*v1.Secret)) HandlerBuilder {
	hb.onUpdate = fn
	return hb
}

func (hb *handlerBuilder) OnDelete(fn func(*v1.Secret)) HandlerBuilder {
	hb.onDelete = fn
	return hb
}

func (hb *handlerBuilder) Create() Handler {
	return handler(*hb)
}

func (h handler) OnInitialize(objs []*v1.Secret) {
	if h.onInitialize != nil {
		h.onInitialize(objs)
	}
}

func (h handler) OnCreate(obj *v1.Secret) {
	if h.onCreate != nil {
		h.onCreate(obj)
	}
}

func (h handler) OnUpdate(obj *v1.Secret) {
	if h.onUpdate != nil {
		h.onUpdate(obj)
	}
}

func (h handler) OnDelete(obj *v1.Secret) {
	if h.onDelete != nil {
		h.onDelete(obj)
	}
}
