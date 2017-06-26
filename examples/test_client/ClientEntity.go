package main

import (
	"fmt"

	"reflect"

	"math/rand"
	"time"

	"github.com/xiaonanln/goTimer"
	. "github.com/xiaonanln/goworld/common"
	"github.com/xiaonanln/goworld/gwlog"
	"github.com/xiaonanln/typeconv"
)

const (
	AVERAGE_DO_SOMETHING_INTERVAL = time.Second * 15
)

type ClientAttrs map[string]interface{}

func (attrs ClientAttrs) HasKey(key string) bool {
	_, ok := attrs[key]
	return ok
}

type ClientEntity struct {
	owner    *ClientBot
	TypeName string
	ID       EntityID

	Attrs     ClientAttrs
	IsPlayer  bool
	destroyed bool
	timers    map[*timer.Timer]bool

	currentThing          string
	currentThingStartTime time.Time
	currentTimeoutTimer   *timer.Timer
}

func newClientEntity(owner *ClientBot, typeName string, entityid EntityID, isPlayer bool) *ClientEntity {
	e := &ClientEntity{
		owner:    owner,
		TypeName: typeName,
		ID:       entityid,
		Attrs:    make(ClientAttrs),
		IsPlayer: isPlayer,
		timers:   map[*timer.Timer]bool{},
	}

	e.OnCreated()
	return e
}

func (e *ClientEntity) String() string {
	return fmt.Sprintf("%s<%s>", e.TypeName, e.ID)
}

func (e *ClientEntity) Destroy() {
	if e.destroyed {
		return
	}
	e.destroyed = true
}

func (e *ClientEntity) OnCreated() {
	if !quiet {
		gwlog.Debug("%s.OnCreated, IsPlayer=%v", e, e.IsPlayer)
	}
	if !e.IsPlayer {
		return
	}

	if e.TypeName == "Avatar" {
		e.onAvatarCreated()
	} else if e.TypeName == "Account" {
		e.onAccountCreated()
	}

}

func (e *ClientEntity) onAvatarCreated() {
}

func (e *ClientEntity) doSomethingLater() {
	randomDelay := time.Duration(rand.Int63n(int64(AVERAGE_DO_SOMETHING_INTERVAL * 2)))
	e.AddCallback(randomDelay, func() {
		e.doSomething()
	})
}

func (e *ClientEntity) AddCallback(d time.Duration, callback timer.CallbackFunc) *timer.Timer {
	var t *timer.Timer
	t = timer.AddCallback(d, func() {
		e.owner.Lock()
		defer e.owner.Unlock()

		if !e.timers[t] {
			// timer is cancelled
			return
		}

		delete(e.timers, t)
		if e.destroyed {
			return
		}

		callback()
	})
	e.timers[t] = true
	return t
}

func (e *ClientEntity) AddTimer(d time.Duration, callback timer.CallbackFunc) *timer.Timer {
	var t *timer.Timer
	t = timer.AddTimer(d, func() {
		e.owner.Lock()
		defer e.owner.Unlock()

		if !e.timers[t] {
			// timer is cancelled
			return
		}

		if e.destroyed {
			t.Cancel()
			delete(e.timers, t)
			return
		}

		callback()
	})
	e.timers[t] = true
	return t
}

func (e *ClientEntity) CancelTimer(t *timer.Timer) {
	t.Cancel()
	delete(e.timers, t)
}

type _Something struct {
	Method  string
	Weight  int
	Timeout time.Duration
}

var (
	DO_THINGS = []*_Something{
		{"DoEnterRandomSpace", 100, time.Minute},
	}
)

func (e *ClientEntity) doSomething() {
	if e.currentThing != "" {
		gwlog.Panicf("%s can not do something while doing %s", e, e.currentThing)
	}

	thing := e.chooseThingByWeight()
	e.currentThing = thing.Method
	e.currentThingStartTime = time.Now()
	e.currentTimeoutTimer = e.AddCallback(thing.Timeout, func() {
		gwlog.Warn("[%s] %s %s TIMEOUT !!!", time.Now(), e, thing)

		e.currentThing = ""
		e.currentThingStartTime = time.Time{}
		e.currentTimeoutTimer = nil

		e.doSomethingLater()
	})

	gwlog.Debug("[%s] %s STARTS %s", e.currentThingStartTime, e, e.currentThing)
	reflect.ValueOf(e).MethodByName(thing.Method).Call(nil)
}

func (e *ClientEntity) notifyThingDone(thing string) {
	if e.currentThing == thing {
		now := time.Now()
		gwlog.Info("[%s] %s FINISHES %s, TAKES %s", now, e, thing, now.Sub(e.currentThingStartTime))

		e.currentThing = ""
		e.currentThingStartTime = time.Time{}
		e.currentTimeoutTimer.Cancel()
		e.currentTimeoutTimer = nil

		e.doSomethingLater()
	}
}

func (e *ClientEntity) chooseThingByWeight() *_Something {
	return DO_THINGS[0]
}

func (e *ClientEntity) DoEnterRandomSpace() {
	curSpaceKind := 0
	if e.owner.currentSpace != nil {
		curSpaceKind = e.owner.currentSpace.Kind
	}

	spaceKind := SPACE_KIND_MIN + rand.Intn(SPACE_KIND_MAX-SPACE_KIND_MIN+1)
	for spaceKind == curSpaceKind {
		if spaceKind == SPACE_KIND_MAX {
			spaceKind = SPACE_KIND_MIN
		} else {
			spaceKind += 1
		}
	}

	e.CallServer("EnterSpace", spaceKind)
}

func (e *ClientEntity) onAccountCreated() {
	timer.AddCallback(0, func() {

		username := e.owner.username()
		password := e.owner.password()
		e.CallServer("Login", username, password)
	})
}

func (e *ClientEntity) CallServer(method string, args ...interface{}) {
	e.owner.CallServer(e.ID, method, args)
}

func (e *ClientEntity) applyAttrChange(path []string, key string, val interface{}) {
	attr := e.findAttrByPath(path)
	var rootkey string
	if len(path) > 0 {
		rootkey = path[len(path)-1]
	} else {
		rootkey = key
	}

	if _, ok := val.(map[interface{}]interface{}); ok {
		val = typeconv.MapStringAnything(val)
	}
	attr[key] = val

	callbackFuncName := "OnAttrChange_" + rootkey
	callbackMethod := reflect.ValueOf(e).MethodByName(callbackFuncName)
	if !callbackMethod.IsValid() {
		gwlog.Warn("Attribute change callback of %s is not defined (%s)", rootkey, callbackFuncName)
		return
	}
	callbackMethod.Call([]reflect.Value{}) // call the attr change callback func
}

func (e *ClientEntity) applyAttrDel(path []string, key string) {
	attr := e.findAttrByPath(path)
	var rootkey string
	if len(path) > 0 {
		rootkey = path[len(path)-1]
	} else {
		rootkey = key
	}

	delete(attr, key)

	callbackFuncName := "OnAttrChange_" + rootkey
	callbackMethod := reflect.ValueOf(e).MethodByName(callbackFuncName)
	if !callbackMethod.IsValid() {
		gwlog.Warn("Attribute change callback of %s is not defined (%s)", rootkey, callbackFuncName)
		return
	}
	callbackMethod.Call([]reflect.Value{}) // call the attr change callback func
}

func (entity *ClientEntity) findAttrByPath(path []string) map[string]interface{} {
	// note that path is reversed
	attr := entity.Attrs // root attr
	plen := len(path)
	for i := plen - 1; i >= 0; i-- {
		name := path[i]
		attr = attr[name].(map[string]interface{})
	}
	return attr
}

func (attrs ClientAttrs) GetInt(key string) int {
	return int(typeconv.Int(attrs[key]))
}

func (entity *ClientEntity) OnAttrChange_exp() {
	if !quiet {
		gwlog.Debug("%s: attr exp change to %d", entity, entity.Attrs.GetInt("exp"))
	}
}

func (entity *ClientEntity) OnAttrChange_testpop() {
	var v int
	if entity.Attrs.HasKey("testpop") {
		v = entity.Attrs.GetInt("testpop")
	} else {
		v = -1
	}
	if !quiet {
		gwlog.Debug("%s: attr testpop change to %d", entity, v)
	}
}

func (entity *ClientEntity) OnAttrChange_subattr() {
	var v interface{}
	if entity.Attrs.HasKey("subattr") {
		v = entity.Attrs["subattr"]
	} else {
		v = nil
	}
	if !quiet {
		gwlog.Debug("%s: attr subattr change to %v", entity, v)
	}
}