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
		/* margin-top:30px; */
		margin-bottom:20px;
	}
	.headContent img{
		vertical-align:top;
		margin-right:14px;
	}
	.boxBorderCon{
		width:700px;
		height:27px;
		padding-top:13px;
		border:1px solid #DEDFE6;
	}
	.bottomline{
		width:80%;
		height:1px;
		background:#E9E9E9;
		margin-top:35px;
		margin-bottom:20px;
	}
	.el-input__inner{
	    -webkit-appearance: none;
	    background-color: #fff;
	    background-image: none;
	    border-radius: 4px;
	    border: 1px solid #dcdfe6;
	    box-sizing: border-box;
	    color: #606266;
	    display: inline-block;
	    font-size: inherit;
	    outline: none;
	    padding: 0 15px;
	    transition: border-color .2s cubic-bezier(.645,.045,.355,1);
	    width: 100%;
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
	#addStatisticPage{
		box-sizing:border-box;
		padding:20px;
	}
	#addStatisticPage .el-pairgrid{
		width:80%;
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
</style>
<div id="addStatisticPage">
	<el-form :model="ruleForm" :rules="rules" ref="ruleForm" label-position="top">
		<div class="headContent">
			<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("JiBenXinXi") %>
		</div>
		<el-form-item label="<%=rb.getString("GuiZeMingCheng")%>" prop="statisticName">
			<el-input ref="taskNameInput" maxLength=50 :disabled="viewFlag" v-model="ruleForm.statisticName" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		
		<el-form-item prop='statisticIndicator' label='<%=rb.getString("FaShengCiShu")%>' v-if="false">
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.statisticIndicator' :disabled='viewFlag'>
					<el-radio :label="0" style="margin-left:21px"><%=rb.getString("FaShengCiShu")%></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<el-form-item prop='statisticType' label='<%=rb.getString("TongJiLeiXing")%>'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.statisticType' :disabled='viewFlag'>
					<el-radio :label="0" style="margin-left:21px"><%=rb.getString("SheBeiTongJi")%></el-radio>
					<el-radio :label="1" style="margin-left:25px"><%=rb.getString("SheBeiZu")%></el-radio>
					<el-radio :label="2" style="margin-left:25px"><%=rb.getString("AlarmId")%></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<el-form-item prop='statisticTime' label='<%=rb.getString("TongJiLiDu")%>'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.statisticTime' :disabled='viewFlag'>
					<el-radio :label="0" style="margin-left:21px"><%=rb.getString("XiaoShi")%></el-radio>
					<el-radio :label="1" style="margin-left:25px"><%=rb.getString("Tian")%></el-radio>
					<el-radio :label="2" style="margin-left:25px"><%=rb.getString("Zhou")%></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<div class="bottomLine"></div>
		<div class='headContent'>
			<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("TiaoJianSheZhi")%>
		</div>
		<el-form-item label='<%=rb.getString("SheBeiLeiXing") %>'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.deviceType' :disabled='viewFlag'>
					<el-radio :label="0" style="margin-left:21px">eNB</el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<el-form-item label='<%=rb.getString("SheBeiXuanZe") %>'>
			<div class="boxBorderCon">
				<el-radio-group v-model='ruleForm.deviceSelectMode' :disabled='viewFlag'>
					<el-radio :label="1" style="margin-left:21px"><%=rb.getString("SheBeiZu") %></el-radio>
					<el-radio :label="0" style="margin-left:21px"><%=rb.getString("SheBeiLieBiao") %></el-radio>
				</el-radio-group>
			</div>
		</el-form-item>
		<!-- 设备组列表 -->
		<el-form-item v-if="ruleForm.deviceSelectMode == '1'" style="height:300px;">
			<el-ctable border=true :readonly='viewFlag' ref="deviceGroupTable" :default-checked="deviceChecked" row-key="id" :rownumber="true" pagination="true" :page-list="pageList":query-params="queryDeviceForm" :url="deviceGroupUrl" style="width:60%;height:100%;border:1px solid #DEDFE6;" @selection-change='changeDeviceGroupSelect'>
				<el-table-column type="selection" width=55></el-table-column>
				<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
			</el-ctable>
		</el-form-item>
		<el-form-item prop='groupId'>
			<el-input v-model='ruleForm.groupId' v-show="false"></el-input>
		</el-form-item>
		<!-- 基站设备列表  -->
		<div style="margin-left:34px" v-if="ruleForm.deviceSelectMode == '0'"  key="enbTable">
			<el-pairgrid ref="cpairgrid" @selection-change='selectChange' query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="rightUrl" :left-url="enbleftUrl" :height="height" :readonly='viewFlag' row-key="serial_number" :query-params="queryForm" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
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
			<el-pairgrid :readonly='viewFlag' ref="calarmPairgrid" @selection-change='alarmSelectChange' query-name="ALARM_IDENTIFIER" :rownumber="true" :page-list="pageList" :right-url="alarmRightUrl" :left-url="alarmLeftUrl" :height="height" row-key="ALARM_IDENTIFIER" :query-params="alarmQueryForm" :title="alarmDeviceTitle" :messages="{placeholder:'<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'}">
				<template slot="left">
					<el-table-column type="selection" width="45"></el-table-column>
					<el-table-column prop='DEVICE_TYPE_NAME' label='<%=rb.getString("SheBeiLeiXing")%>' width="100"></el-table-column>
					<el-table-column prop='ALARM_IDENTIFIER' label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' width="130"></el-table-column>
					<el-table-column prop='ALARM_NAME' label='<%=rb.getString("KeNengYuanYin")%>' width="" show-overflow-tooltip></el-table-column>
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
			<!-- 告警类型  -->
		<el-form-item prop='status' label='<%=rb.getString("GaoJingLeiXing")%>' style="margin-top:30px;margin-left:unset;">
			<div class="boxBorderCon boxContainerAlarmType" style="padding-left:23px;min-height:27px;height:unset;">
				<el-checkbox-group v-model="ruleForm.alarmType" :disabled='viewFlag'>
					<el-checkbox label="30003"><%=rb.getString("SheBeiGaoJing")%></el-checkbox>
					<el-checkbox label="30001"><%=rb.getString("FuWuZhiLiangGaoJing")%></el-checkbox>
					<el-checkbox label="30000"><%=rb.getString("TongXinGaoJing")%></el-checkbox>
					<el-checkbox label="30004" style="margin-left:0px;"><%=rb.getString("HuanJingGaoJing")%></el-checkbox>
					<el-checkbox label="30002"><%=rb.getString("ChuLiShiBaiGaoJing")%></el-checkbox>
					<el-checkbox label="30006"><%=rb.getString("XingNengYiChuGaoJing")%></el-checkbox>
					<!-- <el-checkbox label="30007">Event Alarm</el-checkbox> -->
				</el-checkbox-group>
			</div>
		</el-form-item>
		<!-- 告警级别  -->
		<el-form-item prop='status' label='<%=rb.getString("GaoJingJiBie")%>' style="margin-top:30px;margin-left:unset;">
			<div class="boxBorderCon" style="padding-left:23px;">
				<el-checkbox-group v-model="ruleForm.alarmServerity" :disabled='viewFlag'>
					<el-checkbox label="31001"><%=rb.getString("JinJiGaoJing")%></el-checkbox>
					<el-checkbox label="31002"><%=rb.getString("ZhuYaoGaoJing")%></el-checkbox>
					<el-checkbox label="31003"><%=rb.getString("CiYaoGaoJing")%></el-checkbox>
					<el-checkbox label="31004"><%=rb.getString("JingGaoGaoJing")%></el-checkbox>
				</el-checkbox-group>
			</div>
		</el-form-item>
		<el-form-item prop='dataRange' label='<%=rb.getString("TongJiShiJian")%>' style="margin-left:unset;">
			<div class="boxBorderCon" style="padding-left:23px;padding-bottom:13px;"><!-- :picker-options="pickerOptions" -->
				<el-date-picker :picker-options=pickerOptions :disabled='viewFlag' v-model="ruleForm.dataRange" format="yyyy-MM-dd" :editable="false" value-format="yyyy-MM-dd HH:00:00" type="daterange"  rang-separator="to" start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>' >
								
				</el-date-picker>
			</div>
			<p v-show="timeTitleFlag" style="color:red;"><%=rb.getString("TongJiShiJianBuNengWeiKong")%> </p>
		</el-form-item>
		
		</div>
	</el-form>
</div>
<script>
new Vue({
	el:'#addStatisticPage',
	data(){
		var vm = this;
		var validatorName = (rule,value,callback) => {
			//判断任务名称是否为空
			if(value == ""){
				callback(new Error('<%=rb.getString("QingShuRuGuiZeMingCheng")%>'))
			}else if(vm.testTaskName == vm.ruleForm.statisticName){
				//用于修改操作的判断
				callback()
			}else{
				//判断任务名称是否已经存在
				axios.post("${ctx}/cell/fault/statisticNameExist.action",stringify({
					statisticName : vm.ruleForm.statisticName.trim(),
					oldStatisticName : vm.oldName
				})).then((response) => {
					if(response.data.exist){
						callback(new Error('<%=rb.getString("MingChengYiCunZai")%>'))				
					}else{
						callback()
					}
				})
			}
		},
		validatorEmail = (rule,value,callback) => {
			var reg = /^(([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6}\;))*([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6})$/;
			if(value == ""){
				callback(new Error('<%=rb.getString("QingShuRuShouJianRen")%>'))
			}else if(reg.test(value)){
				callback()
			}else{
				callback(new Error('<%=rb.getString("YouXiangGeShiCuoWu")%>'))
			}
		},
		validRange = function(rule,value,callback) { // 校验时间范围
			var dateRange = vm.ruleForm.dataRange;
			if(dateRange && dateRange.length) {
				var endstr = vm.ruleForm.dataRange[1].substr(0,10) + ' 00:00:00';
				var num = differ(endstr,vm.ruleForm.dataRange[0]);
				if(num>6){
					callback('Date spans more than seven days');
				}else{
					callback();
				}
			}else {
				callback('<%=rb.getString("QingXuanZeShiJian")%>');
			}
		};
		
		return {
		
			pickerOptions:{
				disabledDate(time){
					let _now = Date.now();
					return time.getTime() > _now;
				}
			},
			timeTitleFlag:false,
			testTaskName:"",
			alarmDeviceTitle:['<%=rb.getString("XuanZeGaoJing")%>','<%=rb.getString("YiXuanGaoJing")%>'],  
			alarmQueryForm:{
				search_text:''
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
			alarmLeftUrl:'${ctx}/cell/fault/queryAlarmLevelInfosList.action?device_type=enb&timeZome='+timeZone,
			alarmRightUrl:'',
			pageList:[50,100,200],
			height:'370px',
			selection:'',
			viewFlag:false,
			alarmSelection:'',
			deviceGroupSelection:'',
			oldName:'',
			ruleForm:{
				statisticName:'',
				statisticId:'',
				statisticIndicator:0,
				statisticTime:0,
				statisticType:0,
				deviceType:0,
				deviceSelectMode:1,
				deviceCode:'',
				alarm_id:'',
				groupId:'',
				alarmType:[],
				alarmServerity:[],
				dataRange:[]
			},
			rules:{
				statisticName:[
					{validator:validatorName,trigger:'blur'}
				],
				dataRange: [
					{validator: validRange}
				]
			}
		}
	},
	watch:{
		// 设备组列表已选中列表监测
		deviceGroupSelection(){
			var data = this.deviceGroupSelection;
			let device_group = ""
			data.map(function(item){
				device_group += item.id+','
			})
			this.ruleForm.groupId = device_group;
		},
		// 告警列表已选中列表监测
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
		// 基站设备列表已选中列表监测
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var deviceCodes = '';
			data.map(function(item){
				deviceCodes += item.small_cell_code + ","
			})
			this.ruleForm.deviceCode = deviceCodes;
		}
	},
	methods:{
		/**
		* 设备组列表选中数据
		* @param currentRow{Array}  
		*/
		changeDeviceGroupSelect(selection){
			this.deviceGroupSelection = selection ;
		},
		/**
		* 基站设备列表 左设备组列表选中事件
		* @param currentRow{object}   已选中行数据
		* @param oldCurrentRow{object}   上一个已选中行数据
		*/ 
		changeDeviceGroup(currentRow,oldCurrentRow){
			this.queryForm.groupId = currentRow.id
			this.query();
		},
		// 基站设备列表查询
		query(){
			this.$refs.cpairgrid.reload();
		},
		/**
		* 基站设备列表选中数据
		* @param selection{Array}  
		*/
		selectChange(selection){
			this.selection = selection 
		},
		/**
		* 告警列表选中数据
		* @param selection{Array}  
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
					params.statisticName = vm.ruleForm.statisticName;
					params.statisticIndicator = vm.ruleForm.statisticIndicator;
					params.statisticTime = vm.ruleForm.statisticTime;
					params.statisticType = vm.ruleForm.statisticType;
					params.deviceType = vm.ruleForm.deviceType;
					params.alarmType = vm.ruleForm.alarmType;
					params.alarmServerity = vm.ruleForm.alarmServerity;
					params.deviceSelectMode = vm.ruleForm.deviceSelectMode;
					params.timeZone = timeZone;
					if(vm.ruleForm.deviceSelectMode == 0){
						params.deviceCode = vm.ruleForm.deviceCode;
					}else{
						params.groupId = vm.ruleForm.groupId;
					}
					params.alarmId = vm.ruleForm.alarm_id;
					//判断时间是否为空
					if(vm.ruleForm.dataRange.length == 0 || vm.ruleForm.dataRange == null){//没有选择时间不能提交
						vm.timeTitleFlag = true;
						return false;
					}else{
						vm.timeTitleFlag = false;
						params.startTime = vm.ruleForm.dataRange[0];
						params.endTime = vm.ruleForm.dataRange[1].substr(0,10)+' 23:59:59';
					}
					
					axios.post("${ctx}/cell/fault/saveAlarmStatistic.action",stringify(params)).then(function(response){
	    				var data = response.data;
						eventBus.$emit('change-loading')
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
		// 修改保存
		submitEdit(){
			var vm = this;
			vm.$refs.ruleForm.validate((valid) => {
				if(valid){
					let params = {};
					params.statisticId = vm.ruleForm.statisticId
					params.statisticName = vm.ruleForm.statisticName;
					params.statisticIndicator = vm.ruleForm.statisticIndicator;
					params.statisticTime = vm.ruleForm.statisticTime;
					params.statisticType = vm.ruleForm.statisticType;
					params.deviceType = vm.ruleForm.deviceType;
					params.alarmType = vm.ruleForm.alarmType;
					params.alarmServerity = vm.ruleForm.alarmServerity;
					params.deviceSelectMode = vm.ruleForm.deviceSelectMode;
					params.timeZone = timeZone;
					if(vm.ruleForm.deviceSelectMode == 0){
						params.deviceCode = vm.ruleForm.deviceCode;
					}else{
						params.groupId = vm.ruleForm.groupId || "";
					}
					params.alarmId = vm.ruleForm.alarm_id;
					//判断时间是否为空
					if(vm.ruleForm.dataRange.length == 0 || vm.ruleForm.dataRange.length == null){//没有选择时间不能提交
						vm.timeTitleFlag = true;
						return false;
					}else{
						vm.timeTitleFlag = false;
						params.startTime = vm.ruleForm.dataRange[0];
						params.endTime = vm.ruleForm.dataRange[1].substr(0,10)+' 23:59:59';
					}
					axios.post("${ctx}/cell/fault/modifyAlarmStatistic.action",stringify(params)).then(function(response){
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
					eventBus.$emit('hide-statistic')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-statistic')
			}
		},
		/**
		* 信息页面数据请求
		* @param statisticId{number}   模板id
		*/ 
		infoTask(statisticId){
			var vm = this;
			axios.post("${ctx}/cell/fault/queryStatisticInfoById.action",stringify({statisticId:statisticId,timeZone:timeZone})).then(function(response){
				var data = response.data;
				vm.viewFlag = true;
				vm.ruleForm.statisticName = data.statisticName;
				vm.ruleForm.statisticIndicator = data.statisticIndicator;
				vm.ruleForm.statisticTime = data.statisticTime;
				vm.ruleForm.statisticType = data.statisticType;
				vm.deviceChecked = data.groupId;
				vm.ruleForm.groupId = data.groupId.join(",");
				vm.ruleForm.deviceType = data.deviceType;
				vm.ruleForm.deviceSelectMode = data.deviceSelectMode;
			    vm.oldName = data.statisticName;
			    vm.ruleForm.alarmType = data.alarmType;
			    
				vm.ruleForm.alarmServerity = data.alarmServerity;
			    if(data.deviceSelectMode == 0){
					vm.rightUrl="${ctx}/cell/fault/queryStatisticDeviceById.action?statisticId="+statisticId+"&deviceType=0"
				}
				vm.alarmRightUrl = "${ctx}/cell/fault/queryStatisticIdentifierById.action?statisticId="+statisticId+"&deviceType="+data.deviceType
				var timeRange = [];
				timeRange.push(data.startTime);
				timeRange.push(data.endTime);
				vm.ruleForm.dataRange = timeRange;
				setTimeout(function(){
					initForm(vm.$refs.ruleForm);
				},500); 
			})
		},
		/**
		* 修改页面请求数据
		* @param statisticId{number}   模板id
		*/ 
		editTask(statisticId){
			var vm = this;
			vm.ruleForm.statisticId = statisticId
			axios.post("${ctx}/cell/fault/queryStatisticInfoById.action",stringify({statisticId:statisticId,timeZone:timeZone})).then(function(response){
				var data = response.data;
				vm.viewFlag = false;
				vm.ruleForm.statisticName = data.statisticName;
				vm.ruleForm.statisticIndicator = data.statisticIndicator;
				vm.ruleForm.statisticTime = data.statisticTime;
				vm.ruleForm.statisticType = data.statisticType;
				vm.deviceChecked = data.groupId;
				vm.ruleForm.groupId = data.groupId.join(",");
				vm.ruleForm.deviceType = data.deviceType;
				vm.ruleForm.deviceSelectMode = data.deviceSelectMode;
			    vm.oldName = data.statisticName;
			    vm.ruleForm.alarmType = data.alarmType;
				vm.ruleForm.alarmServerity = data.alarmServerity;
			    if(data.deviceSelectMode == 0){
					vm.rightUrl="${ctx}/cell/fault/queryStatisticDeviceById.action?statisticId="+statisticId+"&deviceType=0"
				}
				vm.alarmRightUrl = "${ctx}/cell/fault/queryStatisticIdentifierById.action?statisticId="+statisticId+"&deviceType="+data.deviceType
				var timeRange = [];
				timeRange.push(data.startTime);
				timeRange.push(data.endTime);
				vm.ruleForm.dataRange = timeRange;
				setTimeout(function(){
					initForm(vm.$refs.ruleForm);
				},500); 
			})
		},
	},
	created(){
		
	},
	mounted(){
		eventBus.$off('handle-ok-new').$on('handle-ok-new',this.submit);
		eventBus.$off('info-task').$on('info-task',this.infoTask);
		eventBus.$off('edit-task').$on('edit-task',this.editTask);
		eventBus.$off("hander-cancel").$on("hander-cancel",this.cancel);
		eventBus.$off('handle-ok-edit').$on('handle-ok-edit',this.submitEdit);	
	}
})
</script>