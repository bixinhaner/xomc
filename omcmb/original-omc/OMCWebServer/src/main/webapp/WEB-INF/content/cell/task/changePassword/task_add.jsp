<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#cpeChangePasswordAddTask .el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	#cpeChangePasswordAddTask .el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	#cpeChangePasswordAddTask .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#cpeChangePasswordAddTask .titleStyML{
		margin-left: 20px;
	}
	#cpeChangePasswordAddTask .newPasswordCls{
		display: inline-block;
	}
	#cpeChangePasswordAddTask .newPasswordCls input[type="password"]::-ms-reveal{
		display: none;
	}
	#cpeChangePasswordAddTask .deviceTableBox{
		margin:10px 0px 0px 45px;
		display: flex;
	}
	#cpeChangePasswordAddTask .deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	#cpeChangePasswordAddTask .deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	#cpeChangePasswordAddTask .specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	#cpeChangePasswordAddTask .specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	#cpeChangePasswordAddTask .specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	
	#cpeChangePasswordAddTask .changePasswordDeviceTableBox{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	#cpeChangePasswordAddTask .changePasswordDeviceTableBox .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	#cpeChangePasswordAddTask .changePasswordDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	#cpeChangePasswordAddTask .changePasswordDeviceTableBox .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	#cpeChangePasswordAddTask .tableTitles{
		margin-left: 45px;
		padding-top: 20px;
		font-size: 14px;
	}
</style>
<!-- 新建修改密码任务 -->
<div class="flex-ctn" id="cpeChangePasswordAddTask" style="overflow:hidden">
	<%-- <div  id='temp_add_close' class="placeholder-bt" style='position:absolute;top:20px;right:20px;overflow: hidden;' placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div> --%>
	
	<el-form ref="addform" :model="form" :rules="formRules" label-position="left"  :hide-required-asterisk='true'>
		<div class="group-title not-extend titleStyML" style='padding-top: 30px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<!-- 基本信息 -->
		<div style="padding: 20px 45px 0px 45px;">
			<el-form-item prop="taskName" label="<%=rb.getString("RenWuMingCheng")%>"  label-width="120px" style="margin-bottom:20px;">
				<el-input style='width:400px;padding-top:7px;' v-model="form.taskName" :readonly="readonly" placeholder="<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100"></el-input>
			</el-form-item>
			
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		
		<div class="tableTitles">CPEs</div>
		<div class="deviceTableBox">
			<div class="deviceSpecifiedBox">
				<div class="deviceSpecifiedTitle"><%=rb.getString("SheBeiZhiDing")%></div>
				<div class="specifiedTypeBox">
					<el-radio-group  v-model="selectedType" :disabled="readonly" @change="selectAllChange">
						<el-radio  label="true" style="margin-bottom:26px;"><%=rb.getString("QuanBu")%></el-radio>
						<el-radio  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
					</el-radio-group>
				</div>
			</div>
			<div class="changePasswordDeviceTableBox">
				<el-pairgrid
					v-if="showPairGrid && selectedType == 'false'" 
					:id="'select_device_list'" 
					:rownumber="true" 
					style="margin-right:45px;"
					ref="changePasswordPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:row-key="'small_cell_code'" 
					:query-params="queryParams" 
					query-name="mac_address" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("CPEMacAddress")%>'}" 
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
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>"  min-width="140"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>"  min-width="140"></el-table-column>
						<el-table-column prop="pci" label="PCI"  min-width="140"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / PCI"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>"></el-table-column>
						<el-table-column prop='host_name' label='<%=rb.getString("CPEName")%>'></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="200"></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable 
					id="all_device_list" 
					v-if="selectedType == 'true'" 
					style="border:1px solid #E9E9E9;margin-right:45px;" 
					ref="all_device_list"  
					:url="leftUrl" 
					:height="height" 
					front-pagination="true" 
					pagination="true" 
					:query-params="queryParams"
				 >
				 	<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%>"></el-query>
					</template>
					<el-table-column prop="connection_status" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connection_status=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
					<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>"  min-width="140"></el-table-column>
					<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>"  min-width="140"></el-table-column>
					<el-table-column prop="pci" label="PCI"  min-width="140"></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
				</el-ctable>
				<div v-if="!showPairGrid && selectedType == 'false'" >
					<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="selected_device_list"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="queryParams">
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>"  min-width="140"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>"  min-width="140"></el-table-column>
						<el-table-column prop="pci" label="PCI"  min-width="140"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
		<el-form-item style="margin-left:45px;" prop="devices"></el-form-item>
		<div class="alarmBottomLine"></div>
		<!-- 操作类型选择 -->
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("CaoZuoLeiXing")%></span>
		</div>
		<div style="padding: 15px 45px 0px 30px;">
			<div class="el-textarea__inner" style="border:none;position:relative;">
				<el-form-item prop="password" label="<%=rb.getString("XiuGaiMiMa")%>" label-width="160px"  class="newPasswordCls">
					<el-password v-model="form.password" size="mini" placeholder="" show-password :disabled="readonly" style="padding-top: 5px;"></el-password>
					<el-input v-model="form.password" style="display: none;"></el-input>
				</el-form-item>
				
			</div>
		</div>
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
						<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
						<el-radio label="timing" style='margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop='quartzTime' style='display:inline-block;' class='timeItem'>
					<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" :disabled="isTimeReadonly" v-model='form.quartzTime' size="mini" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
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
	var passVue = new Vue({
		el: '#cpeChangePasswordAddTask',
		data(){
			var vm = this;
			var validateDevice = function(rule,value,callback) { // 校验设备
					if(vm.selectedType == 'true'){
						callback();
					}else{
						if(value.length == 0) {
							callback('<%=rb.getString("QingXuanZeSheBei")%>');
						}else {
							callback();
						}
					}
					
				},
				validatePWD = function(rule,value,callback) { // 校验密码
					if(value) {
						if(value.length<5 || value.length>64) { // 长度不符合 5 - 64位
							callback('<%=rb.getString("ZiFuChangDu")%> 5-64');
						}else {
							var reg = /[\u4E00-\u9FA5\uF900-\uFA2D]/; // 中文校验
							if(reg.test(value)) {
								callback('<%=rb.getString("FeiZhongWenZiFu")%>');
							}else {
								callback();
							}
						}
					}else {
						callback('<%=rb.getString("QingShuRuMiMa")%>');
					}
				},
				validateTime = function(rule,value,callback) { // 校验定时时间
					if(vm.form.executeType == 'timing') {
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
					searchText: '',
					timeZone: timeZone

				},
				height:'370px',
				form: { // 表单数据
					taskId: '',
					taskName: '${addTaskName}',
					status: 'off',
					password: '',
					devices: '',
					timeZone: timeZone,
					executeType: 'active',
					quartzTime: '',
				},
				selectedType:'false',
				rules: { // 校验规则
					taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					devices:[
						{validator: validateDevice}
					],
					password:[
						{validator: validatePWD}
					],
					quartzTime:[
						{validator: validateTime}
					]
				},
				showPairGrid: true,
		    	leftUrl : '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
		    	rightUrl : '',
				deviceTitle:['','<%=rb.getString("YiXuan")%>'],
				groupOptions:[],
				versionOptions:[],
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
					this.form.quartzTime = '';
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
							if(vm.form.selectedType == ''){
								vm.selectedType = 'false';
							}else{
								vm.selectedType = 'true';
							}
							initForm(vm.$refs.addform);
						}
					}).catch(function(error){})
					vm.rightUrl = '${ctx}/task/cpe/changepwd/getTaskSelectedList.action?taskId='+id;

				}else{
					
					if(cpeChangePasswordVue.selection.length > 0){
						vm.$nextTick(function(){
							vm.$refs.changePasswordPairgrid.appendCheckedRows(cpeChangePasswordVue.selection);
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
			setTime(){
				this.form.quartzTime = formatDate(new Date(gloableTime));
				this.$refs.addform.validateField('quartzTime');
			},
			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {// 
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.changePasswordPairgrid.getData();
					vm.form.devices = rows.map(function(row){ return row.small_cell_code ;}).sort().join(',');
				})
			},
			// 关闭
			closePanel(){
				var vm = this;

				if(vm.operateType == 'view'){
					eventBus.$emit('hide-changePassword-slide');
				}else{
					vm.cancel();
				}
				
			},
			// 执行方式改变事件
			executeTypeChange(val){
				var vm = this;
				if(val !== 'timing'){
					vm.form.quartzTime = '';
					vm.$refs.addform.clearValidate('quartzTime')
				}
			},
			submit(){ // 提交新增模板数据
				var vm = this,
					params={},confirmStr='<%=rb.getString("QueRenXiuGaiMiMa")%>',
					message = '<%=rb.getString("ChengGong")%>';
                // 防止多次提交
                if(cpeChangePasswordVue.slideSubmitLoading)return

				Object.assign(params, vm.form);
				if(vm.selectedType == 'false'){
					delete params.selectedType
				}else{
					params.selectedType = 'all';
					params.devices = '';
				}
				vm.$refs.addform.validate(function(valid){
					if(valid){
						
						vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
                            cpeChangePasswordVue.slideSubmitLoading = true;
							axios.post('${ctx}/task/cpe/changepwd/addTask.action',stringify(params)).then(function(response){
								var data = response.data;
			    				if(data["success"]){
			    					vm.$message({
			    						message: message,
			    						type:'success',
			    					})
                                    eventBus.$emit('hide-changePassword-slide')
			    				}else{
			    					vm.$message.error(data["message"]);
                                    cpeChangePasswordVue.slideSubmitLoading = false;
			    				}
							}).catch(function(error){})
						}).catch(() => {})
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
						eventBus.$emit('hide-changePassword-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-changePassword-slide')
				}
			},
		
			// 设备执行类别  1 全部执行 2 指定执行
			selectAllChange(val){
				var vm = this;

				vm.$refs.addform.validateField('devices');
				vm.queryParams.search_text = '';
				if(val == '1'){
					vm.rightUrl = '';
				}else{
					vm.rightUrl="${ctx}/cell/password/getTaskSelectedList.action?taskId="+vm.taskId+"&timeZone="+timeZone;
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