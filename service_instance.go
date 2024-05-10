package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strconv"
	"time"
)

func GetSizeLose(pass_node_name string) *NodeInfo {
	service, err := GetThisTypeService()
	if err != nil {
		return nil
	}

	if len(service) == 0 {
		return nil
	}

	var min *NodeInfo
	var minSize int64 = -1

	for _, v := range service {

		if v.Name == pass_node_name {
			continue
		}

		val := globalRedisClient.Get(context.Background(), "mqtt_size:"+v.Name).Val()
		if val == "" {
			val = "0"
		}

		zap.S().Infof("节点 %s 数量为 %s", v.Name, val)

		i, err := strconv.ParseInt(val, 10, 64)

		if err != nil {
			continue
		}

		if i < v.Size {

			if min == nil || v.Size < minSize {
				min = &v
				minSize = v.Size
			}
		} else {
			continue
		}
	}

	if min == nil {
		return nil
	}
	zap.S().Infof("选中节点 = %#v", min)

	return min
}

// 获取所有实例列表
func GetThisTypeService() ([]NodeInfo, error) {

	// 使用 context.Background() 作为请求的上下文
	all, err := globalRedisClient.HGetAll(context.Background(), "register:"+globalConfig.NodeInfo.Type).Result()
	if err != nil {
		return nil, err // 如果有错误，返回错误
	}

	var res []NodeInfo
	for _, v := range all {
		var nodeInfo NodeInfo
		err := json.Unmarshal([]byte(v), &nodeInfo)
		if err != nil {
			return nil, err
		}
		res = append(res, nodeInfo)
	}

	return res, nil

}

// 根据节点名称获取节点配置
func GetNodeInfo(name string) (NodeInfo, error) {
	ctx := context.Background()
	val, err := globalRedisClient.HGet(ctx, "register:"+globalConfig.NodeInfo.Type, name).Result()
	if err == redis.Nil {
		return NodeInfo{}, errors.New("node info not found")
	}
	if err != nil {
		return NodeInfo{}, err
	}

	var nodeInfo NodeInfo
	err = json.Unmarshal([]byte(val), &nodeInfo)
	if err != nil {
		return NodeInfo{}, err
	}

	return nodeInfo, nil
}

// 删除一个节点
func RemoveNodeInfo(name string) error {
	ctx := context.Background()
	_, err := globalRedisClient.HDel(ctx, "register:"+globalConfig.NodeInfo.Type, name).Result()

	if err != nil {
		return err
	}
	return nil
}

// 健康任务: 每秒写入心跳数据到redis
func BeatTask(f NodeInfo) {

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			// 执行定时任务的逻辑
			Register(f)
		}
	}
}

// 注册数据
func Register(f NodeInfo) {
	jsonData, err := json.Marshal(f)

	zap.S().Debugf("健康数据 data %v", string(jsonData))

	if err != nil {
		zap.S().Fatalf("Error marshalling to JSON: %v", err)
	}
	globalRedisClient.Set(context.Background(), "beat:"+f.Type+":"+f.Name, f.Name, 1200*time.Millisecond)
	globalRedisClient.HSet(context.Background(), "register:"+f.Type, f.Name, jsonData)
	zap.S().Debugf("健康心跳")
}

// 获取节点使用信息
func GetNodeUseInfo() {

}
