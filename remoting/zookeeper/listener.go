/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package zookeeper

import (
	"path"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	perrors "github.com/pkg/errors"

	uatomic "go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var defaultTTL = 10 * time.Minute

// nolint
type ZkEventListener struct {
	client      *gxzookeeper.ZookeeperClient
	pathMapLock sync.Mutex
	pathMap     map[string]*uatomic.Int32
	wg          sync.WaitGroup
	exit        chan struct{}
}

// NewZkEventListener returns a EventListener instance
func NewZkEventListener(client *gxzookeeper.ZookeeperClient) *ZkEventListener {
	return &ZkEventListener{
		client:  client,
		pathMap: make(map[string]*uatomic.Int32),
		exit:    make(chan struct{}),
	}
}

// nolint
func (l *ZkEventListener) SetClient(client *gxzookeeper.ZookeeperClient) {
	l.client = client
}

// ListenServiceNodeEvent listen a path node event
func (l *ZkEventListener) ListenServiceNodeEvent(zkPath string, listener remoting.DataListener) {
	// listen l service node
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		if l.listenServiceNodeEvent(zkPath, listener) {
			listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
			l.pathMapLock.Lock()
			delete(l.pathMap, zkPath)
			l.pathMapLock.Unlock()
		}
		logger.Warnf("ListenServiceNodeEvent->listenSelf(zk path{%s}) goroutine exit now", zkPath)
	}(zkPath, listener)
}

// nolint
func (l *ZkEventListener) listenServiceNodeEvent(zkPath string, listener ...remoting.DataListener) bool {
	defer l.wg.Done()

	l.pathMapLock.Lock()
	a, ok := l.pathMap[zkPath]
	if !ok || a.Load() > 1 {
		l.pathMapLock.Unlock()
		return false
	}
	a.Inc()
	l.pathMapLock.Unlock()
	defer a.Dec()

	var zkEvent zk.Event
	for {
		keyEventCh, err := l.client.ExistW(zkPath)
		if err != nil {
			logger.Warnf("existW{key:%s} = error{%v}", zkPath, err)
			return false
		}

		select {
		case zkEvent = <-keyEventCh:
			logger.Warnf("get a zookeeper keyEventCh{type:%s, server:%s, path:%s, state:%d-%s, err:%s}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
			switch zkEvent.Type {
			case zk.EventNodeDataChanged:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDataChanged}", zkPath)
				if len(listener) > 0 {
					content, _, err := l.client.Conn.Get(zkEvent.Path)
					if err != nil {
						logger.Warnf("zk.Conn.Get{key:%s} = error{%v}", zkPath, err)
						return false
					}
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeUpdate, Content: string(content)})
				}
			case zk.EventNodeCreated:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeCreated}", zkPath)
				if len(listener) > 0 {
					content, _, err := l.client.Conn.Get(zkEvent.Path)
					if err != nil {
						logger.Warnf("zk.Conn.Get{key:%s} = error{%v}", zkPath, err)
						return false
					}
					listener[0].DataChange(remoting.Event{Path: zkEvent.Path, Action: remoting.EventTypeAdd, Content: string(content)})
				}
			case zk.EventNotWatching:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNotWatching}", zkPath)
			case zk.EventNodeDeleted:
				logger.Warnf("zk.ExistW(key{%s}) = event{EventNodeDeleted}", zkPath)
				return true
			}
		case <-l.exit:
			return false
		}
	}
}

func (l *ZkEventListener) handleZkNodeEvent(zkPath string, children []string, listener remoting.DataListener) {
	contains := func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	}

	newChildren, err := l.client.GetChildren(zkPath)
	if err != nil {
		// FIXME always false
		if err == errNilChildren {
			content, _, connErr := l.client.Conn.Get(zkPath)
			if connErr != nil {
				logger.Errorf("Get new node path {%v} 's content error,message is  {%v}",
					zkPath, perrors.WithStack(connErr))
			} else {
				listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeUpdate, Content: string(content)})
			}

		} else {
			logger.Errorf("path{%s} child nodes changed, zk.Children() = error{%v}", zkPath, perrors.WithStack(err))
		}
		return
	}

	// a node was added -- listen the new node
	var (
		newNode string
	)
	for _, n := range newChildren {

		newNode = path.Join(zkPath, n)
		logger.Debugf("[Zookeeper Listener] add zkNode{%s}", newNode)
		content, _, connErr := l.client.Conn.Get(newNode)
		if connErr != nil {
			logger.Errorf("Get new node path {%v} 's content error,message is  {%v}",
				newNode, perrors.WithStack(connErr))
		}

		if !listener.DataChange(remoting.Event{Path: newNode, Action: remoting.EventTypeAdd, Content: string(content)}) {
			continue
		}
		// listen l service node
		l.wg.Add(1)
		go func(node string, listener remoting.DataListener) {
			// invoker l.wg.Done() in l.listenServiceNodeEvent
			if l.listenServiceNodeEvent(node, listener) {
				logger.Warnf("delete zkNode{%s}", node)
				listener.DataChange(remoting.Event{Path: node, Action: remoting.EventTypeDel})
				l.pathMapLock.Lock()
				delete(l.pathMap, zkPath)
				l.pathMapLock.Unlock()
			}
			logger.Debugf("handleZkNodeEvent->listenSelf(zk path{%s}) goroutine exit now", node)
		}(newNode, listener)
	}

	// old node was deleted
	var oldNode string
	for _, n := range children {
		if contains(newChildren, n) {
			continue
		}

		oldNode = path.Join(zkPath, n)
		logger.Warnf("delete oldNode{%s}", oldNode)
		listener.DataChange(remoting.Event{Path: oldNode, Action: remoting.EventTypeDel})
	}
}

func (l *ZkEventListener) listenDirEvent(conf *common.URL, zkRootPath string, listener remoting.DataListener) {
	defer l.wg.Done()

	var (
		failTimes int
		ttl       time.Duration
		event     chan struct{}
	)
	event = make(chan struct{}, 4)
	ttl = defaultTTL
	if conf != nil {
		timeout, err := time.ParseDuration(conf.GetParam(constant.RegistryTTLKey, constant.DefaultRegTTL))
		if err == nil {
			ttl = timeout
		} else {
			logger.Warnf("[Zookeeper EventListener][listenDirEvent] Wrong configuration for registry.ttl, error=%+v, using default value %v instead", err, defaultTTL)
		}
	}
	defer close(event)
	for {
		// Get current children with watcher for the zkRootPath
		children, childEventCh, err := l.client.GetChildrenW(zkRootPath)
		if err != nil {
			failTimes++
			if MaxFailTimes <= failTimes {
				failTimes = MaxFailTimes
			}
			logger.Errorf("[Zookeeper EventListener][listenDirEvent] Get children of path {%s} with watcher failed, the error is %+v", zkRootPath, err)

			// May be the provider does not ready yet, sleep failTimes * ConnDelay senconds to wait
			after := time.After(timeSecondDuration(failTimes * ConnDelay))
			select {
			case <-after:
				continue
			}
		}
		failTimes = 0
		if len(children) == 0 {
			logger.Warnf("[Zookeeper EventListener][listenDirEvent] Can not gey any children for the path {%s}, please check if the provider does ready.", zkRootPath)
			continue
		}
		for _, c := range children {
			// Only need to compare Path when subscribing to provider
			if strings.LastIndex(zkRootPath, constant.ProviderCategory) != -1 {
				provider, _ := common.NewURL(c)
				if provider.ServiceKey() != conf.ServiceKey() {
					continue
				}
			}

			// Build the children path
			zkNodePath := path.Join(zkRootPath, c)

			// Save the path to avoid listen repeatedly
			l.pathMapLock.Lock()
			_, ok := l.pathMap[zkNodePath]
			if !ok {
				l.pathMap[zkNodePath] = uatomic.NewInt32(0)
			}
			l.pathMapLock.Unlock()
			if ok {
				logger.Warnf("[Zookeeper EventListener][listenDirEvent] The child with zk path {%s} has already been listened.", zkNodePath)
				continue
			}

			// When Zk disconnected, the Conn will be set to nil, so here need check the value of Conn
			l.client.RLock()
			if l.client.Conn == nil {
				l.client.RUnlock()
				break
			}
			content, _, err := l.client.Conn.Get(zkNodePath)

			l.client.RUnlock()
			if err != nil {
				logger.Errorf("[Zookeeper EventListener][listenDirEvent] Get content of the child node {%v} failed, the error is %+v", zkNodePath, perrors.WithStack(err))
			}
			logger.Debugf("[Zookeeper EventListener][listenDirEvent] Get children!{%s}", zkNodePath)
			if !listener.DataChange(remoting.Event{Path: zkNodePath, Action: remoting.EventTypeAdd, Content: string(content)}) {
				continue
			}
			logger.Debugf("[Zookeeper EventListener][listenDirEvent] listen dubbo service key{%s}", zkNodePath)
			l.wg.Add(1)
			go func(zkPath string, listener remoting.DataListener) {
				// invoker l.wg.Done() in l.listenServiceNodeEvent
				if l.listenServiceNodeEvent(zkPath, listener) {
					listener.DataChange(remoting.Event{Path: zkPath, Action: remoting.EventTypeDel})
					l.pathMapLock.Lock()
					delete(l.pathMap, zkPath)
					l.pathMapLock.Unlock()
				}
				logger.Warnf("listenDirEvent->listenSelf(zk path{%s}) goroutine exit now", zkPath)
			}(zkNodePath, listener)

			// listen sub path recursive
			// if zkPath is end of "providers/ & consumers/" we do not listen children dir
			if strings.LastIndex(zkRootPath, constant.ProviderCategory) == -1 &&
				strings.LastIndex(zkRootPath, constant.ConsumerCategory) == -1 {
				l.wg.Add(1)
				go func(zkPath string, listener remoting.DataListener) {
					l.listenDirEvent(conf, zkPath, listener)
					logger.Warnf("listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
				}(zkNodePath, listener)
			}
		}
		if l.startScheduleWatchTask(zkRootPath, children, ttl, listener, childEventCh) {
			return
		}
	}
}

func (l *ZkEventListener) startScheduleWatchTask(
	zkRootPath string, children []string, ttl time.Duration,
	listener remoting.DataListener, childEventCh <-chan zk.Event) bool {
	// Periodically update provider information
	tickerTTL := ttl
	if tickerTTL > 20e9 {
		tickerTTL = 20e9
	}
	ticker := time.NewTicker(tickerTTL)
	for {
		select {
		case <-ticker.C:
			l.handleZkNodeEvent(zkRootPath, children, listener)
			if tickerTTL < ttl {
				tickerTTL *= 2
				if tickerTTL > ttl {
					tickerTTL = ttl
				}
				ticker.Stop()
				ticker = time.NewTicker(tickerTTL)
			}
		case zkEvent := <-childEventCh:
			logger.Debugf("Get a zookeeper childEventCh{type:%s, server:%s, path:%s, state:%d-%s, err:%v}",
				zkEvent.Type.String(), zkEvent.Server, zkEvent.Path, zkEvent.State, gxzookeeper.StateToString(zkEvent.State), zkEvent.Err)
			ticker.Stop()
			if zkEvent.Type == zk.EventNodeChildrenChanged {
				l.handleZkNodeEvent(zkEvent.Path, children, listener)
			}
			return false
		case <-l.exit:
			logger.Warnf("listen(path{%s}) goroutine exit now...", zkRootPath)
			ticker.Stop()
			return true
		}
	}
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

// ListenServiceEvent is invoked by ZkConsumerRegistry::Register/ZkConsumerRegistry::get/ZkConsumerRegistry::getListener
// registry.go:Listen -> listenServiceEvent -> listenDirEvent -> listenServiceNodeEvent
//                            |
//                            --------> listenServiceNodeEvent
func (l *ZkEventListener) ListenServiceEvent(conf *common.URL, zkPath string, listener remoting.DataListener) {
	logger.Infof("[Zookeeper Listener] listen dubbo path{%s}", zkPath)
	l.wg.Add(1)
	go func(zkPath string, listener remoting.DataListener) {
		l.listenDirEvent(conf, zkPath, listener)
		logger.Warnf("ListenServiceEvent->listenDirEvent(zkPath{%s}) goroutine exit now", zkPath)
	}(zkPath, listener)
}

//func (l *ZkEventListener) valid() bool {
//	return l.client.ZkConnValid()
//}

// Close will let client listen exit
func (l *ZkEventListener) Close() {
	close(l.exit)
	l.wg.Wait()
}
