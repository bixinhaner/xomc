<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
  <title>File</title>
  <style>
    .flex-form {
      display: flex;
      flex-wrap: wrap;
    }
    .flex-form .el-form-item {
      margin-right: 100px;
      margin-bottom: 10px;
    }
  </style>
</head>
<body>
  <div id="gnodeb_tasklist_ctn" class="container">
    <!-- 操作按钮 -->
    <div class="operations" style="background: #fff;">
      <div v-show="false" class="placeholder-bt" placeholder="<%=rb.getString("XinZeng")%>" @click="addTask">
        <span class="el-icon el-icon-circle-add"></span>
      </div>
    </div>
    <!-- 表格组件 -->
      <!--:url="taskUrl"-->
    <el-ctable ref="ctable"
      id="gnb_task_list"
      :time="6"
      :url="taskUrl"
      :query-params="queryParams">
      <!-- 列表toolbar -->
      <template slot="toolbar">
        <el-query placeholder="<%=rb.getString("RenWuMingCheng")%>" @query="query" @advance-query="advanceQuery" @reset="resetQuery">
          <el-form slot="form" :model="queryForm" class="flex-form">
            <el-form-item label="<%=rb.getString("RenWuMingCheng")%>">
              <el-input v-model="queryForm.taskName" maxlength="100"></el-input>
            </el-form-item>
            <el-form-item label="<%=rb.getString("KaiShiShiJian")%>">
              <el-date-picker type="datetimerange" v-model="queryForm.timeRange"
                value-format="yyyy-MM-dd HH:mm:ss"></el-date-picker>
            </el-form-item>
          </el-form>
        </el-query>
      </template>
      <!-- 列表columns -->
      <el-table-column prop="op" label=" " width="50" align="center">
        <template slot-scope="scope">
          <div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="opClick(scope.row,event)"></div>
        </template>
      </el-table-column>
      <el-table-column label='<%=rb.getString("RenWuMingCheng")%>' width="400"  prop="TASK_NAME"></el-table-column>
			<el-table-column label='<%=rb.getString("CaoZuoRen")%>' width="80" prop="CREATE_USER"></el-table-column>
			<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' width="200" prop="CREATE_TIME"></el-table-column>
			<el-table-column label='<%=rb.getString("WenJianMing")%>' width="300" prop="FILE_NAME"></el-table-column>
			<el-table-column label='<%=rb.getString("BanBen")%>' width="200" prop="VERSION"></el-table-column>
			<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="180" prop="PRODUCT"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="120" prop="TASK_STATUS">
				<template slot-scope="scope">
					<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("JinDu")%>' width="100" prop="TASK_PROGRESS"></el-table-column>
			<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="TASK_RESULT" :formatter="resultFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("BaoLiuPeiZhi")%>' width="200" prop="IS_KEEP_CONFIG" :formatter="configFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="START_TIME"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian")%>' width="180" prop="END_TIME"></el-table-column>
    </el-ctable>

    <!-- 菜单 -->
    <el-cmenu ref="menu"
      @click="clickMenu"
      :data="menus"></el-cmenu>
  </div>

  <script>
	  new Vue({
		  el: '#gnodeb_tasklist_ctn',
	    data () {
	      return {
          taskUrl: '${ctx}/task/upgrade/getUpgradeTaskList.action',
          queryForm: {
            taskName: '',
            timeRange: []
          },
          queryParams: {
            timeZone: timeZone,
            taskName: '',
            startTime: '',
            endTime: '',
            taskType: 1,
            isGnb: 1
          },
          menus: [],
	      }
	    },
	    methods: {
        resultFmt(row,column,cellValue,index){//软件升级 任务结果fmt
	    	    	var resultObj = {
	    					"1" : "<%=rb.getString("ChengGong")%>",
	    					"2" : "<%=rb.getString("BuFenChengGong")%>",
	    					"3" : "<%=rb.getString("ShiBai")%>",
	    					"" : ""
	    			}
	    			return resultObj[cellValue];
	    	},
        query(text) {
          var vm = this;

          vm.resetQuery();
          Object.assign(vm.queryParams, {
            taskName: text,
            startTime: '',
            endTime: ''
          });
        },
        resetQuery() {
          Object.assign(this.queryForm, {
            taskName: '',
            timeRange: []
          });
        },
        advanceQuery() {
          var vm = this,
              startTime = '',
              endTime = '';

          if(vm.queryForm.timeRange && vm.queryForm.timeRange.length) {
              startTime = vm.queryForm.timeRange[0];
              endTime = vm.queryForm.timeRange[1];
          }

          Object.assign(vm.queryParams,{
            taskName: vm.queryForm.taskName,
            startTime: startTime,
            endTime: endTime
          })
        },
        configFmt(row,column,cellValue,index){
          if(cellValue == 'true'){
            return '<%=rb.getString("Fou")%>';
          }else{
            return '<%=rb.getString("Shi")%>';
          }
        },
        opClick(row,ev){//软件升级点击操作出现下拉菜单  1.等待   2.进行中  3.暂停  4.已结束 5.终止中 6.暂停中
          var vm = this,
              status = row.TASK_STATUS;

            vm.taskStatus = row.TASK_STATUS;
            if(writableMap['CODE_GNB'] == true) {
                vm.menus= [
                    {label:'<%=rb.getString("JieGuo")%>',code:'view', row: row},
                    {label:'<%=rb.getString("KaiShi")%>',code:'start', row: row},
                    {label:'<%=rb.getString("ZanTing")%>',code:'stop', row: row},
                    {label:'<%=rb.getString("ZhongZhiRenWu")%>',code:'terminate', row: row},
                    {label:'<%=rb.getString("XinXi")%>',code:'info', row: row},
                    {label:'<%=rb.getString("XiuGai")%>',code:'edit', row: row},
                    {label:'<%=rb.getString("ShanChu")%>',code:'del', row: row}
                ];
            }else {
            	vm.menus= [
                    {label:'<%=rb.getString("JieGuo")%>',code:'view', row: row},
                    {label:'<%=rb.getString("XinXi")%>',code:'info', row: row}
                ];
            }

            initTaskStatus(status,vm.menus);

            vm.$nextTick(function(){
              document.body.click();
              vm.showMenus(ev);
            });
        },
        clickMenu(item){//软件升级菜单点击方法
          var vm = this,
              row = item.row,
              codes = {
                view: vm.viewResultUpgradeTask,
                start: vm.activeUpgradeTask,
                stop: vm.suspendUpgradeTask,
                terminate: vm.terminateTask,
                info: vm.viewUpgradeTask,
                edit: vm.editUpgradeTask,
                del: vm.delUpgradeTask
              };

          if(codes[item.code]){
            codes[item.code](row["TASK_ID"], row["TYPE"], row["TASK_STATUS"], row["PRODUCT"]);
          }
        },
        viewResultUpgradeTask(task_id,type,status,product){ // 结果
          var vm = this;
          
          eventBus.$emit('show-sub-result',{taskId: task_id, productType: product});
        },
        activeUpgradeTask(task_id,type){ // 开始
          var vm = this;
          axios.post('${ctx}/task/upgrade/activeTask.action',stringify({
            taskId : task_id,
            type : type,
            isGnb: 1
          })).then(function(response){
            var data = response.data;
            if(data["success"]){
              vm.$refs.ctable.refresh()
            }else{
              vm.$message.error(data["message"])
            }
          })
        },
        suspendUpgradeTask(task_id,type){ // 暂停
          var vm = this;
          axios.post('${ctx}/task/upgrade/suspendTask.action',stringify({
            taskId : task_id,
            type : type,
            isGnb: 1
          })).then(function(response){
            var data = response.data;
            if(data["success"]){
              vm.$refs.ctable.refresh()
            }else{
              vm.$message.error(data["message"])
            }
          })
        },
        terminateTask(task_id,type){ // 终止任务
          var vm = this;
          axios.post('${ctx}/task/upgrade/terminateUpgradeTask.action',stringify({
            taskId : task_id,
            type : type,
            isGnb: 1
          })).then(function(response){
            var data = response.data;
            if(data["success"]){
              vm.$refs.ctable.refresh()
            }else{
              vm.$message.error(data["message"])
            }
          })
        },
        viewUpgradeTask(task_id,type,status){ // 信息
          var vm = this;
          
          eventBus.$emit('add-gnb-task',{taskId: task_id, type: 'info'});
        },
        editUpgradeTask(task_id,type,status){ //  信息查看
          var vm = this;
          
          eventBus.$emit('add-gnb-task',{taskId: task_id, type: 'edit'});
        },
        delUpgradeTask(task_id,type){//软件升级  删除任务
          var vm = this;
          this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
            customClass:'warningConfirm',
            confirmButtonText:'<%=rb.getString("QueDing")%>',
            cancelButtonText:'<%=rb.getString("QuXiao")%>',
            type:'warning',
            closeOnClickModal:false
          }).then(() => {
            axios.post('${ctx}/task/upgrade/delUpgradeTask.action',stringify({
              taskId:task_id,
              type:type,
              isGnb: 1
            })).then(function(response){
              var data = response.data;
              if(data["success"]){
                vm.$refs.ctable.refresh()
                vm.$message({
                  type:'success',
                  message:'<%=rb.getString("ChengGong")%>'
                })
              }else{
                vm.$message.error(data["message"])
              }
            }).catch(function(error){
              
            })
          }).catch()
        },
	      hideMenus() {
	        this.$refs.menu.hide();
	      },
	      showMenus(evt) {
	        this.$refs.menu.show(evt);
	      },
	      addTask(){
          eventBus.$emit('add-gnb-task');
	      }
	    }
	  })
  </script>
</body>
</html>