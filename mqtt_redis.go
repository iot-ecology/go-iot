package main

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
)

func AddNoUseConfig(config MqttConfig, body []byte) {
	globalRedisClient.HSet(context.Background(), "mqtt_config:no", config.ClientId, body)
}

func GetNoUseConfig() []string {
	var noUseConfigs []string
	for _, s2 := range globalRedisClient.HGetAll(context.Background(), "mqtt_config:no").Val() {

		noUseConfigs = append(noUseConfigs, s2)
	}
	return noUseConfigs
}

func RemoveNoUseConfig(config MqttConfig) {
	globalRedisClient.HDel(context.Background(), "mqtt_config:no", config.ClientId)
}

func AddUseConfig(config MqttConfig) {

	jsonStr, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	globalRedisClient.HSet(context.Background(), "mqtt_config:use", config.ClientId, jsonStr)
}

func GetUseConfig(client_id string) string {
	return globalRedisClient.HGet(context.Background(), "mqtt_config:use", client_id).Val()
}
func GetNoUseConfigById(client_id string) string {
	return globalRedisClient.HGet(context.Background(), "mqtt_config:no", client_id).Val()
}

func RemoveUseConfig(config MqttConfig) {
	globalRedisClient.HDel(context.Background(), "mqtt_config:use", config.ClientId)
}

func CheckHasConfig(config MqttConfig) bool {
	return globalRedisClient.HExists(context.Background(), "mqtt_config:use", config.ClientId).Val()
}

func BindNode(config MqttConfig, node_name string) {
	globalRedisClient.RPush(context.Background(), "node_bind:"+node_name, config.ClientId)

	RemoveNoUseConfig(config)
	AddUseConfig(config)

}

func RemoveBindNode(client_id string, node_name string) {
	globalRedisClient.LRem(context.Background(), "node_bind:"+node_name, 0, client_id)

	config_str := GetUseConfig(client_id)
	var config MqttConfig
	var byt = []byte(config_str)

	err := json.Unmarshal(byt, &config)
	if err != nil {
		zap.S().Error("Error unmarshalling JSON:", err)
		return
	}

	RemoveUseConfig(config)
	AddNoUseConfig(config, byt)
}

func GetBindClientId(node_name string) []string {
	return globalRedisClient.LRange(context.Background(), "node_bind:"+node_name, 0, -1).Val()

}
