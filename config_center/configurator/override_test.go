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
package configurator

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

const (
	defaults = "default"
	override = "override"
	failfast = "failfast"
	failover = "failover"
)

func TestConfigureVersion2p6(t *testing.T) {
	url, err := common.NewURL("override://0.0.0.0:0/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator(defaults, url)
	assert.Equal(t, override, configurator.GetUrl().Protocol)

	providerUrl, err := common.NewURL("jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	assert.NoError(t, err)
	configurator.Configure(providerUrl)
	assert.Equal(t, failfast, providerUrl.GetParam(constant.ClusterKey, ""))
}

func TestConfigureVersionOverrideAddr(t *testing.T) {
	url, err := common.NewURL("override://0.0.0.0:0/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService&providerAddresses=127.0.0.2:20001|127.0.0.3:20001")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator(defaults, url)
	assert.Equal(t, override, configurator.GetUrl().Protocol)

	providerUrl, err := common.NewURL("jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	assert.NoError(t, err)
	configurator.Configure(providerUrl)
	assert.Equal(t, failover, providerUrl.GetParam(constant.ClusterKey, ""))
}

func TestConfigureVersion2p6WithIp(t *testing.T) {
	url, err := common.NewURL("override://127.0.0.1:20001/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator(defaults, url)
	assert.Equal(t, override, configurator.GetUrl().Protocol)

	providerUrl, err := common.NewURL("jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	assert.NoError(t, err)
	configurator.Configure(providerUrl)
	assert.Equal(t, failfast, providerUrl.GetParam(constant.ClusterKey, ""))
}

func TestConfigureVersion2p7(t *testing.T) {
	url, err := common.NewURL("jsonrpc://0.0.0.0:20001/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService&configVersion=1.0&side=provider")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator(defaults, url)

	providerUrl, err := common.NewURL("jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	assert.NoError(t, err)
	configurator.Configure(providerUrl)
	assert.Equal(t, failfast, providerUrl.GetParam(constant.ClusterKey, ""))
}
