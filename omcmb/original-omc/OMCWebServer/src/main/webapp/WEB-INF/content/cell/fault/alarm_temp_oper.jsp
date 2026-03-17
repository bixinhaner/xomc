<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.el-form-item{
		margin-left:40px;
	}
	.headContent{
		font-size:14px;
		color:#0F344D;
		font-weight:700;
		margin-top:30px;
		margin-bottom:20px;
		margin-left:20px;
	}
	.headContent img{
		vertical-align:top;
		margin-right:14px;
	}
	.boxBorderCon{
		width:680px;
		height:27px;
		padding-top:13px;
		border:1px solid #DEDFE6;
	}
	.bottomline{
		width:80%;
		height:1px;
		background:#E9E9E9;
		margin-top:35px;
	}
	.el-input__prefix{
		top:-5px;
	}
	.el-select .el-input .el-select__caret{
		line-height:28px;
	}
	.boxContainerAlarmType .el-checkbox{
		width:180px;
	}
	.demonstration{
		display:block;
		margin-bottom:10px;
	}
	.alarmSpan{
		display:inline-block;
		height:21px;
		line-height:24px;
		margin-left:8px;
	}
	.el-icon-search{
		margin-left:0px !important;
	} 
	.boxBorderCon .el-icon-circle-close{
		line-height:20px;
	}
	.minor .el-icon-status-alarm:before{
		color:#D0D53B;
	}
	.major .el-icon-status-alarm:before{
		color:#FB9F50;
	}
	.critical .el-icon-status-alarm:before{
		color:#FF7B7B;
	}
	.warning .el-icon-status-alarm:before{
		color:#67DFF8;
	}
	.cycleOptions{
		min-height: 28px;
		max-height: 28px;
	}
	.cycleOptions .el-input__inner{
		min-height: 28px !important;
		max-height: 28px !important;
	}
</style>
<div id="addNotificationPage" style="padding:0 40px;">
	<el-form :model="ruleForm" :rules="rules" ref="ruleForm" label-position="top">
		<div class="headContent">
			<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("JiBenXinXi") %>
		</div>
		<el-form-item label="<%=rb.getString("MuBanMingCheng")%>" prop="temp_name">
			<el-input ref="taskNameInput" maxLength=50 :disabled="viewFlag" v-model="ruleForm.temp_name" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
			<el-input type="textarea" :disabled="viewFlag" :rows="3" v-model="ruleForm.desc" maxlength="500" style="width:680px;" >
		</el-form-item>
		<el-form-item prop='state' label='<%=rb.getString("ZhuangTai") %>'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.state' :disabled='viewFlag'>
					<el-radio label="1" style="margin-left:21px"><%=rb.getString("QiYong") %></el-radio>
					<el-radio label="0" style="margin-left:223px"><%=rb.getString("JinYong") %></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<el-form-item label="<%=rb.getString("TongZhiZhouQi") %><%=rb.getString("FenZhong")%>" prop='cycle'>
			<el-select v-model="ruleForm.cycle" placeholder="" :disabled='viewFlag'  class="cycleOptions">
					<el-option v-for="item in cycleOptions" :key="item.id" :label="item.text" :value="item.id">
					</el-option>
			</el-select>
		</el-form-item>
		<div class="bottomLine"></div>
		<div class='headContent'>
			<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("TiaoJianSheZhi")%>
		</div>
		<el-form-item label='<%=rb.getString("SheBeiLeiXing") %>' prop='device_type'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.device_type' :disabled='viewFlag' @change="deviceTypeChange">>
					<!-- <el-radio label="eNB" style="margin-left:21px">eNB</el-radio> -->
					<el-radio v-for="(item,index) in initAlarmTypeArr" :key="index"  :label="item.toUpperCase()" style="margin-left:21px" >{{item}}</el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<el-form-item label='<%=rb.getString("SheBeiXuanZe") %>' v-if="ruleForm.device_type=='ENB'" prop='device_select_mode'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.device_select_mode' :disabled='viewFlag'>
					<el-radio label="1" style="margin-left:21px"><%=rb.getString("SheBeiZu") %></el-radio>
					<el-radio label="0" style="margin-left:21px"><%=rb.getString("SheBeiLieBiao") %></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<!-- 设备组列表 -->
		<el-form-item v-if="ruleForm.device_select_mode == '1' && ruleForm.device_type=='ENB'">
			<el-ctable height='300px' @load-success="tableLoadSuccess('ENBZ')" :readonly='viewFlag' ref="deviceGroupTable" :default-checked="deviceChecked" row-key="id" :rownumber="true" pagination="true" :page-list="pageList":query-params="queryDeviceForm" :url="deviceGroupUrl" style="width:60%;border:1px solid #E9E9E9;" @selection-change='changeDeviceGroupSelect'>
				<el-table-column type="selection" width=55></el-table-column>
				<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
			</el-ctable>
		</el-form-item>
		<el-form-item prop='group_id' v-if="ruleForm.device_type=='ENB'">
			<el-input v-model='ruleForm.group_id' v-show="false"></el-input>
		</el-form-item>
		<!-- 基站设备列表  -->
		<div style="margin-left:34px" v-if="ruleForm.device_select_mode == '0' && ruleForm.device_type=='ENB'"  key="enbTable">
			<el-pairgrid ref="cpairgrid" @right-load-success="tableLoadSuccess('enb')" @selection-change='selectChange' query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="rightUrl" :left-url="enbleftUrl" :height="height" :readonly='viewFlag' row-key="serial_number" :query-params="queryForm" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
				<template slot="prev">
					<el-ctable style="width:200px;height:100%;" row-key="id" :url="deviceGroupUrl" rownumber="true" pagination="false" :page-list="pageList" :query-params="queryDeviceForm" @current-change='changeDeviceGroup'>
						<template slot="toolbar">
							<%=rb.getString("SheBeiZu") %>
						</template> 
						<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
					</el-ctable>
				</template>
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
					<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
					<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
				</template>
				<template slot='toolbar'>
					<el-form class="queryGroup" :model='queryForm' ref="queryForm" label-position="top">
						<el-input class='pairgrid-query' style='width:400px;' v-model="queryForm.searchText" @keyup.enter.native="query"
							placeholder="<%=rb.getString("XiaoZhanBianMa")%>" size="mini" ></el-input>
						<el-input style="display:none"></el-input>
				    	<i @click='query' class="el-icon el-icon-common-search"></i>
					</el-form>
				</template>
				<template slot='right'>
					<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
					<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
				</template>
			</el-pairgrid>
		</div>
		<el-form-item prop='device_codes'>
			<el-input v-model='ruleForm.device_codes' v-show="false"></el-input>
		</el-form-item>
		<!--告警列表标识-->
		<div style="margin-left:34px;margin-top:30px">
			<el-pairgrid :readonly='viewFlag' @right-load-success="tableLoadSuccess('calarm')" ref="calarmPairgrid" @selection-change='alarmSelectChange' query-name="ALARM_IDENTIFIER" :rownumber="true" :page-list="pageList" :right-url="alarmRightUrl" :left-url="alarmLeftUrl" :height="height" row-key="ALARM_IDENTIFIER" :query-params="alarmQueryForm" :title="alarmDeviceTitle" :messages="{placeholder:'<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'}">
				<template slot="left">
					<el-table-column type="selection" width="45"></el-table-column>
					<el-table-column prop='DEVICE_TYPE_NAME' label='<%=rb.getString("SheBeiLeiXing")%>' width="100"></el-table-column>
					<el-table-column prop='ALARM_IDENTIFIER' label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' width="130"></el-table-column>
					<el-table-column prop='ALARM_NAME' label='<%=rb.getString("KeNengYuanYin")%>' width=""></el-table-column>
					<el-table-column prop='SERVERITY_TYPE' label='<%=rb.getString("GaoJingJiBie")%>' width="110">
						<template slot-scope="scope">
							<div class="tableTdContainer minor" v-if="scope.row.SERVERITY_TYPE == 'Minor'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("CiYaoGaoJing")%></span>
							</div>
							<div class="tableTdContainer major" v-else-if="scope.row.SERVERITY_TYPE == 'Major'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("ZhuYaoGaoJing")%></span>
							</div>
							<div class="tableTdContainer critical" v-else-if="scope.row.SERVERITY_TYPE == 'Critical'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan" ><%=rb.getString("JinJiGaoJing")%></span>
							</div>
							<div class="tableTdContainer warning" v-else-if="scope.row.SERVERITY_TYPE == 'Warning'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("JingGaoGaoJing")%></span>
							</div>
							
						</template>
					</el-table-column>
				</template>
				<template slot='toolbar'>
					<el-form class="queryGroup" :model='alarmQueryForm' ref="alarmQueryForm" label-position="top">
						<el-input class='pairgrid-query' style='width:400px;' v-model="alarmQueryForm.search_text" @keyup.enter.native="queryAlarm"
							placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>" size="mini" ></el-input>
						<el-input style="display:none"></el-input>
				    	<i @click='queryAlarm' class="el-icon el-icon-common-search"></i>
					</el-form>
				</template>
				<template slot='right'>
					<el-table-column prop='ALARM_IDENTIFIER' label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'></el-table-column>
					<el-table-column prop='SERVERITY_TYPE' label='<%=rb.getString("GaoJingJiBie")%>'>
						<template slot-scope="scope">
							<div class="tableTdContainer minor" v-if="scope.row.SERVERITY_TYPE == 'Minor'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("CiYaoGaoJing")%></span>
							</div>
							<div class="tableTdContainer major" v-else-if="scope.row.SERVERITY_TYPE == 'Major'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("ZhuYaoGaoJing")%></span>
							</div>
							<div class="tableTdContainer critical" v-else-if="scope.row.SERVERITY_TYPE == 'Critical'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan" ><%=rb.getString("JinJiGaoJing")%></span>
							</div>
							<div class="tableTdContainer warning" v-else-if="scope.row.SERVERITY_TYPE == 'Warning'">
								<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("JingGaoGaoJing")%></span>
							</div>
							
						</template>
					</el-table-column>
				</template>
			</el-pairgrid>
			
			<el-form-item prop='alarmIds'>
				<el-input v-model='ruleForm.alarm_id' v-show="false"></el-input>
			</el-form-item>
		</div>
		
		<div class="bottomLine"></div>
		<div class='headContent'>
			<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("TiaoJianSheZhi")%>
		</div>
		<el-form-item label="<%=rb.getString("JieShouRen")%>" prop="email">
			<el-input type="textarea" :disabled="viewFlag" :rows="3" maxlength="500" v-model="ruleForm.email" style="width:680px;">
			
		</el-form-item>
		<span style="display:blcok;margin-left:33px;"><%=rb.getString("YouXiangDiZhiTiShi")%></span>
	</el-form>
</div>
<script>
new Vue({
	el:'#addNotificationPage',
	data(){
		var vm = this;
		var validatorName = (rule,value,callback) => {
			//判断任务名称是否为空
			
			if(value == ""){
				callback(new Error('<%=rb.getString("QingShuRuMuBanMingCheng")%>'))
			}else if(vm.testTaskName == vm.ruleForm.temp_name){
				//用于修改操作的判断
				callback()
			}else{
				//判断任务名称是否已经存在
				axios.post("${ctx}/cell/fault/queryCheckTempNameCount.action",stringify({
					temp_name : vm.ruleForm.temp_name.trim(),
					old_temp_name : vm.oldName
				})).then((response) => {
					if(response.data.flag == "1"){
						callback(new Error('<%=rb.getString("MingChengYiCunZai")%>'))				
					}else{
						callback()
					}
				})
			}
		}
		// 邮箱验证
		validatorEmail = (rule,value,callback) => {
			var reg = /^(([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6}\;))*([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6})$/;
			if(value == ""){
				callback(new Error('<%=rb.getString("QingShuRuShouJianRen")%>'))
			}else if(reg.test(value)){
				callback()
			}else{
				callback(new Error('<%=rb.getString("YouXiangGeShiCuoWu")%>'))
			}
		}
		return {
			initAlarmTypeArr: [],
			testTaskName:"",
			alarmDeviceTitle:['<%=rb.getString("XuanZeGaoJing")%>','<%=rb.getString("YiXuanGaoJing")%>'],  
			alarmQueryForm:{
				search_text:'',
				device_type: ''
			},
			queryDeviceForm:{
				/* searchText:'', */
			}, 
			queryForm:{
				searchText:'',
				groupId:''
			},
			deviceChecked:[],
			pageSize:50,
			deviceGroupUrl:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action',
			deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			enbleftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
			rightUrl:'',
			alarmLeftUrl:'${ctx}/cell/fault/queryAlarmLevelInfosList.action?timeZome='+timeZone,
			alarmRightUrl:'',
			pageList:[50,100,200],
			height:'370px',
			selection:'',
			viewFlag:false,
			alarmSelection:'',
			deviceGroupSelection:'',
			cycleOptions:[],//{id:'5',text:'Real Time'},{id:'10',text:'10'},{id:'30',text:'30'},{id:'60',text:'60'}
			oldName:'',
			ruleForm:{
				temp_name:'',
				temp_id:'',
				desc:'',
				state:"0",
				cycle:'5',
				device_type:'ENB',
				device_select_mode:'1',
				device_code:'',
				alarm_id:'',
				group_id:'',
				email:'',
			},
			rules:{
				temp_name:[
					{validator:validatorName,trigger:'blur'}
				],
				email:[
					{validator:validatorEmail,trigger:'blur'}
				]
			},
			tableStatus:{
				ENBZ:{loaded:false,isFirst:0},
				enb:{loaded:false,isFirst:0},
				calarm:{loaded:false,isFirst:0},
			},
		}
	},
	watch:{
		// 监测 设备组列表选中数据 
		deviceGroupSelection(){
			var data = this.deviceGroupSelection;
			if(data) {
				this.ruleForm.group_id = data.map(function(row){
					return row.id
				})
			}
		},
		// 监测 告警列表选中数据 
		alarmSelection(){
			var data = this.$refs.calarmPairgrid.getData();
			var alarmIds = '';
			if(data.length != 0){
				data.map(function(item){
					alarmIds += item.ALARM_IDENTIFIER + ","
				})
			}
			this.ruleForm.alarm_id = alarmIds
		},
		// 监测 基站设备列表选中数据 
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var deviceCodes = '';
			data.map(function(item){
				deviceCodes += item.small_cell_code + ","
			})
			this.ruleForm.device_code = deviceCodes;
		},
		'ruleForm.device_type': function(val) {
			this.alarmQueryForm.device_type = (val||'').toLowerCase();
		}
	},
	methods:{
		// 表格数据加载成功回调
		tableLoadSuccess(val){
			var vm = this;
			if(vm.ruleForm.device_type !== 'ENB'){
				vm.tableStatus['enb'].loaded = true;
				vm.tableStatus['enb'].isFirst += 1;
			}else{
				vm.tableStatus['enb'].loaded = true;
			}

			vm.tableStatus[val].loaded = true;
			vm.tableStatus[val].isFirst += 1;
			
			var isAllReady = true,
				boolList = [],
				numList = [];
			for(var key in vm.tableStatus){
				var item = vm.tableStatus[key];
				boolList.push(item.loaded);
				numList.push(item.isFirst);
			}
			
			if(boolList.includes(false) || numList.some((item)=>{return item >1})){
				isAllReady = false;
				
			}else{
				isAllReady = true;
			}
			if(isAllReady){
				vm.$nextTick(function(){
					initForm(vm.$refs.ruleForm);
				})
			}
		},
		
		/**
		* 设备组列表选中事件
		* @param selection{Array}   已选中列表
		*/ 
		changeDeviceGroupSelect(selection){
			this.deviceGroupSelection = selection ;
		},
		/**
		* 基站设备列表 左设备组列表选中事件
		* @param currentRow{object}   已选中行数据
		* @param currentRow{object}   上一个已选中行数据
		*/ 
		changeDeviceGroup(currentRow,oldCurrentRow){
			this.queryForm.groupId = currentRow.id
			this.query();
		},
		// 基站设备列表查询
		query(){
			this.$refs.cpairgrid.reload();
		},
		// 设备类型切换
		deviceTypeChange(){
			this.$refs.calarmPairgrid.clear();
		},
		/**
		* 基站设备列表选中事件
		* @param selection{Array}   已选中列表
		*/ 
		selectChange(selection){
			this.selection = selection 
		},
		/**
		* 告警列表选中事件
		* @param selection{Array}   已选中列表
		*/ 
		alarmSelectChange(selection){
			this.alarmSelection = selection 
		},
		// 告警列表查询
		queryAlarm(){
			this.$refs.calarmPairgrid.reload();
		},
		//保存新建任务
		submit(){
			var vm = this;
			vm.$refs.ruleForm.validate((valid) => {
				if(valid){
					let params = {};
					params.temp_name = vm.ruleForm.temp_name;
					params.desc = vm.ruleForm.desc;
					params.state = vm.ruleForm.state;
					params.cycle = vm.ruleForm.cycle;
					params.device_type = vm.ruleForm.device_type;
					params.device_select_mode = vm.ruleForm.device_select_mode;
					if(vm.ruleForm.device_select_mode == 0){
						params.device_code = vm.ruleForm.device_code;
					}
					params.alarm_id = vm.ruleForm.alarm_id;
					params.group_id = vm.ruleForm.group_id;
					params.email =  vm.ruleForm.email;
					eventBus.$emit('change-loading')
					axios.post("${ctx}/cell/fault/saveAlarmEmailInfos.action",stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:"<%=rb.getString("ChengGong")%>",
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-newAlarm')
							eventBus.$emit('change-loading')
	    				}else{
	    					vm.$message.error(data["message"]) 
	    				}
	    			})
				}
			})
		},
		// 修改提交
		submitEdit(){
			var vm = this;
			vm.$refs.ruleForm.validate((valid) => {
				if(valid){
					let params = {};
					params.temp_id =  vm.ruleForm.temp_id;
					params.temp_name = vm.ruleForm.temp_name;
					params.desc = vm.ruleForm.desc;
					params.state = vm.ruleForm.state;
					params.cycle = vm.ruleForm.cycle;
					params.device_type = vm.ruleForm.device_type;
					params.device_select_mode = vm.ruleForm.device_select_mode;
					if(vm.ruleForm.device_select_mode == 0){
						params.device_code = vm.ruleForm.device_code;
					}else{
						params.group_id = vm.ruleForm.group_id || "";
					}
					params.alarm_id = vm.ruleForm.alarm_id;
					//params.group_id = vm.ruleForm.group_id;
					params.email =  vm.ruleForm.email;
					axios.post("${ctx}/cell/fault/updateAlarmEmailInfos.action",stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:"<%=rb.getString("ChengGong")%>",
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-newAlarm')
	    				}else{
	    					vm.$message.error(data["message"]) 
	    				}
	    			})
				}
			})
		},
		//取消保存
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(this.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hide-alarmNotification')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-alarmNotification')
			}
		},
		/**
		* 信息页面
		* @param temp_id{number}   模板id
		*/ 
		infoTask(temp_id){
			var vm = this;
			axios.post("${ctx}/cell/fault/queryEmailAlarmInfo.action",stringify({temp_id:temp_id})).then(function(response){
				var data = response.data;
				vm.viewFlag = true;
				vm.ruleForm.temp_name = data.temp_name;
				vm.ruleForm.desc = data.desc;
				vm.ruleForm.state = data.state;
				vm.ruleForm.cycle = data.cycle;
				vm.deviceChecked = data.group_id;
				vm.ruleForm.device_type = data.device_type.toUpperCase();
				vm.ruleForm.device_select_mode = data.device_select_mode;
				vm.ruleForm.email = data.email;
				vm.ruleForm.group_id = data.group_id.join(",");
				if(data.device_select_mode == 0){
					vm.rightUrl="${ctx}/cell/fault/getEmailDeviceList.action?temp_id="+temp_id+"&deviceType=0";
				}
				vm.alarmRightUrl = "${ctx}/cell/fault/getEmailAlarmServerityList.action?temp_id="+temp_id+"&device_type="+data.device_type;
				initForm(vm.$refs.ruleForm);
			})
		},
		/**
		* 修改页面
		* @param temp_id{number}   模板id
		*/ 
		editTask(temp_id){
			var vm = this;
			axios.post("${ctx}/cell/fault/queryEmailAlarmInfo.action",stringify({temp_id:temp_id})).then(function(response){
				var data = response.data;
				vm.viewFlag = false;
				vm.ruleForm.temp_id = temp_id;
				vm.ruleForm.temp_name = data.temp_name;
				vm.ruleForm.desc = data.desc;
				vm.ruleForm.state = data.state;
				vm.ruleForm.cycle = data.cycle;
				vm.deviceChecked = data.group_id;
				vm.ruleForm.group_id = data.group_id;
				vm.ruleForm.device_type = data.device_type.toUpperCase();
				vm.ruleForm.device_select_mode = data.device_select_mode;
				vm.ruleForm.group_id = data.group_id.join(",");
				vm.ruleForm.email = data.email;
			    vm.oldName = data.temp_name;
			    if(data.device_select_mode == 0){
					vm.rightUrl="${ctx}/cell/fault/getEmailDeviceList.action?temp_id="+temp_id+"&deviceType=0";
				}
				vm.alarmRightUrl = "${ctx}/cell/fault/getEmailAlarmServerityList.action?temp_id="+temp_id+"&device_type="+data.device_type;
				initForm(vm.$refs.ruleForm);
			})
		},
		// 通知周期数据
		getCycle(){
			var vm = this
			axios.post("${ctx}/cell/fault/getEmailCycleList.action").then(function(response){
				if(response.data instanceof Array){
					vm.cycleOptions = response.data;
				}
			})
		},
		//初始化 告警类型值
		initAlarmType(){
			var vm = this;
			axios.post("${ctx}/cell/fault/queryAlarmType.action").then((res) => {
				vm.initAlarmTypeArr = res.data.AlarmType;
			})
		}
	},
	created(){
		this.getCycle();
	},
	mounted(){
		eventBus.$off("hander-cancel").$on("hander-cancel",this.cancel);
		eventBus.$off('handle-ok-new').$on('handle-ok-new',this.submit);
		eventBus.$off('handle-ok-edit').$on('handle-ok-edit',this.submitEdit);
		eventBus.$off('info-task').$on('info-task',this.infoTask);
		eventBus.$off('edit-task').$on('edit-task',this.editTask);
		this.initAlarmType();
	}
})
</script>