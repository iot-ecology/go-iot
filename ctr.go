package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

func HandlerFaild(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析JSON主体数据
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Failed to parse JSON body", http.StatusBadRequest)
		return
	}

	// 打印解析后的JSON数据
	zap.S().Debugf("接收到需要处理的异常数据: ", data)

	if value, ok := data["error_node"]; ok {
		zap.S().Debugf("Found error_node:", value)

	} else {
		zap.S().Debugf("error_node not found in the map")
	}

	// 向客户端发送响应消息
	fmt.Fprintf(w, "Received and parsed JSON data successfully")

}

func SendCreateMqttMessage(node *NodeInfo, param string) bool {

	url := fmt.Sprintf("http://%s:%d/create_mqtt", node.Host, node.Port)
	data := []byte(param)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		zap.S().Fatalf("Error creating request:", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	bodyString := string(bodyBytes)

	zap.S().Infof("Response Status: %v , body = %v", resp.Status, bodyString)
	return resp.Status == "200 OK"

}

func SendBeat(node *NodeInfo, param string) bool {

	url := fmt.Sprintf("http://%s:%d/beat", node.Host, node.Port)
	data := []byte(param)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(data))
	if err != nil {
		zap.S().Fatalf("Error creating request:", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//zap.S().Fatalf("发送请求异常:", err)
		return false
	}
	defer resp.Body.Close()

	return resp.Status == "200 OK"

}

func HttpBeat(w http.ResponseWriter, r *http.Request) {
	// 检查请求方法
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return
	}
	// 向客户端发送响应消息
	fmt.Fprintf(w, "ok")
}

func CreateMqttClientHttp(w http.ResponseWriter, r *http.Request) {
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return
	}
	// 读取请求体
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// 解析JSON请求体到MqttConfig结构体
	var config MqttConfig
	err = json.Unmarshal(body, &config)
	if err != nil {
		http.Error(w, "Error decoding JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if CheckHasConfig(config) {
		json.NewEncoder(w).Encode(map[string]any{"status": 400, "message": "已经存在客户端id"})
		return

	} else {

		usz := CreateMqttClient(config, body)

		if usz == -1 {
			json.NewEncoder(w).Encode(map[string]any{"status": 400, "message": "达到最大客户端数量"})
			return

		} else {
			globalRedisClient.Incr(context.Background(), "mqtt_size:"+globalConfig.NodeInfo.Name)
			AddNoUseConfig(config, body)
			BindNode(config, globalConfig.NodeInfo.Name)
			json.NewEncoder(w).Encode(map[string]any{"status": 200, "message": "创建成功", "size": usz})
			return

		}
	}

}

func PubCreateMqttClientHttp(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return
	}
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	str := string(body)
	if PubCreateMqttClientOp(str) == 1 {
		json.NewEncoder(w).Encode(map[string]any{"status": 200, "message": "创建成功"})
		return
	} else {
		json.NewEncoder(w).Encode(map[string]any{"status": 200, "message": "创建失败"})
		return

	}
}

func NodeList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	service, err := GetThisTypeService()
	if err != nil {
		zap.S().Errorf("节点列表获取失败: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 设置响应的Content-Type为application/json
	w.Header().Set("Content-Type", "application/json")

	// 编码并发送JSON响应
	json.NewEncoder(w).Encode(map[string]any{
		"status":  200,
		"message": "创建成功",
		"data":    service,
	})
}

func NodeUsingStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	service, err := GetThisTypeService()
	if err != nil {
		zap.S().Errorf("节点列表获取失败: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 定义一个结构体用于存放节点名称和大小
	type NodeInfo struct {
		Name        string       `json:"name"`
		Size        int64        `json:"size"`
		ClientIds   []string     `json:"client_ids"`
		ClientInfos []MqttConfig `json:"client_infos"`
		MaxSize     int64        `json:"max_size"`
	}

	// 初始化一个NodeInfo类型的切片
	var nodeInfos []NodeInfo

	// 遍历service中的每个info
	for _, info := range service {
		result := globalRedisClient.Get(context.Background(), "mqtt_size:"+info.Name).Val()
		if result == "" {
			result = "0"
		}

		// 将字符串result解析为int64
		size, err := strconv.ParseInt(result, 10, 64)
		if err != nil {
			// 如果解析失败，记录错误并为这个节点设置默认大小0
			zap.S().Errorf("解析节点'%s'的大小失败: %v", info.Name, err)
			size = 0
		}

		var mc []MqttConfig

		for _, el := range GetBindClientId(info.Name) {
			// 假设GetUseConfig函数返回配置的JSON字符串和错误
			configJSON := GetUseConfig(el)

			var config MqttConfig
			bytes := []byte(configJSON)
			err := json.Unmarshal(bytes, &config)
			if err != nil {
				zap.S().Fatalf("HandlerOffNode Error unmarshalling JSON: %s", err)
			}
			mc = append(mc, config)

		}

		// 创建NodeInfo实例并添加到切片中
		nodeInfos = append(nodeInfos, NodeInfo{
			Name:        info.Name,
			Size:        size,
			MaxSize:     info.Size,
			ClientIds:   GetBindClientId(info.Name),
			ClientInfos: mc,
		})

	}

	// 至此，nodeInfos切片包含了所有的节点名称和大小
	// 你可以将这个切片编码为JSON并发送给客户端
	json.NewEncoder(w).Encode(map[string]any{
		"status":  200,
		"message": "成功",
		"data":    nodeInfos,
	})
}

func GetUseMqttConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 从请求中解析查询参数
	query := r.URL.Query()

	// 获取"id"查询参数的值
	id := query.Get("id")

	// 检查是否获取到了id参数
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	// 假设GetUseConfig函数返回配置的JSON字符串和错误
	configJSON := GetUseConfig(id)

	var config MqttConfig
	bytes := []byte(configJSON)
	err := json.Unmarshal(bytes, &config)
	if err != nil {
		zap.S().Fatalf("HandlerOffNode Error unmarshalling JSON: %s", err)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  200,
			"message": "Success",
			"data":    nil,
		})
		return
	}

	// 将配置信息编码为JSON并发送给客户端
	json.NewEncoder(w).Encode(map[string]any{
		"status":  200,
		"message": "Success",
		"data":    config,
	})
	return
}

func GetNoUseMqttConfig(w http.ResponseWriter, r *http.Request) {
	if cros(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 从请求中解析查询参数
	query := r.URL.Query()

	// 获取"id"查询参数的值
	id := query.Get("id")

	// 检查是否获取到了id参数
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	// 假设GetUseConfig函数返回配置的JSON字符串和错误
	configJSON := GetNoUseConfigById(id)

	var config MqttConfig
	bytes := []byte(configJSON)
	err := json.Unmarshal(bytes, &config)
	if err != nil {
		zap.S().Fatalf("HandlerOffNode Error unmarshalling JSON: %s", err)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  200,
			"message": "Success",
			"data":    nil,
		})
		return
	}

	// 将配置信息编码为JSON并发送给客户端
	json.NewEncoder(w).Encode(map[string]any{
		"status":  200,
		"message": "Success",
		"data":    config,
	})
	return
}

func cros(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// 允许的请求方法
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	// 允许的请求头部
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	// 非简单请求时，浏览器会先发送一个预检请求(OPTIONS)，这里处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent) // 200 OK 也可以
		return true
	}
	return false
}
