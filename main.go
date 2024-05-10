package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"time"
)

var globalConfig ServerConfig

func main() {
	InitLog()

	var configPath string
	flag.StringVar(&configPath, "config", "app-node1.yml", "Path to the config file")
	flag.Parse()

	yfile, err := ioutil.ReadFile(configPath)
	if err != nil {
		zap.S().Fatalf("error: %v", err)
	}

	err = yaml.Unmarshal([]byte(yfile), &globalConfig)
	if err != nil {
		zap.S().Fatalf("error: %v", err)
	}

	zap.S().Debugf("node name = %v , host = %v , port = %v", globalConfig.NodeInfo.Name, globalConfig.NodeInfo.Host, globalConfig.NodeInfo.Port)

	initGlobalRedisClient(globalConfig.RedisConfig)
	Pprof()

	before_start()
	startHttp()

}

func before_start() {
	go removeOldData()

	go BeatTask(globalConfig.NodeInfo)
	go ListenerBeat()
	go CBeat()
	go timerNoHandlerConfig()
}
func removeOldData() {
	HandlerOffNode(globalConfig.NodeInfo.Name)
}

// 内部健康检查
func CBeat() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			lock := NewRedisDistLock(globalRedisClient, "c_beat")
			if lock.TryLock() {
				service, err := GetThisTypeService()
				if err == nil {
					for _, info := range service {
						if !SendBeat(&info, "beat") {
							globalRedisClient.Del(context.Background(), "mqtt_size:"+info.Name)
							globalRedisClient.HDel(context.Background(), "register:"+globalConfig.NodeInfo.Type, info.Name)

							if HandlerOffNode(info.Name) {
								return
							}
						}

					}
				}
				lock.Unlock()

			} else {

				zap.S().Error("没有获取到 c_beat 处理的锁")

			}
		}
	}

}

func HandlerOffNode(node_name string) bool {
	globalRedisClient.Del(context.Background(), "mqtt_size:"+node_name)

	mqtt_client_ids := GetBindClientId(node_name)
	for _, ele := range mqtt_client_ids {
		cf := GetUseConfig(ele)
		var config MqttConfig
		bytes := []byte(cf)
		err := json.Unmarshal(bytes, &config)
		if err != nil {
			zap.S().Fatalf("HandlerOffNode Error unmarshalling JSON: %s", err)
			return true
		}

		RemoveBindNode(config.ClientId, node_name)

	}
	return false
}

func startHttp() {
	http.HandleFunc("/beat", HttpBeat)
	http.HandleFunc("/create_mqtt", CreateMqttClientHttp)
	http.HandleFunc("/public_create_mqtt", PubCreateMqttClientHttp)
	http.HandleFunc("/node_list", NodeList)
	http.HandleFunc("/node_using_status", NodeUsingStatus)
	http.HandleFunc("/mqtt_config", GetUseMqttConfig)
	http.HandleFunc("/no_mqtt_config", GetNoUseMqttConfig)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", globalConfig.NodeInfo.Port), nil); err != nil {
		zap.S().Fatalf("Failed to start server: %s", err)
	}
}

func timerNoHandlerConfig() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			// 执行定时任务的逻辑

			noHandlerConfig()
		}
	}
}

func noHandlerConfig() {
	lock := NewRedisDistLock(globalRedisClient, "no_handler_config_lock")
	if lock.TryLock() {

		zap.S().Debugf("获取处理 no_handler_config 的锁")
		config_list := GetNoUseConfig()

		for _, conf := range config_list {

			if PubCreateMqttClientOp(conf) == -1 {
				continue
			}
		}
		lock.Unlock()
	} else {
		zap.S().Error("没有获取到 no_handler_config 处理的锁")
	}

}

func PubCreateMqttClientOp(conf string) int {
	lose := GetSizeLose("")

	if lose != nil {

		if SendCreateMqttMessage(lose, conf) {
			return 1
		} else {
			zap.S().Errorf("发送 MQTT 客户端创建请求异常 %v", lose)
			return -1
		}

	} else {
		zap.S().Error("没有找到可用节点")
		return -1
	}
	return -1
}
