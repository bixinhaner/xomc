<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
    #cpeLogAddTask{
        padding-top: 20px;
    }
	#cpeLogAddTask .el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	#cpeLogAddTask .el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	#cpeLogAddTask .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#cpeLogAddTask .titleStyML{
		margin-left: 20px;
	}
	#cpeLogAddTask .deviceTableBox{
		margin:10px 0px 0px 45px;
		display: flex;
        height: 400px;
	}
	#cpeLogAddTask .cpeLogDeviceTableBox{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	#cpeLogAddTask .cpeLogDeviceTableBox .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	#cpeLogAddTask .cpeLogDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	#cpeLogAddTask .cpeLogDeviceTableBox .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	#cpeLogAddTask .tableTitles{
		margin-left: 45px;
		padding-top: 20px;
		font-size: 12px;
	}
	#cpeLogAddTask .deviceTip {
		position:relative;
	}
	#cpeLogAddTask .deviceTip .deviceNum-tip {
		font-size:12px;
		color:#4D84FF;
		position:absolute;
		top:39px;
		left:60px;
	}
</style>
<div class="flex-ctn" id="cpeLogAddTask" style="overflow:hidden">
	<el-form ref="addform" :model="ruleForm" :rules="formRules" label-position="left"  :hide-required-asterisk='true'>
		<div class="group-title not-extend titleStyML deviceTip">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			<span class="deviceNum-tip">(<%=rb.getString("ZuiDuoXuanZe")%> 20 <%=rb.getString("ZuiDuoXuanZeDevice")%>)</span>
		</div>
		
		<div class="tableTitles">CPE</div>
		<div class="deviceTableBox">
			<div class="cpeLogDeviceTableBox">
				<el-pairgrid
					:id="'select_device_list'" 
					:rownumber="true" 
					style="margin-right:45px;"
					ref="cpeLogPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:limit="20"
					:row-key="'mac_address'"
					:query-params="queryParams" 
					query-name="serial_number" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("CPEBianMa")%>'}" 
					@selection-change="devicesChange" 
					@right-load-success="loadSuccessDevice"
				 >
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
                        <el-table-column prop="connection_status" width="50">
                            <template slot-scope="scope">
                                <div :class="{
                                    'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
                                    '':scope.row.have_connected==2,
                                    'conn_exc':scope.row.connection_status=='Exception',
                                    'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
                            </template>
                        </el-table-column>
                        <el-table-column prop='serial_number' label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
                        <el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="160" ></el-table-column>
                        <el-table-column prop="pci" label="PCI" min-width="100" ></el-table-column>
                        <el-table-column prop='host_name' label='<%=rb.getString("CPEName")%>'></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / PCI"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop="serial_number" label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="160" ></el-table-column>
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
				<el-form-item style='display:inline-block;' prop='status'>
					<el-radio-group v-model="ruleForm.status" :disabled='readonly' @change="executeTypeChange">
						<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
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
	var cpeLogAddTaskVue = new Vue({
		el: '#cpeLogAddTask',
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
				queryParams: {
					search_text: '',
					timeZone: timeZone

				},
				ruleForm: { // 表单数据
					status: 'active',
					devices: '',
					timeZone: timeZone,
					exetime: '',
				},
				rules: { // 校验规则
					devices:[
						{validator: validateDevice}
					],
					exetime:[
						{validator: validateTime}
					]
				},
		    	leftUrl : '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
		    	rightUrl : '',
				deviceTitle:['','<%=rb.getString("YiXuan")%>'],
				readonly: false,
				pickerOptions:{
					disabledDate(time){
						return time.getTime() < Date.now()-8.64e7;
					}
				},
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
					var rows = vm.$refs.cpeLogPairgrid.getData();
					vm.ruleForm.devices = rows.map(function(row){ return row.cpe_code;}).sort().join(',');
				})
			},
			// 关闭
			closePanel(){
				var vm = this;
				vm.cancel();
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
					params={
						timeZone:timeZone,
						isReboot:false,
						device_code:vm.ruleForm.devices,
						device_type:'CPE',
						execute_type:'Immediately',
						start_time:'',
					};
				// 防止多次提交
                if(cpeLogs.slideSubmitLoading)return

				if(vm.ruleForm.status == 'timing'){
					params.start_time = vm.ruleForm.exetime;
				}
				vm.$refs.addform.validate(function(valid){
					if(valid){
                        cpeLogs.slideSubmitLoading = true;
						axios.post('${ctx}/cell/collect/goImmediateCollectLogFile.action',stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message:  '<%=rb.getString("ChengGong")%>',
		    						type:'success',
		    					})
								eventBus.$emit('hide-cpeLog-slide')
		    				}else{
		    					vm.$message.error(data["message"]);
                                cpeLogs.slideSubmitLoading = false;
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
						eventBus.$emit('hide-cpeLog-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-cpeLog-slide')
				}
			},
		},
		created(){},
		mounted(){
			eventBus.$off('cpe-log-addSubmit').$on('cpe-log-addSubmit',this.submit);
			eventBus.$off('cpe-log-addCancel').$on('cpe-log-addCancel',this.cancel);
		}
	});
</script>