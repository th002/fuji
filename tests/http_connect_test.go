// Copyright 2015-2016 Shiguredo Inc. <fuji@shiguredo.jp>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"

	"github.com/shiguredo/fuji"
	"github.com/shiguredo/fuji/broker"
	"github.com/shiguredo/fuji/config"
	"github.com/shiguredo/fuji/gateway"
	MYHTTP "github.com/shiguredo/fuji/http"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

// Fuji for TestHttpConnectLocalPub
func fujiHttpConnectLocalPub(t *testing.T, httpConfigStr string) chan string {
	assert := assert.New(t)

	conf, err := config.LoadConfigByte([]byte(httpConfigStr))
	assert.Nil(err)
	commandChannel := make(chan string, 2)
	go fuji.StartByFileWithChannel(conf, commandChannel)
	t.Logf("fuji started")
	return commandChannel
}

// HTTP server with JSON response
func httpTestServerEchoBack(t *testing.T, cmdChan chan string, expectedJsonBody string) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, expectedJsonBody)
		t.Logf("request arrived: %v", r)
	}))
	defer ts.Close()

	// get usable port number
	t.Logf("Address: %s\n", ts.URL)

	cmdChan <- ts.URL

	// wait operation complete
	<-cmdChan
	t.Log("httpd shutdown request has come")
}

func httpTestServerRedirect(t *testing.T, cmdChan chan string, expectedJsonBody string) {
	ts := httptest.NewServer(http.RedirectHandler("http://127.0.0.1/redirected", 304))
	defer ts.Close()

	// get usable port number
	t.Logf("Address: %s\n", ts.URL)

	cmdChan <- ts.URL

	// wait operation complete
	<-cmdChan
}

func httpTestServerNotFound(t *testing.T, cmdChan chan string, expectedJsonBody string) {
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	// get usable port number
	t.Logf("Address: %s\n", ts.URL)

	cmdChan <- ts.URL

	// wait operation complete
	<-cmdChan
}

// TestHttpConnect general flow
// 1. connect gateway to local broker
// 2. subscribe to HTTP request
// 3. check subscribe
// 4. issue HTTP request
// 5. get HTTP response
// 6. publish response

func TestHttpConnectPostLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`","method":"POST","body":{"a":"b"}}`,
		`{"a":"b"}`,
		`{"id":"aasfa","status":200,"body":{"a":"b"}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httppostconnect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectBadJSONNoIdRequesttLocalPubSub(t *testing.T) {
	expected := []string{
		`{"url":"`,
		`","method": "POST", "body":{"a":"b"}}`,
		`NOT USED`,
		`{"id":"","status":` + strconv.Itoa(MYHTTP.InvalidResponseCode) + `,"body":{}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httppostnoid"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectBadJSONNoUrlRequesttLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","urll":"`,
		`","method": "POST", "body":{"a":"b"}}`,
		`NOT USED`,
		`{"id":"aasfa","status":` + strconv.Itoa(MYHTTP.InvalidResponseCode) + `,"body":{}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httppostnourl"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectBadJSONNoMethodRequesttLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`","body":{"a":"b"}}`,
		`NOT USED`,
		`{"id":"aasfa","status":` + strconv.Itoa(MYHTTP.InvalidResponseCode) + `,"body":{}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httppostnomethod"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectBadJSONNoBodyRequesttLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`","method": "POST"}`,
		`NOT USED`,
		`{"id":"aasfa","status":` + strconv.Itoa(MYHTTP.InvalidResponseCode) + `,"body":{}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httppostnobody"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectGetLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`/?a=b","method":"GET","body":{}}`,
		`NOT USED`,
		`{"id":"aasfa","status":200,"body":{"a":"b"}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httpgetconnect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectBadURLGetLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"badprefix`,
		`/?a=b","method":"GET","body":{}}`,
		`NOT USED`,
		`{"id":"aasfa","status":` + strconv.Itoa(MYHTTP.InvalidResponseCode) + `,"body":{}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httpgetbadurlconnect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerEchoBack)
}

func TestHttpConnectRedirectPostLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`","method":"POST","body":{"a":"b"}}`,
		`{"a":"b"}`, // in fact, not used
		`{"id":"aasfa","status":304,"body":""}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httpgetconnectredirect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerRedirect)
}

func TestHttpConnectNotFoundGetLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`/?a=b","method":"GET","body":{}}`,
		`NOT USED`,
		`{"id":"aasfa","status":404,"body":"404 page not found
"}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httpgetconnectnotfound"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 2
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected, httpTestServerNotFound)
}

func generalTestProcess(t *testing.T, httpConfigStr string, expected []string, httpTestServer func(*testing.T, chan string, string)) {
	// initial wait for previous subscriber client shutdown
	time.Sleep(500 * time.Millisecond)

	assert := assert.New(t)

	requestJson_pre, requestJson_post, expectedJsonBody, expectedJson := expected[0], expected[1], expected[2], expected[3]

	// start fuji
	fujiCommandChannel := fujiHttpConnectLocalPub(t, httpConfigStr)

	// start http server
	httpCommandChannel := make(chan string, 2)
	go httpTestServer(t, httpCommandChannel, expectedJsonBody)
	// wait for bootup
	listener := <-httpCommandChannel
	t.Logf("http started at: %s", listener)

	// pub/sub test to broker on localhost
	// publised JSON messages confirmed by subscriber

	// get config
	conf, err := config.LoadConfigByte([]byte(httpConfigStr))
	assert.Nil(err)

	// get Gateway
	gw, err := gateway.NewGateway(conf)
	assert.Nil(err)

	// get Broker
	brokerList, err := broker.NewBrokers(conf, gw.BrokerChan)
	assert.Nil(err)

	// Setup MQTT pub/sub client to confirm published content.
	//
	subscriberChannel := make(chan [2]string, 2)

	opts := MQTT.NewClientOptions()
	url := fmt.Sprintf("tcp://%s:%d", brokerList[0].Host, brokerList[0].Port)
	opts.AddBroker(url)
	opts.SetClientID(fmt.Sprintf("prefix%s", gw.Name))
	opts.SetCleanSession(false)

	client := MQTT.NewClient(opts)
	// defer client.Disconnect(250)

	assert.Nil(err)
	token := client.Connect()
	token.Wait()
	assert.Nil(token.Error())

	qos := 2
	requestTopic := fmt.Sprintf("%s/%s/http/request", brokerList[0].TopicPrefix, gw.Name)
	expectedTopic := fmt.Sprintf("%s/%s/http/response", brokerList[0].TopicPrefix, gw.Name)
	t.Logf("expetcted topic: %s\nexpected message%s", expectedTopic, expectedJson)
	token = client.Subscribe(expectedTopic, byte(qos), func(client *MQTT.Client, msg MQTT.Message) {
		t.Log("subscriber received topic: %s, message: %s", msg.Topic(), msg.Payload())
		subscriberChannel <- [2]string{msg.Topic(), string(msg.Payload())}
	})
	token.Wait()
	assert.Nil(token.Error())

	// publish JSON
	token = client.Publish(requestTopic, 0, false, requestJson_pre+listener+requestJson_post)
	token.Wait()
	assert.Nil(token.Error())

	// wait for 1 publication of dummy worker
	t.Logf("wait for 1 publication of dummy worker")
	select {
	case message := <-subscriberChannel:
		assert.Equal(expectedTopic, message[0])
		var respJsonMap map[string]interface{}
		var expectedJsonMap map[string]interface{}

		err := json.Unmarshal([]byte(message[1]), &respJsonMap)
		if err != nil {
			break
		}
		err = json.Unmarshal([]byte(expectedJson), &expectedJsonMap)
		assert.Nil(err)
		assert.Equal(expectedJsonMap["id"], respJsonMap["id"])
		assert.Equal(expectedJsonMap["status"], respJsonMap["status"])
		if expectedJsonBody != "NOT USED" {
			assert.Equal(expectedJsonMap["body"], respJsonMap["body"])
		}

	case <-time.After(time.Second * 11):
		assert.Equal("subscribe completed in 11 sec", "not completed")
	}

	time.Sleep(100 * time.Millisecond)
	httpCommandChannel <- "done"
	time.Sleep(100 * time.Millisecond)
	fujiCommandChannel <- "close"
	time.Sleep(300 * time.Millisecond)
}
