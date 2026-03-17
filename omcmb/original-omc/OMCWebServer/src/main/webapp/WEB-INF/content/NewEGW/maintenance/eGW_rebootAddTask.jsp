<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwRebootAddTask .el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	#egwRebootAddTask .el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	#egwRebootAddTask .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#egwRebootAddTask .titleStyML{
		margin-left: 20px;
	}
	#egwRebootAddTask .deviceTableBox{
		margin:10px 0px 0px 45px;
		display: flex;
	}
	#egwRebootAddTask .deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	#egwRebootAddTask .deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	#egwRebootAddTask .specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	#egwRebootAddTask .specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	#egwRebootAddTask .specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	#egwRebootAddTask .egwRebootDeviceTableBox{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	#egwRebootAddTask .egwRebootDeviceTableBox .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	#egwRebootAddTask .egwRebootDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	#egwRebootAddTask .egwRebootDeviceTableBox .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	#egwRebootAddTask .tableTitles{
		margin-left: 45px;
		padding-top: 20px;
		font-size: 14px;
	}
</style>
<div class="flex-ctn" id="egwRebootAddTask" style="overflow:hidden">
	<div  id='temp_add_close' class="placeholder-bt circleIcon" style='position:absolute;top:20px;right:20px;overflow: hidden;' placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div>
	
	<el-form ref="addform" :model="ruleForm" :rules="formRules" label-position="left"  :hide-required-asterisk='true' style='padding-top: 20px;'>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<!-- 基本信息 -->
		<div style="padding: 20px 45px 0px 45px;">
			<el-form-item prop="taskName" label="<%=rb.getString("RenWuMingCheng")%>"  label-width="120px" style="margin-bottom:20px;">
				<el-input style='width:400px;padding-top:7px;' v-model="ruleForm.taskName" :readonly="readonly" placeholder="<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100"></el-input>
			</el-form-item>
			
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		
		<div class="tableTitles">WCG</div>
		<div class="deviceTableBox">
			<div class="deviceSpecifiedBox">
				<div class="deviceSpecifiedTitle"><%=rb.getString("SheBeiZhiDing")%></div>
				<div class="specifiedTypeBox">
					<el-radio-group  v-model="ruleForm.selectAll" :disabled="readonly" @change="selectAllChange">
						<el-radio  label="true" style="margin-bottom:26px;"><%=rb.getString("QuanBu")%></el-radio>
						<el-radio  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
					</el-radio-group>
				</div>
			</div>
			<div class="egwRebootDeviceTableBox">
				<el-pairgrid
					v-if="showPairGrid && ruleForm.selectAll == 'false'" 
					:id="'select_device_list'" 
					:rownumber="true" 
					style="margin-right:45px;"
					ref="egwRebootPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:row-key="'egwCode'"
					:query-params="queryParams" 
					query-name="egwSn" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("eGWBianMa")%>'}" 
					@selection-change="devicesChange" 
					@right-load-success="loadSuccessDevice"
				 >
					<template slot="left">
						<el-table-column type='selection' width="50" align="center"></el-table-column>
						<el-table-column prop="connectionStatus" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connectionStatus!='Exception' && scope.row.connectionStatus!='On' && scope.row.connectionStatus!='updating' && scope.row.connectionStatus!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connectionStatus=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connectionStatus=='On'||scope.row.connectionStatus=='updating'||scope.row.connectionStatus==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
						<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
						<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
						<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("eGWBianMa")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable 
					id="all_device_list" 
					v-if="ruleForm.selectAll == 'true'" 
					style="border:1px solid #E9E9E9;margin-right:45px;" 
					ref="all_device_list"  
					:url="leftUrl" 
					:height="height" 
					front-pagination="true" 
					pagination="true" 
					:query-params="queryParams"
				 >
				 	<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("eGWBianMa")%>"></el-query>
					</template>
					<el-table-column prop="connectionStatus" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connectionStatus!='Exception' && scope.row.connectionStatus!='On' && scope.row.connectionStatus!='updating' && scope.row.connectionStatus!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connectionStatus=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connectionStatus=='On'||scope.row.connectionStatus=='updating'||scope.row.connectionStatus==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
					<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
					<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
					<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
				</el-ctable>
				<div v-if="!showPairGrid && ruleForm.selectAll == 'false'" >
					<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="selected_device_list"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="queryParams">
						<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
						<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
						<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
						<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
		<el-form-item style="margin-left:45px;" prop="devices"></el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<!-- 执行方式 -->
		<div style="padding: 15px 45px 0px 30px;">
			<div class="el-textarea__inner" style="border:none;">
				<el-form-item style='display:inline-block;' prop='status'>
					<el-radio-group v-model="ruleForm.status" :disabled='readonly' @change="executeTypeChange">
						<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
						<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
						<el-radio label="timing" style='margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop='exetime' style='display:inline-block;' class='timeItem'>
					<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" :disabled="isTimeReadonly" v-model='ruleForm.exetime' size="mini" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
				</el-form-item>
			</div>
		</div>
	</el-form>
	
</div>

<script type="text/javascript">
	/**
	*  页面编辑和只读模式通过readonly控制
	*  校验规则也由readonly决定
	**/
	var egwRebootAddTaskVue = new Vue({
		el: '#egwRebootAddTask',
		data(){
			var vm = this;
			var validateDevice = function(rule,value,callback) { // 校验设备
					if(vm.ruleForm.selectAll == 'true'){
						callback();
					}else{
						if(value.length == 0) {
							callback('<%=rb.getString("QingXuanZeSheBei")%>');
						}else {
							callback();
						}
					}
					
				},
				validateTime = function(rule,value,callback) { // 校验定时时间
					if(vm.ruleForm.status == 'timing') {
						if(value) {
							callback();
						}else {
							callback('<%=rb.getString("QingXuanZeShiJian")%>');
						}
					}else {
						callback();
					}
				};
				
			return {
				taskId:'',
				queryParams: {
					search_text: '',
					timeZone: timeZone

				},
				height:'370px',
				ruleForm: { // 表单数据
					taskId: '',
					taskName: '${addTaskName}',
					status: 'active',
					devices: '',
					timeZone: timeZone,
					exetime: '',
					selectAll:'false',
				},
				rules: { // 校验规则
					taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					devices:[
						{validator: validateDevice}
					],
					exetime:[
						{validator: validateTime}
					]
				},
				showPairGrid: true,
		    	leftUrl : '${ctx}/egw/monitor/getEgwMonitorPageList.action',
		    	rightUrl : '',
				deviceTitle:['','<%=rb.getString("YiXuan")%>'],
				readonly: false,
				pickerOptions:{
					disabledDate(time){
						return time.getTime() < Date.now()-8.64e7;
					}
				},
				operateType:'add',
			}
		},
		computed: {
			formRules() { // 只读模式置空校验
				return this.readonly? []:this.rules;
			},
			isTimeReadonly(){
				return this.readonly || this.ruleForm.status != 'timing';
			},
		},
		watch: {
			'ruleForm.devices': {
				handler: function(val){
					this.$refs.addform.validateField('devices');
				},
				deep: true
			},
			'ruleForm.exetime':function(newValue,oldValue){
				if(newValue == null){
					this.ruleForm.exetime = '';
				}
			},
		},
		methods: {
			init(id,operateType) {// 初始化任务信息
				var vm = this,
					params = {taskId: id, timeZone: timeZone};
				vm.taskId = id;
				vm.operateType = operateType;
				vm.readonly = vm.operateType == 'view' ? true : false;
				
				if(vm.operateType == 'view'){
					vm.showPairGrid = false;
				}
				if(vm.operateType != 'add'){
					axios.post('${ctx}/egw/reboot/getTaskInfo.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data){
							Object.assign(vm.ruleForm, data);
							if(data.selectAll == ''){
								vm.ruleForm.selectAll = 'false';
							}
							if(data.status == 'timing'){
								vm.ruleForm.exetime = data.time;
							}
							initForm(vm.$refs.addform);
						}
					}).catch(function(error){})
					vm.rightUrl = '${ctx}/egw/reboot/getSelectedEGWInfo.action?taskId='+id;

				}else{
					
					if(egwRebootVue.selection.length > 0){
						vm.$nextTick(function(){
							vm.$refs.egwRebootPairgrid.appendCheckedRows(egwRebootVue.selection);
						})
					}
				}
				
			},
			loadSuccessDevice(){
				initForm(this.$refs.addform)
			},
			queryDeviceList(val){// 右侧的筛选搜索
				var vm = this;
				vm.queryParams.search_text = val;
			},
			setTime(){
				this.ruleForm.exetime = formatDate(new Date(gloableTime));
				this.$refs.addform.validateField('exetime');
			},
			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {// 
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.egwRebootPairgrid.getData();
					vm.ruleForm.devices = rows.map(function(row){ return row.egwCode;}).sort().join(',');
				})
			},
			// 关闭
			closePanel(){
				var vm = this;

				if(vm.operateType == 'view'){
					eventBus.$emit('hide-egwReboot-slide');
				}else{
					vm.cancel();
				}
				
			},
			// 执行方式改变事件
			executeTypeChange(val){
				var vm = this;
				if(val !== 'timing'){
					vm.ruleForm.exetime = '';
					vm.$refs.addform.clearValidate('exetime')
				}
			},
			submit(){ // 提交新增模板数据
				var vm = this,
                    urls = '',
					params={
						timeZone:timeZone,
						egwCodes:vm.ruleForm.devices,
						taskName:vm.ruleForm.taskName,
						status:vm.ruleForm.status,
						selectAll:vm.ruleForm.selectAll,
					},
					message = '<%=rb.getString("ChengGong")%>';
				// 防止多次提交
                if(egwRebootVue.slideSubmitLoading)return

				if(vm.ruleForm.selectAll == 'true'){
					params.egwCodes = '';
				}
				if(vm.ruleForm.status == 'timing'){
					params.time = vm.ruleForm.exetime;
				}
                if(vm.operateType == 'add'){
						urls = '${ctx}/egw/reboot/addTask.action';
					}else{
						params.taskId = vm.taskId
						urls = '${ctx}/egw/reboot/editTask.action';
					}
				vm.$refs.addform.validate(function(valid){
					if(valid){
                        egwRebootVue.slideSubmitLoading = true;
						axios.post(urls,stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message: message,
		    						type:'success',
		    					})
                                eventBus.$emit('hide-egwReboot-slide')
		    				}else{
		    					vm.$message.error(data["message"]);
                                egwRebootVue.slideSubmitLoading = false;
		    				}
						}).catch(function(error){})
					}
				});
			},
			// 关闭新建弹窗
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
				if(isFormChanged(vm.$refs.addform)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-egwReboot-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-egwReboot-slide')
				}
			},
		
			// 设备执行类别  true 全部执行 false 指定执行
			selectAllChange(val){
				var vm = this;

				vm.$refs.addform.validateField('devices');
				vm.queryParams.search_text = '';
				if(val == 'true'){
					vm.rightUrl = '';
				}else{
					vm.rightUrl="${ctx}/egw/reboot/getSelectedEGWInfo.action?taskId="+vm.taskId+"&timeZone="+timeZone;
				}
			},
		},
		created(){},
		mounted(){
			eventBus.$off('action-save').$on('action-save',this.submit);
			eventBus.$off('action-init').$on('action-init',this.init);
			eventBus.$off('cancel-slide').$on('cancel-slide',this.cancel);
		}
	});
</script>