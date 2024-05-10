package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type ServerConfig struct {
	NodeInfo    NodeInfo    `yaml:"node_info" json:"node_info"`
	RedisConfig RedisConfig `yaml:"redis_config" json:"redis_config"`
}

type NodeInfo struct {
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Port int    `json:"port,omitempty" yaml:"port,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	Size int64  `json:"size,omitempty" yaml:"size,omitempty"` // 最大处理数量
}

type RedisConfig struct {
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     int    `json:"port,omitempty" yaml:"port,omitempty"`
	Db       int    `json:"db,omitempty" yaml:"db,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// 监听过期键
func ListenerBeat() {
	client := globalRedisClient

	client.ConfigSet(context.Background(), "notify-keyspace-events", "Ex")
	pubsub := client.Subscribe(context.Background(), "__keyevent@0__:expired")
	defer pubsub.Close()
	for {
		msg, err := pubsub.ReceiveMessage(context.Background())
		if err != nil {
			zap.S().Fatalf("Error %s", err)
			return
		}
		zap.S().Debugf("过期键 %s ", msg.Payload)

		if strings.HasPrefix(msg.Payload, "beat") {

			parts := strings.Split(msg.Payload, ":")

			lastElement := parts[len(parts)-1]

			HandlerOffNode(lastElement)
		}
	}
}

// 发送异常数据给需要处理的机器
func sendErrorMsg(node *NodeInfo, error_node string) {
	data := map[string]interface{}{
		"error_node": error_node,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		zap.S().Error("Error marshalling JSON:", err)
		return
	}

	url := fmt.Sprintf("http://%s:%d/handler_faild", node.Host, node.Port) // Replace <PORT> with the actual port number
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		zap.S().Error("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zap.S().Error("发送请求异常:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
}
