<script setup lang="ts">
import {onMounted, reactive, ref} from 'vue'
import {ElMessage, TabsPaneContext} from 'element-plus'
import axios from 'axios'

const activeName = ref('first')
const tableData = ref([])
const handleClick = (tab: TabsPaneContext, event: Event) => {
  console.log(tab, event)
}
onMounted(async () => {
  get_node_status();
})
const get_node_status = () => {
  const url = 'http://localhost:9999/node_using_status';

  axios.get(url).then(function (response) {
    if (response.data.status == 200) {
      tableData.value = response.data.data

    }
  }).catch(function (error) {
    console.log(error);
  });

}
const cur_mqtt_list = ref([])
const show_mqtt_client = (scope: any) => {
  console.log(scope)
  cur_mqtt_list.value = scope.client_infos;
  dialogVisible.value = true
};


const dialogVisible = ref(false)
const smq = ref(false)

const show_new_mqtt = ()=>{
  smq.value = true
}
const add_param = reactive({
  "broker": "",
  "port": 0,
  "username": "",
  "password": "",
  "sub_topic": "",
  "client_id":""
})
const onSubmit = ()=>{
  console.log(add_param);



  axios.post("http://localhost:9999/public_create_mqtt",add_param)
      .then((response) => {
        console.log(JSON.stringify(response.data));
        if (response.data.status == 200) {
          ElMessage('创建成功')
          get_node_status();
          smq.value = false
        }
        else {
          ElMessage('创建失败')

        }
      })
      .catch((error) => {
        console.log(error);
      });

}
</script>

<template>
  <div>
    <el-button @click="show_new_mqtt">新增客户端</el-button>
    <el-tabs v-model="activeName" class="demo-tabs" @tab-click="handleClick">
      <el-tab-pane label="节点状态" name="first">

        <el-table :data="tableData" style="width: 100%">omom
          <el-table-column prop="name" label="节点名称" width="180"/>
          <el-table-column prop="max_size" label="最大可用数量" width="180"/>
          <el-table-column prop="size" label="已用数量"/>
          <el-table-column fixed="right" label="操作" width="120">
            <template #default="scope">
              <el-button link type="primary" size="small" @click="show_mqtt_client(scope.row)">
                客户端信息
              </el-button>
            </template>
          </el-table-column>
        </el-table>

      </el-tab-pane>
    </el-tabs>


    <el-dialog
        v-model="dialogVisible"
        title="客户端情况"
        width="75%"
    >
      <el-table :data="cur_mqtt_list" style="width: 100%">
        <el-table-column prop="broker" label="IP" width="180"/>
        <el-table-column prop="port" label="端口" width="180"/>
        <el-table-column prop="username" label="账号"/>
        <el-table-column prop="password" label="密码" width="180"/>
        <el-table-column prop="sub_topic" label="订阅地址" width="180"/>
        <el-table-column prop="client_id" label="客户端id"/>
      </el-table>

    </el-dialog><el-dialog
        v-model="smq"
        title="新增 MQTT 客户端"
        width="75%"
    >

    <el-form :model="add_param" label-width="auto" style="max-width: 600px">
      <el-form-item label="IP">
        <el-input v-model="add_param.broker" />
      </el-form-item>
       <el-form-item label="端口">
        <el-input v-model="add_param.port" />
      </el-form-item>
       <el-form-item label="账号">
        <el-input v-model="add_param.username" />
      </el-form-item>
       <el-form-item label="密码">
        <el-input v-model="add_param.password" />
      </el-form-item>
       <el-form-item label="订阅地址">
        <el-input v-model="add_param.sub_topic" />
      </el-form-item>
       <el-form-item label="客户端id">
        <el-input v-model="add_param.client_id" />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="onSubmit">Create</el-button>
        <el-button @click="smq =false">Cancel</el-button>
      </el-form-item>
    </el-form>


    </el-dialog>
  </div>
</template>

<style scoped>

</style>
