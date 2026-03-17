<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	.el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	.alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	.titleStyML{
		margin-left: 20px;
	}
	.deviceTableBox{
		margin:10px 0px 0px 45px;
		display: flex;
		height: 60%;
	}
	.egwLogDeviceTableBox{
		width: 100%;
		font-size: 12px !important;
	}
	.egwLogDeviceTableBox .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	.egwLogDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	.egwLogDeviceTableBox .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	.tableTitles{
		margin-left: 45px;
		padding-top: 20px;
		font-size: 14px;
	}
</style>
<div class="panelDefault" id="egwLogAddTask" style="overflow:auto">
	<div  id='temp_add_close' class="placeholder-bt" style='position:absolute;top:0px;right:20px;overflow: hidden;' placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div>
	
	<el-form ref="addform" :model="form" :rules="formRules" label-position="left"  :hide-required-asterisk='true'>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		
		<div class="tableTitles">
			eGWS
			<span style="font-size:12px;color:#4D84FF;margin-left:10px;"><%=rb.getString("ZuiDuoXuanZeSheBei5")%></span>
		</div>
		<div class="deviceTableBox">
			<div class="egwLogDeviceTableBox">
				<el-pairgrid
					:id="'select_device_list'" 
					:rownumber="true" 
					:limit="limitNum"
					style="margin-right:45px;"
					ref="egwLogPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:row-key="'gw_id'" 
					:query-params="queryParams" 
					query-name="gw_name" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("EGWMingCheng")%>'}" 
					@selection-change="devicesChange" 
					@right-load-success="loadSuccessDevice"
				 >
					<template slot="left">
						<el-table-column type='selection' width="50" align="center"></el-table-column>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="gw_name" show-overflow-tooltip label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
						<el-table-column prop="gw_ip" show-overflow-tooltip label="<%=rb.getString("eGWIP")%>" min-width="160" ></el-table-column>
						<el-table-column prop="gw_port" show-overflow-tooltip label="<%=rb.getString("EGWDuanKou")%>" min-width="250" ></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("EGWMingCheng")%> / <%=rb.getString("eGWIP")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop="gw_name" show-overflow-tooltip label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
					</template>
				</el-pairgrid>
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
				<el-form-item style='display:inline-block;' prop='executeType'>
					<el-radio-group v-model="form.executeType" :disabled='readonly' @change="executeTypeChange">
						<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
						<el-radio label="timing" style='margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop='quartzTime' style='display:inline-block;' class='timeItem'>
					<el-date-picker 
						:disabled="isTimeReadonly" 
						v-model='form.quartzTime' size="mini"  
						value-format="yyyy-MM-dd HH:mm:ss" 
						type="datetimerange"
						:picker-options="pickerOptions"
						range-separator="——"  
						start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
						end-placeholder='<%=rb.getString("JieShuShiJian")%>'
					></el-date-picker>	
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
	var egwLogAddTaskVue = new Vue({
		el: '#egwLogAddTask',
		data(){
			var vm = this;
			var validateDevice = function(rule,value,callback) { // 校验设备
					if(value.length == 0) {
						callback('<%=rb.getString("QingXuanZeSheBei")%>');
					}else {
						callback();
					}
				},
				validateTime = function(rule,value,callback) { // 校验定时时间
					if(vm.form.executeType == 'timing') {
						if(value && value.length > 0) {
							var startTime = new Date(value[0]);
							var endTime = new Date(value[1]);
							var start = startTime.getTime() + 24*60*60*1000; 
							var end_time = endTime.getTime();
							var start_time = startTime.getTime();						
							var curTime = new Date(gloableTime).getTime(); //当前时间戳 当前运营商的时间 ok						
							
							if(end_time > start){
								callback(new Error('<%=rb.getString("ShiJianFanWei")%>'));
							}else if(start_time == end_time){
								callback(new Error('<%=rb.getString("JieShuShiJianXuWanYuKaiShiShiJian")%>'));
							}else if(start_time <= curTime || end_time <= curTime){// 不可选择过去的时间
								callback(new Error('<%=rb.getString("BuKeXuanZeGuoQuDeShiJian")%>'));
							}else{
								callback();
							}
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
					searchText: '',
					timeZone: timeZone

				},
				height:'370px',
				form: { // 表单数据
					taskId: '',
					taskName: '${addTaskName}',
					status: 'off',
					devices: '',
					timeZone: timeZone,
					executeType: 'active',
					quartzTime: [],
				},
				rules: { // 校验规则
					taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					devices:[
						{validator: validateDevice}
					],
					quartzTime:[
						{validator: validateTime}
					]
				},
				showPairGrid: true,
		    	leftUrl : '${ctx}/egw/register/queryEgwPageList.action',
		    	rightUrl : '',
				deviceTitle:['','<%=rb.getString("YiXuan")%>'],
				readonly: false,
				pickerOptions:{
					disabledDate(time){
						return time.getTime() < Date.getNow()-8.64e7;
					}
				},
				operateType:'add',
				limitNum:'5'
			}
		},
		computed: {
			formRules() { // 只读模式置空校验
				return this.readonly? []:this.rules;
			},
			isTimeReadonly(){
				return this.readonly || this.form.executeType != 'timing';
			},
		},
		watch: {
			'form.devices': {
				handler: function(val){
					this.$refs.addform.validateField('devices');
				},
				deep: true
			},
			'form.quartzTime':function(newValue,oldValue){
				if(newValue == null){
					this.form.quartzTime = [];
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
					axios.post('${ctx}/task/cpe/changepwd/getTask.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data){
							Object.assign(vm.form, data);
							initForm(vm.$refs.addform);
						}
					}).catch(function(error){})
					vm.rightUrl = '${ctx}/task/cpe/changepwd/getTaskSelectedList.action?taskId='+id;

				}else{
					
					if(egwLogVue.selection.length > 0){
						vm.$nextTick(function(){
							vm.$refs.egwLogPairgrid.appendCheckedRows(egwLogVue.selection);
						})
					}
				}
				
			},
			loadSuccessDevice(){
				initForm(this.$refs.addform)
			},
			queryDeviceList(val){// 右侧的筛选搜索
				var vm = this;
				vm.queryParams.searchText = val;
			},
			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {// 
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.egwLogPairgrid.getData();
					vm.form.devices = rows.map(function(row){ return row.small_cell_code ;}).sort().join(',');
				})
			},
			// 关闭
			closePanel(){
				var vm = this;

				if(vm.operateType == 'view'){
					eventBus.$emit('hide-egwLog-slide');
				}else{
					vm.cancel();
				}
				
			},
			// 执行方式改变事件
			executeTypeChange(val){
				var vm = this;
				if(val !== 'timing'){
					vm.form.quartzTime = [];
					vm.$refs.addform.clearValidate('quartzTime')
				}
			},
			submit(){ // 提交新增模板数据
				var vm = this,
					params={},
					message = '<%=rb.getString("ChengGong")%>';
				Object.assign(params, vm.form);
				
				vm.$refs.addform.validate(function(valid){
					if(valid){
						axios.post('${ctx}/task/cpe/changepwd/addTask.action',stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message: message,
		    						type:'success',
		    					})
                                eventBus.$emit('hide-egwLog-slide')
		    				}else{
		    					vm.$message.error(data["message"])
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
						eventBus.$emit('hide-egwLog-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-egwLog-slide')
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