package main

import (
	"context"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
)

// MqttConfig 定义了MQTT客户端配置的结构体
type MqttConfig struct {
	Broker   string `json:"broker"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	SubTopic string `json:"sub_topic"`
	ClientId string `json:"client_id"`
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	zap.S().Debugf("处理消息: %s  消息主题: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	zap.S().Debugf("MQTT客户端链接成功")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {

	reader := client.OptionsReader()
	id := reader.ClientID()
	RemoveBindNode(id, globalConfig.NodeInfo.Name)
	globalRedisClient.Decr(context.Background(), "mqtt_size:"+globalConfig.NodeInfo.Name)

	zap.S().Error("失去链接: %v", err)
}

func CreateMqttClientMin(broker string, port int, username string, password string, sub_topic string, client_id string) mqtt.Client {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))

	opts.SetClientID(client_id)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	sub(client, sub_topic)
	return client

}

func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	zap.S().Debugf("订阅主题: %s", topic)
}

func CreateMqttClient(config MqttConfig, body []byte) int64 {

	result := globalRedisClient.Get(context.Background(), "mqtt_size:"+globalConfig.NodeInfo.Name).Val()
	if result == "" {

		result = "0"
	}
	i, err := strconv.ParseInt(result, 10, 64)

	if err == nil {
		if globalConfig.NodeInfo.Size > i {
			CreateMqttClientMin(config.Broker, config.Port, config.Username, config.Password, config.SubTopic, config.ClientId)
			zap.S().Debugf("创建mqtt客户端成功")
			return i + 1

		} else {
			return -1

		}
	}

	return -1

}
