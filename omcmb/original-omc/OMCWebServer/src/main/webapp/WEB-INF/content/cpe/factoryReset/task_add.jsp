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
		margin-left: 50px;
	}
	.deviceTableBox{
		margin:20px 0px 0px 76px;
		display: flex;
		height:372px;
		background:#FFFFFF;
	}
	.deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	.deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	.specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	.specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	.specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	.productBoxCls .el-input__inner{
		height: 26px !important;
	}
	.changeDeviceTableWarp{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	.changeDeviceTableWarp .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	.changeDeviceTableWarp .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	.changeDeviceTableWarp .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	.tableTitles{
		margin-left: 45px;
		padding-top: 20px;
		font-size: 14px;
	}
	.el-radio__input.is-checked+.el-radio__label,
	.el-radio{
		color:#333333;
	}
	.footer{
		width:100%;
		border-top:1px solid #E9E9E9;
		position:absolute;
		bottom:0px;
		height:50px;
		line-height:50px;
		background:#FFFFFF;
		z-index:99;
	}
	.footer div{
		padding-left:40px;
	}
	.el-form-item__error{
		padding-top:0px;
	}
	.closeSlideBtn{
		position:absolute;
		top:20px;
		right:20px;
		overflow: hidden;
	}
	.mainWarp{
		overflow:hidden;
	}
	.basicBox{
		padding-top:40px;
	}
	.basicInfo{
		padding: 20px 76px 0;
	}
	.allSelectTable{
		border:1px solid #E9E9E9;
		margin-right:45px;
	}
	.testCpeCode{
		margin-left:76px;
		margin-top:6px;
	}
	.executeTypeWarp{
		padding: 20px 60px 80px;
	}
</style>
<div class="flex-ctn mainWarp" id="factoryResetAddTask">
	<div  id='temp_add_close' class="placeholder-bt closeSlideBtn" placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div>
	<el-form ref="addform" :model="form" :rules="formRules" label-position="left" :hide-required-asterisk='true'>
		<div class="group-title not-extend titleStyML basicBox">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<!-- 基本信息 -->
		<div class="basicInfo">
			<el-form-item prop="taskName" label="<%=rb.getString("RenWuMingCheng")%>" label-width="120px" style="margin-bottom:20px;">
				<el-input style='width:400px;padding-top:7px;' v-model="form.taskName" :readonly="readonly" placeholder="<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100"></el-input>
			</el-form-item>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>

		<div class="deviceTableBox">
			<div class="deviceSpecifiedBox">
				<div class="deviceSpecifiedTitle"><%=rb.getString("ZhiDingSheBeiZhiXing")%></div>
				<div class="specifiedTypeBox">
					<el-radio-group v-model="selectAll" :disabled="readonly" @change="selectAllChange">
						<el-radio label="1" style="margin-bottom:26px;"><%=rb.getString("QuanBu")%></el-radio>
						<el-radio label="0"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
					</el-radio-group>
				</div>
			</div>
			<div class="changeDeviceTableWarp">
				<el-pairgrid
					v-if="showPairGrid && selectAll == '0'" 
					:id="'select_device_list'" 
					:rownumber="true" 
					style="margin-right:45px;"
					ref="deviceListPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:row-key="'small_cell_code'" 
					:query-params="queryParams" 
					query-name="mac_address" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("CPEMacAddress")%>'}" 
					@selection-change="devicesChange" 
					@right-load-success="loadSuccessDevice">
					<template slot="left">
						<el-table-column type='selection' width="50" align="center"></el-table-column>					
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="140"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="140"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='host_name' label='<%=rb.getString("CPEName")%>'></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" ></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable 
					id="all_device_list" 
					v-if="selectAll == '1'" 
					class="allSelectTable" 
					ref="all_device_list"  
					:url="leftUrl" 
					:height="height" 
					front-pagination="true" 
					pagination="true" 
					:query-params="queryParams">
				 	<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%>"></el-query>
					</template>
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
					<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="140"></el-table-column>
					<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="140"></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
				</el-ctable>
				
				<div v-if="!showPairGrid && selectAll == '0'">
					<el-ctable :id="'selected_device_list'" ref="selected_device_list"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="queryParams" class="allSelectTable">
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>" min-width="140"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="140"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
		<el-form-item class="testCpeCode" prop="cpeCodes"></el-form-item>

		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<!-- 执行方式 -->
		<div class="executeTypeWarp">
			<div class="el-textarea__inner" style="border:none;">
				<el-form-item style='display:inline-block;' prop='executeType'>
					<el-radio-group v-model="form.executeType" :disabled='readonly' @change="executeTypeChange">
						<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
						<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
						<el-radio label="timing" style='margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop='time' style='display:inline-block;' class='timeItem'>
					<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" :disabled="isTimeReadonly" v-model='form.time' size="mini" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
				</el-form-item>
			</div>
		</div>
	</el-form>
	<div class='footer'>	
		<div>
			<el-button :disabled="readonly" type="primary" @click="submit"><%=rb.getString("QueDing")%></el-button>
			<el-button :disabled="readonly" @click="cancel"><%=rb.getString("QuXiao")%></el-button>
			<el-checkbox v-if="false" :disabled="readonly" v-model="keepConfig" :true-label="'1'" :false-label="'0'" style='margin-left:30px;'><%=rb.getString("BaoLiuPeiZhi")%></el-checkbox>
		</div>		
	</div>
</div>

<script type="text/javascript">
	/**
	*  页面编辑和只读模式通过readonly控制
	*  校验规则也由readonly决定
	**/
	var factoryResetAddTaskVue = new Vue({
		el: '#factoryResetAddTask',
		data(){
			var vm = this;
			var validateDevice = function(rule,value,callback) { // 校验设备
					if(vm.selectAll == '1'){
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
				// 表单数据
				form: { 
					timeZone: timeZone,
					taskName: '${addTaskName}',
					cpeCodes: '',
					executeType: 'active',
					time: '',					
					taskId: '',				
				},
				keepConfig:'0',
				selectAll:'0',
				// 校验规则
				rules: { 
					taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					cpeCodes:[
						{validator: validateDevice}
					],
					
					time:[
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
				isPswModel: true,
				operateType:'add',
				selection:[]
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
			'form.cpeCodes': {
				handler: function(val){
					this.$refs.addform.validateField('cpeCodes');
				},
				deep: true
			},
			'form.time':function(newValue,oldValue){
				if(newValue == null){
					this.form.time = '';
				}
			},
		},
		methods: {
			// 初始化任务信息
			init(row,operateType) {
				var vm = this,
				params = {taskId: row.TASK_ID, timeZone: timeZone};
			
				vm.taskId = row.TASK_ID;
				vm.operateType = operateType;
				
				vm.readonly = vm.operateType == 'information' ? true : false;
				
				if(vm.operateType == 'information'){
					vm.showPairGrid = false;
				}

				if(vm.operateType != 'add'){
					axios.post('${ctx}/cpe/factoryReset/getTaskInfo.action',stringify(params)).then(function(response){
						var data = response.data;		
						if(data){
							vm.form.taskName = data.TASK_NAME;
							vm.form.executeType = data.EXECUTE_TYPE;
							vm.form.time = data.TIME;
							vm.keepConfig = data.KEEP_CONFIG;
							if(data.SELECT_ALL == '0'){
								vm.selectAll = '0';
							}else{
								vm.selectAll = '1';
							}
						}
					}).catch(function(error){})
					
					if(vm.operateType == 'modify'){
						vm.rightUrl="${ctx}/cpe/factoryReset/getSelectedCPEInfo.action?taskId="+vm.taskId+"&type="+vm.operateType;
						if(row && row.length) {
							vm.$refs.deviceListPairgrid.appendCheckedRows(row);
						}
					}else if(vm.operateType == 'information'){
						vm.rightUrl="${ctx}/cpe/factoryReset/getSelectedCPEInfo.action?taskId="+vm.taskId;
					}
				}else{					
					if(factoryResetTaskVue.selection.length > 0){
						vm.$nextTick(function(){
							vm.$refs.deviceListPairgrid.appendCheckedRows(factoryResetTaskVue.selection);
						})
					}
				}  				
			},
			loadSuccessDevice(){
				initForm(this.$refs.addform)
			},
			// 右侧的筛选搜索
			queryDeviceList(val){
				var vm = this;
				vm.queryParams.searchText = val;
			},
			setTime(){
				this.form.time = formatDate(new Date(gloableTime));
				this.$refs.addform.validateField('time');
			},
			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.deviceListPairgrid.getData();
					vm.form.cpeCodes = rows.map(function(row){ return row.small_cell_code ;}).sort().join(',');
				})
			},
			
			// 关闭
			closePanel(){
				var vm = this;
				if(vm.operateType == 'information'){
					eventBus.$emit('hide-slide');
				}else{
					vm.cancel();
				}				
			}, 
			// 执行方式改变事件
			executeTypeChange(val){
				var vm = this;
				if(val !== 'timing'){
					vm.form.time = '';
					vm.$refs.addform.clearValidate('time')
				}
			},
			// 提交新增模板数据
			submit(){ 
				var vm = this,
					params={},
					message = '<%=rb.getString("ChengGong")%>';
				Object.assign(params, vm.form);
				//1 all，此时 cpeCodes 参数为 空, 0:指定，cpeCodes 以逗号分隔；
				if(vm.selectAll == '0'){
					params.selectAll = '0';
				}else{
					params.selectAll = '1';
					params.cpeCodes = '';
				}
				
				params.keepConfig = vm.keepConfig;
				//taskId 在修改时，点击保存传递该参数； 新建时，此参数为空
				if(vm.operateType == 'modify'){
					params.taskId = vm.taskId;
				}else if(vm.operateType == 'add'){
					params.taskId = '';
				}
				
				vm.$refs.addform.validate(function(valid){
					if(valid){
						axios.post('${ctx}/cpe/factoryReset/addTask.action',stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message: message,
		    						type:'success',
		    					})
                                eventBus.$emit('hide-slide')
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
						eventBus.$emit('hide-slide')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-slide')
				}
			},
		
			// 设备执行类别  1 全部执行 0 指定执行
			selectAllChange(val){
				var vm = this;
				
				vm.queryParams.searchText = '';
				if(val == '1'){
					vm.rightUrl = '';
				}else{
					vm.$refs.addform.validateField('cpeCodes');
					if(vm.operateType == 'modify'){
						vm.rightUrl="${ctx}/cpe/factoryReset/getSelectedCPEInfo.action?taskId="+vm.taskId+"&type="+vm.operateType;
					}else if(vm.operateType == 'information'){
						vm.rightUrl="${ctx}/cpe/factoryReset/getSelectedCPEInfo.action?taskId="+vm.taskId;
					}					
				}
			},
		},
		created(){},
		mounted(){
			eventBus.$off('modify-task').$on('modify-task',this.init);
		}
	});
</script>