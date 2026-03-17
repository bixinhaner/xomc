<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#gnbChangePasswordAddTask .el-icon-arrow-up:before {
		line-height: 1;
	}
	
	#gnbChangePasswordAddTask .el-input__inner[readonly] {
		background-color: #f5f7fa;
	}
	#gnbChangePasswordAddTask .el-select .el-input__inner[readonly] {
		background-color: #fff;
	}
	
	#gnbChangePasswordAddTask .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#gnbChangePasswordAddTask .titleStyML{
		margin-left: 20px;
	}
	#gnbChangePasswordAddTask .newPasswordCls{
		display: inline-block;
	}
	#gnbChangePasswordAddTask .newPasswordCls input[type="password"]::-ms-reveal{
		display: none;
	}
	#gnbChangePasswordAddTask .deviceTableBox{
		margin:10px 0px 0px 45px;
		display: flex;
	}
	#gnbChangePasswordAddTask .deviceSpecifiedBox{
		width: 200px;
		height: 370px;
		border:1px solid #E9E9E9;
		border-right: none;
	}
	#gnbChangePasswordAddTask .deviceSpecifiedTitle{
		height: 36px;
		width: 200px;
		font-size: 12px;
		line-height: 36px;
		text-align: center;
		background: #F6F7FB;
		box-sizing: border-box;
		border-bottom:1px solid #E9E9E9;
	}
	#gnbChangePasswordAddTask .specifiedTypeBox{
		flex: 1;
		padding-top: 30px;
		padding-left: 40px;
	}
	#gnbChangePasswordAddTask .specifiedTypeBox .el-radio__label{
		font-size: 12px !important;
	}
	#gnbChangePasswordAddTask .specifiedTypeBox .el-radio+.el-radio{
		margin-left: 0px;
		display: block;
	}
	#gnbChangePasswordAddTask .productBoxCls .el-input__inner{
		height: 26px !important;
	}
	#gnbChangePasswordAddTask .changePasswordDeviceTableBox{
		width: calc(100% - 200px)!important;
		font-size: 12px !important;
	}
	#gnbChangePasswordAddTask .changePasswordDeviceTableBox .pairgrid-right{
		top:40px!important;
		height: calc(100% - 40px)!important;
	}
	#gnbChangePasswordAddTask .changePasswordDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
	#gnbChangePasswordAddTask .changePasswordDeviceTableBox .transition-box .el-form-item{
		display: inline-block;
		margin-right: 30px;
	}
	#gnbChangePasswordAddTask .tableTitles{
		margin-left: 45px;
		font-size: 14px;
	}
</style>

<div class="flex-ctn" id="gnbChangePasswordAddTask" style="overflow:hidden">
	<%-- <div  id='temp_add_close' class="placeholder-bt" style='position:absolute;top:20px;right:20px;overflow: hidden;' placeholder="<%=rb.getString("GuanBi")%>">		
		<span class="el-icon el-icon-circle-close" @click="closePanel"></span>
	</div> --%>
	
	<el-form ref="addform" :model="form" :rules="formRules" label-position="left"  :hide-required-asterisk='true'>
		<div class="group-title not-extend titleStyML" style="padding-top: 20px;">
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
		
		<el-form-item class="productBoxCls" prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>"  label-width="120px" style="margin-bottom:20px;padding: 20px 45px 0px 45px;">
			<el-select v-model='productVal' style="padding-top:7px;" @change="productValChange" :disabled="readonly">	
				<el-option v-for='item in productTypeList' :label="item.name" :value="item.name" :key="item.name"></el-option>
			</el-select>
		</el-form-item>
		
		<div class="tableTitles">gNBs</div>
		<div class="deviceTableBox">
			<div class="deviceSpecifiedBox">
				<div class="deviceSpecifiedTitle"><%=rb.getString("SheBeiZhiDing")%></div>
				<div class="specifiedTypeBox">
					<el-radio-group  v-model="form.selectAll" :disabled="readonly" @change="selectAllChange">
						<el-radio  label="true" style="margin-bottom:26px;"><%=rb.getString("QuanBu")%></el-radio>
						<el-radio  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
					</el-radio-group>
				</div>
			</div>
			<div class="changePasswordDeviceTableBox">
				<el-pairgrid
					v-if="showPairGrid && form.selectAll == 'false'" 
					:id="'gnb_select_device_list'" 
					:rownumber="true" 
					style="margin-right:45px;"
					ref="changePasswordPairgrid" 
					:right-url="rightUrl" 
					:left-url="leftUrl" 
					:height="'100%'" 
					:row-key="'small_cell_code'" 
					:query-params="queryParams" 
					:title="deviceTitle" 
					:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" 
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
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("HostName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</template>
					<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>"></el-query>
					</template>
					<template slot='right'>
						<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
						<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable 
					id="gnb_all_device_list" 
					v-if="form.selectAll == 'true'" 
					style="border:1px solid #E9E9E9;margin-right:45px;" 
					ref="all_device_list"  
					:url="leftUrl"
					:row-key="'small_cell_code'" 
					:height="height" 
					front-pagination="true" 
					pagination="true" 
					:query-params="queryParams"
				 >
				 	<template slot='toolbar'>
						<el-query type="normal" @query="queryDeviceList" placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>"></el-query>
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
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("HostName")%>" min-width="250" ></el-table-column>
					<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
				</el-ctable>
				<div v-if="!showPairGrid && form.selectAll == 'false'" >
					<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="selected_device_list"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="queryParams">
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("HostName")%>" min-width="250" ></el-table-column>
						<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
		<el-form-item style="margin-left:45px;" prop="snList"></el-form-item>
		<div class="alarmBottomLine"></div>
		<!-- 操作类型选择 -->
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("CaoZuoLeiXing")%></span>
		</div>
		<div style="padding: 15px 45px 0px 30px;">
			<div class="el-textarea__inner" style="border:none;position:relative;">
				<el-form-item style='display:inline-block;margin:10px 0px' prop='executeType'>
					<el-radio-group v-model="form.taskType" :disabled='readonly' @change="taskTypeChange">
						<el-radio v-if="false" label="reset" style='margin-right:80px;' :disabled="taskTypeShow"><%=rb.getString("MiMaChongZhi")%></el-radio>
						<el-radio label="update" style='margin-right:10px;'><%=rb.getString("XiuGaiMiMa")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop="newPassword" label-width="0px" class="newPasswordCls">
					<el-password v-model="form.newPassword" size="mini" placeholder="" show-password :disabled="form.taskType !== 'update' || readonly" style="padding-top: 5px;"></el-password>
					<el-input v-model="form.newPassword" style="display: none;"></el-input>
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
						<el-radio label="wait" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
						<el-radio label="timing" style='margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
					</el-radio-group>
				</el-form-item>
				<el-form-item prop='time' style='display:inline-block;' class='timeItem'>
					<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" :disabled="isTimeReadonly" v-model='form.time' size="mini" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
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
		el: '#gnbChangePasswordAddTask',
		data(){
			var vm = this;
			var validateDevice = function(rule,value,callback) { // 校验设备
					if(vm.form.selectAll == 'true'){
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
					if(vm.form.taskType == 'reset'){
						callback();
					}else{
						if(value) {
							/*
							if(value.length<8 || value.length>20) { // 长度不符合 5 - 64位
								callback('<%=rb.getString("ZiFuChangDu")%> 8-20');
							}else {
								var reg = /[\u4E00-\u9FA5\uF900-\uFA2D]/; // 中文校验
								var pwdReg = /^([a-zA-Z0-9(\~)(\@)(\#)(\_)(\^)(\*)(\%)(\/)(\.)(\+)(\:)(\;)(\=)]+)$/; // 空格校验
								if(reg.test(value)) {
									callback('<%=rb.getString("FeiZhongWenZiFu")%>');
								}else if(pwdReg.test(value)) {
									var inReg = /[~@#_^*%/.+:;=]/g;
									if(value.match(inReg)) {
										callback();
									}else {
										callback('Must contain one character in [~@#_^*%/.+:;=]');
									}
								}else {
									callback('Rule: 0-9,a-z,A-Z,~@#_^*%/.+:;=');
								}
							}
							*/
							boolMap = checkPasswordSpecialRule(value);
							
							if(boolMap.threeValid === false) {
								callback('<%=rb.getString("MiMa3LeiTiShi")%>');
							}else if(boolMap.orderValid === false) {
								callback('<%=rb.getString("MiMaZiFuLianXuTiShi")%>');
							}else if(boolMap.repeatValid === false) {
								callback('<%=rb.getString("MiMaZiFuChongFuTiShi")%>');
							}else if(boolMap.yearValid === false) {
								callback('<%=rb.getString("MiMaNianFenTiShi")%>');
							}else {
								callback();
							}
						}else {
							callback('<%=rb.getString("QingShuRuMiMa")%>');
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
					search_text: '',
					productValue:'',
					isUpdatePW:'true',
					isGnb: 1
				},
				height:'370px',
				form: { // 表单数据
					taskId: '',
					taskName: '${addTaskName}',
					status: 'off',
					newPassword: '',
					snList: '',
					timeZone: timeZone,
					executeType: 'active',
					taskType:'update',
					selectAll:'false',
					time: '',
					productType:'FAP/\\w*BSC\\w+',
				},
				rules: { // 校验规则
					taskName:[
						{required: true,message:'<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'},
						{type:'string',max: 100,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 100'}
					],
					snList:[
						{validator: validateDevice}
					],
					newPassword:[
						{validator: validatePWD}
					],
					time:[
						{validator: validateTime}
					]
				},
				showPairGrid: true,
		    	leftUrl : '',
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
				productTypeList:[],
				taskTypeShow:false,
				//productVal:'FAP/\\w*BSC\\w+',
				productVal: '',
				operateType:'add',
				oldProductVal:'',
				addProductVal:''
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
			'form.snList': {
				handler: function(val){
					this.$refs.addform.validateField('snList');
				},
				deep: true
			},
			'form.time':function(newValue,oldValue){
				if(newValue == null){
					this.form.time = '';
				}
			},
			'productVal':function(newValue,oldValue){
				if(this.productTypeList.length == 0){
					return
				}
				var selectData;
				selectData = this.productTypeList.find((item)=>{
					return item.name == newValue
				});
								
				this.queryParams.productValue = selectData.value;
				this.form.productType = selectData.value;
				var strNum = Math.random().toString();
				this.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action?random='+strNum;
				//目前只有修改密码
				/*if(selectData.updatePWType == 'update'){
					this.taskTypeShow = true;
					this.form.taskType = 'update';
				}else{
					this.form.taskType = 'reset';
					this.taskTypeShow = false;
				}*/
			},
		},
		methods: {
			init(id,operateType,productVal) {// 初始化任务信息
				var vm = this,
					params = {taskId: id, timeZone: timeZone};
				vm.taskId = id;
				vm.operateType = operateType;
				vm.readonly = vm.operateType == 'view' ? true : false;
				vm.addProductVal = productVal;
				vm.getProductType();

				if(vm.operateType == 'view'){
					vm.showPairGrid = false;
				}
				if(vm.operateType != 'add'){
					axios.post('${ctx}/cell/password/getCellUpdatePwdTaskInfos.action',stringify(params)).then(function(response){
						var data = response.data;
						if(data){
							Object.assign(vm.form, data);
							if(vm.form.selectAll == ''){
								vm.form.selectAll = 'false';
							}
							vm.form.snList = data.snList.sort().toString();
							vm.productVal = data.productType;
							vm.oldProductVal = data.productType;
							initForm(vm.$refs.addform);
						}
					}).catch(function(error){})
					vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action';
					vm.rightUrl = '${ctx}/cell/password/getTaskSelectedList.action?taskId='+id;

				}else{
					if(changePasswordGnbVue.selection.length > 0){
						vm.$nextTick(function(){
							vm.$refs.changePasswordPairgrid.appendCheckedRows(changePasswordGnbVue.selection);
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
				this.form.time = formatDate(new Date(gloableTime));
				this.$refs.addform.validateField('time');
			},
			/**
			 * 设备选择变化时，更新选择设备记录
			 * @param value:
			*/
			devicesChange(value) {// 
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.changePasswordPairgrid.getData()
					vm.form.snList = rows.map(function(row){ return row.small_cell_code ;}).sort().join(',');
				})
			},
			// 关闭
			closePanel(){
				var vm = this;

				if(vm.operateType == 'view'){
					eventBus.$emit('hide-changePassword-slide-gnb');
				}else{
					vm.cancel();
				}
				
			},
			// 操作类型改变事件
			taskTypeChange(val){
				var vm = this;
				if(val !== 'update'){
					vm.form.newPassword = '';
					vm.$refs.addform.clearValidate('newPassword')
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
			submit(){ // 提交新增模板数据
				var vm = this;
				var message = '<%=rb.getString("ChengGong")%>';
                // 防止多次提交
                if(changePasswordGnbVue.slideSubmitLoading)return

				vm.$refs.addform.validate(function(valid){
					if(valid){
						var params = {};

						Object.assign(params, vm.form)
						params.newPassword = AesEncrypt(params.newPassword);
                        changePasswordGnbVue.slideSubmitLoading = true;
						axios.post('${ctx}/cell/password/addCellUpdatePwdTask.action?isGnb=1',stringify(params)).then(function(response){
							var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
		    						message: message,
		    						type:'success',
		    					})
                                eventBus.$emit('hide-changePassword-slide-gnb');
		    				}else{
		    					vm.$message.error(data["message"]);
                                changePasswordGnbVue.slideSubmitLoading = false;
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
						eventBus.$emit('hide-changePassword-slide-gnb')
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-changePassword-slide-gnb')
				}
			},
			// 获取产品类型
			getProductType(){
				var vm = this,params={isUpdatePW:"true", isGnb: 1};
				
				axios.post('${ctx}/cell/version/getProductType.action',stringify(params)).then(function(response){
					vm.productTypeList = (response.data||[]).filter(function(item){
						return item.name != 'BaiBNX';
					});
					if(vm.operateType == 'add'){
						vm.productVal = vm.addProductVal;
					}
				}).catch(function(error){
					
				})
			},
			// 类型改变
			productValChange(){
				var vm = this;
				if(vm.oldProductVal != vm.productVal){
					vm.$refs.changePasswordPairgrid.clear();
				}else{
					var str = Math.random().toString();
					this.rightUrl = '${ctx}/cell/password/getTaskSelectedList.action?taskId='+vm.taskId+ "&randomValue="+str;
				}
			},
			// 设备执行类别  1 全部执行 2 指定执行
			selectAllChange(val){
				var vm = this;

				vm.$refs.addform.validateField('snList');
				vm.queryParams.search_text = '';
				if(val == '1'){
					vm.rightUrl = '';
				}else{
					vm.rightUrl="${ctx}/cell/password/getTaskSelectedList.action?taskId="+vm.taskId+"&timeZone="+timeZone;
				}
			},
		},
		created(){
			//this.getProductType();
		},
		mounted(){
			eventBus.$off('action-save-gnb').$on('action-save-gnb',this.submit);
			eventBus.$off('action-init-gnb').$on('action-init-gnb',this.init);
			eventBus.$off('cancel-slide-gnb').$on('cancel-slide-gnb',this.cancel);
		}
	});
</script>