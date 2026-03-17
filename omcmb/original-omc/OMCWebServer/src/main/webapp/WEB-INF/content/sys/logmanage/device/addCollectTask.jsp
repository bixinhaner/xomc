<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<style>
#addCollectLog{
	box-shadow:none;
}
#addCollectLog .queryGroup {
	margin: 10px 20px;
}
#addCollectLog .pairgridBox{
	margin-left:35px;
	margin-right:35px;
	margin-top:20px;
}
#addCollectLog .pt20{
	padding-top: 20px
}
#addCollectLog .ml10{
	margin-left:10px
}
#addCollectLog .excuteMode-radio .el-radio__input{
	margin-top:-2px;
}
#addCollectLog .deviceTip {
	position:relative;
}
#addCollectLog .deviceTip .deviceNum-tip {
	font-size:12px;
	color:#4D84FF;
	position:absolute;
	top:44px;
	left:100px;
}
#addCollectLog .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addCollectLog .titleStyML{
	margin-left: 20px;
}

#addCollectLog .editButton{
	position: absolute;
	right: 130px;
	padding:0 10px;
	height:24px;
	background:#F2F9FF;
	border-radius:2px;
	line-height:24px;
	cursor:pointer;
	margin-left:10px;
	border:1px solid #1DA3FC;
	display:inline-block;
	top: -2px;
}
#addCollectLog .editButton i{
	font-size:14px !important;
}
#addCollectLog .editButton span{
	font-size:12px;
}
</style>

<!--新建日志收集任务页面 -->
<div id='addCollectLog' style="overflow: auto;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" class="pt20">
	 	<div class="group-title not-extend deviceTip titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			<span class="deviceNum-tip" v-show="showDeviceNum"><%=rb.getString("ZuiDuoXuanZeSheBei100")%></span>
		</div>
		<el-pairgrid :limit="limitNum" class="pairgridBox" :id="'select_device_list'" :rownumber="true" ref="cpairgrid" @selection-change='selectChange' 
					:left-url="leftUrl" :height="height" :row-key="'serial_number'" :query-params="queryForm" style="margin-left:45px"
					:title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
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
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width="300"></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
			<!-- 模糊查询 -->
			<template slot="toolbar">
				<div class="queryGroup">
					<el-input class='pairgrid-query' v-model='queryForm.search_text' @keyup.enter.native="query"
						:placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'" size="small"></el-input>
			    	<i @click='query' class="el-icon el-icon-common-search ml10" ></i>
		    	</div>
		    	<div class="editButton" size="mini" @click="addBatchSn" v-show="showDeviceNum && curSelectDataLength < 100">
					<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
					<span><%=rb.getString("PiLiangShuRu")%></span>
				</div>
			</template>
			<!--右侧的下拉表格 -->
			<template slot='right'>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
		</el-pairgrid>
		<el-form-item prop='cellCodes' style="margin-left:45px;">
			<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		<el-form-item prop='serial_number' style="margin-bottom:0px;">
			<el-input v-model='ruleForm.serial_number' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine" v-show="showExecuteMode"></div>

		<!-- 类型 -->
		<div>
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("Type")%></span>
			</div>
			<div style="padding: 10px 30px;">
				<div class="el-textarea__inner" style='border:none;'>
					<el-form-item style='display:inline-block;margin:10px 0px' prop='logType'>
						<el-radio-group v-model="ruleForm.logType" @change="logTypeChange">
							<el-radio label="deviceLog" style='margin-right:80px;' class="excuteMode-radio"><%=rb.getString("DeviceSheBeiRiZhi")%></el-radio>
							<el-radio label="securityLog" style='margin-right:80px;' class="excuteMode-radio">Security Logs</el-radio>
						</el-radio-group>
					</el-form-item>
					
				</div>
			</div>
		</div>
		<div class="alarmBottomLine" v-show="showExecuteMode"></div>

		<!-- 执行方式 -->
		<div v-show="showExecuteMode">
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("ZiKaiZhanZhiXingFangShi")%></span>
			</div>
			<div style="padding: 10px 30px;">
				<div v-if="ruleForm.logType == 'deviceLog'" class="el-textarea__inner" style='border:none;'>
					<el-form-item style='display:inline-block;margin:10px 0px' prop='execute_type'>
						<el-radio-group v-model="ruleForm.execute_type">
							<el-radio label="Immediately" style='margin-right:80px;' class="excuteMode-radio"><%=rb.getString("LiJiZhiXing")%></el-radio>
						</el-radio-group>
					</el-form-item>
					
				</div>
				<div class="el-textarea__inner" style='border:none;'>
					<el-form-item style='display:inline-block;margin:10px 0px' prop='execute_type'>
						<el-radio-group v-model="ruleForm.execute_type">
							<el-radio label="Periodically" style='margin-right:10px;' class="excuteMode-radio"><%=rb.getString("DingShiZhiXing")%></el-radio>
						</el-radio-group>
					</el-form-item>
					<el-form-item prop="time" style='display:inline-block;margin:10px 0px' class='timeItem'>
						<el-date-picker size="mini"
							type="datetimerange"
							v-model="ruleForm.time" 
							:disabled="setTimeEnable"
							@focus='setTime'
							:picker-options="pickerOptions"
							range-separator="——"
							value-format="yyyy-MM-dd HH:mm:ss"
							start-placeholder="<%=rb.getString("KaiShiShiJian")%>" 
							end-placeholder="<%=rb.getString("JieShuShiJian")%>">
						</el-date-picker>
					</el-form-item> 

					<el-form-item style='display:inline-block;margin:-15px 0 -15px 30px;' :label="ruleForm.logType == 'deviceLog'?'<%=rb.getString("FenZhongZhouQi") %>':'<%=rb.getString("TianZhouQi") %>'">
						
					</el-form-item>
					<el-form-item style="display:inline-block;margin:10px 0px;" prop="reportPeriod">
						<el-select v-model="ruleForm.reportPeriod" size="mini" :disabled="setTimeEnable">
							<el-option v-for="item in periodList" :label='item.label' :value='item.value'></el-option>
						</el-select>
					</el-form-item>

				</div>
			</div> 
		</div>	
	</el-form>
	
	<el-dialog class='dialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">

var colectLogsVue = new Vue({
	el:'#addCollectLog',
	data(){
		
		//校验定时执行-时间
		var vm = this;
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.execute_type !== 'Periodically'){
				callback()
			}else{
				if(value == '' || value==null){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'));
				}else{
					//周期上报选择时间范围：不可超过24H
					if(Array.isArray(value)) {				
					 	var startTime = new Date(value[0]);
					 	var endTime = new Date(value[1]);
					 	var start = startTime.getTime() + 24*60*60*1000; 
					 	var end_time = endTime.getTime();
					 	var start_time = startTime.getTime();						
						var curTime = new Date(gloableTime).getTime(); //当前时间戳 当前运营商的时间 ok		
						
						if(vm.ruleForm.logType == 'securityLog') {
							start = startTime.getTime() + 6*24*60*60*1000;
						}
						
						if(end_time > start){
							if(vm.ruleForm.logType == 'securityLog') {
								callback(new Error('<%=rb.getString("SASRiZhiShiJianFanWei")%>'));
							}else {
								callback(new Error('<%=rb.getString("ShiJianFanWei")%>'));
							}
						}else if(start_time == end_time){
							callback(new Error('<%=rb.getString("JieShuShiJianXuWanYuKaiShiShiJian")%>'));
						}else if(start_time <= curTime || end_time <= curTime){// 不可选择过去的时间
							callback(new Error('<%=rb.getString("BuKeXuanZeGuoQuDeShiJian")%>'));
						}else{
							callback();
						}
												
					}
				}
			}		
		},
		validatorNum = (rule,value,callback) => {
			var serialNumber = value, 
				temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
					return item.length > 0;
				});
			
			if (serialNumber == null || serialNumber.length == 0) {
				callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
			}else {
				var nameFlag = list.every(function(item,index){
					return temp.test(item)
				})
				if(nameFlag){
					callback()
				}else{
					callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
				}
			}
		};
		return{
			queryForm:{
				timeZone:timeZone,
				search_text:'',
				like_fields : 'serial_number,host_name'
			},
			params:'',
		    rowData:[],
		    leftUrl:'',
		    selection:'',
		    setTimeEnable:true,
		
		    //新建收集任务-表单默认项
		    ruleForm:{
				cellCodes:'', //设备方式
				logType: 'deviceLog',
				execute_type:'Immediately',//立即执行为选中状态

				time:'',
				//定时执行
				reportPeriod:'900'//周期默认为 15
			},
			rules:{
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
				],
				//新建收集任务-定时执行
				time:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
		    groupOptions:[],
		    deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
		    limitNum:'',
		    pageSize:'100',
		    height:'70%',
		    width:'100%',
		    showTip:true,
		    modal:false,
		    queryButton:'<%=rb.getString("ChaXun")%>',
		    resetButton:'<%=rb.getString("ChaXunChongZhi")%>',
		    showExecuteMode : false,
		    showDeviceNum: false,
		    pickerOptions:{
				disabledDate(time){
					//周期上报选择时间范围：当前时间之前的日期为置灰状态
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			
			listVisible:false,
			dialogTitle:"<%=rb.getString("TianJia")%>",
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator: validatorNum,trigger:'change'}
				]
			},
			curSelectDataLength: ''
		}		
	},
	computed: {
		periodList() {
			var vm = this,
				type = vm.ruleForm.logType;

			if(type == 'deviceLog') {
				return [{label: '15', value: '900'},{label: '30', value: '1800'},{label: '60', value: '3600'}];
			}else {
				return [
					{label: '0.5', value: '12'},
					{label: '1', value: '24'},
					{label: '1.5', value: '36'},
					{label: '2', value: '48'},
					{label: '2.5', value: '60'},
					{label: '3', value: '72'},
					{label: '3.5', value: '84'},
					{label: '4', value: '96'},
					{label: '4.5', value: '108'},
					{label: '5', value: '120'},
					{label: '5.5', value: '132'},
					{label: '6', value: '144'},
					{label: '6.5', value: '156'},
					{label: '7', value: '168'}
				];
			}
		}
	},
	methods:{ 
		init:function(){
			var vm = this;

			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			
			vm.leftUrl = '${ctx}/system/device/enodeb/queryENBInfoPageListForSelect.action'
			vm.$refs.cpairgrid.reload();

			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			
		},
		// 手动导入 sn
		addBatchSn(){
			this.listVisible = true;
		},
		closeBatchSn(){
			var vm = this;
			vm.listVisible = false;
			vm.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '',
			list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			var params = {
					serialNumbers : list.join(";"),
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/cpeinfos/getTaskCheckSNList.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){		
							vm.$refs.cpairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message("<%=rb.getString("MeiYouKePiPeiSheBei")%>")
						}
					})
				}
			})
		},
		/**
		 * 查询
		 * @param val:输入的参数
		*/
		query(val){ 
			this.$refs.cpairgrid.reload();
		},
		/**
		 * 表格全选
		 * @param selection:选择的数据
		*/
		selectChange(selection){ 
			this.selection = selection
		},
		
		/**
		 * 新建设备上报日志任务 
		 * executeMode == "first"  显示执行方式
		**/
		initExecuteMode(executeMode){
			if(executeMode == "first"){
				this.showExecuteMode = true;
				this.showDeviceNum = true;
				this.limitNum = "100";
			}else{
				this.queryForm.featureGroup = 'Maintenance';
				this.queryForm.featureCode = 'CODE_ENB_ALARM_LOG';
				this.showExecuteMode = false;
				this.showDeviceNum = false;
				this.limitNum = "";
			}		
		},
		
		/**
		 * 点击确定函数
		 * @param pane:保存时候判断的参数 是first或者 second
		*/
		submit(pane){ 
	    	var vm = this;
	    	var message = '';
            // 防止多次提交
            if(enbLogsVue.slideSubmitLoading)return

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			if(pane == 'first'){
	    				params.serial_number = vm.ruleForm.serial_number;//设备编码 				    				
	    				params.timeZone = timeZone;
	    				params.isReboot = "false";//是否重新启动
	        			params.device_code = vm.ruleForm.cellCodes;
	        			params.device_type = 'eNB';
	        			var dateTime = vm.ruleForm.time;  	        			
	        			//执行方式： Periodically 定时执行--开始时间，结束时间
	        			params.execute_type = vm.ruleForm.execute_type;
						params.logType = vm.ruleForm.logType;
	        			
	        			//执行方：立即执行，周期粒度 传空
	        			if (params["execute_type"] == "Immediately") {
	        				params.reportPeriod = '';
	        				params.start_time = dateTime[0];
		    				params.end_time = dateTime[1];
	        			}else{
	        				params.reportPeriod = vm.ruleForm.reportPeriod;
	        				//判断时间是否为空
		    				if(dateTime.length == 0 || dateTime == null){
		    					return false;
		    				}else{
		    					if(dateTime[0] == 2){
		    						params.start_time = '';
		    						params.end_time = '';
		    						vm.ruleForm.time = []; 
		    						return false;
		    					}else{	
		    						params.start_time = dateTime[0];
		    	    				params.end_time = dateTime[1];
		    					}
		    					
		    				} 
	        			}
	        			
	        	    	url = '${ctx}/cell/collect/goImmediateCollectLogFile.action'
	    			}else if(pane == 'second'){
	    				params.device_type = 'eNB';
	    				params.enodebCodes = vm.ruleForm.cellCodes;
	    				
	    				url = "${ctx}/cell/collect/alarm/addTask.action";
	    			}
	    			
	    	    	message = '<%=rb.getString("ChengGong")%>'
                    enbLogsVue.slideSubmitLoading = true;
	    			axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            eventBus.$emit('close-collect');
	    				}else if(data.responseCode == "901"){
		    				vm.$message.error(data.message);
		    				enbLogsVue.slideSubmitLoading = false;
		    			}else if(data.responseCode == "401"){//存在未完成的任务时，再次创建任务失败，给出提示
		    				vm.$message.error(data["message"])
                            enbLogsVue.slideSubmitLoading = false;
		    			}else{
		    	 			vm.$message.error('<%=rb.getString("ShouJiShiBai")%>');
                            enbLogsVue.slideSubmitLoading = false;
		    	 		}
	    			})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		cancel(){ // 取消函数
			eventBus.$emit('hide-collect')
		},

		setTime(){
			if(this.ruleForm.time && this.ruleForm.time.length) {
				
			}else {
				this.ruleForm.time = [formatDate(new Date(gloableTime)),formatDate(new Date(gloableTime))];
			}
			this.$refs.ruleForm.validateField('time');
		},
		logTypeChange(type) {
			var vm = this;

			if(type == 'deviceLog') {
				vm.ruleForm.reportPeriod = '900';
			}else if(type == 'securityLog') {
				vm.ruleForm.reportPeriod = '12';
				vm.ruleForm.execute_type = 'Periodically';
			}
		}
	},
	watch:{
		selection(){ // 监听选择
			var data = this.$refs.cpairgrid.getData();
			this.curSelectDataLength = data.length;
			var cellCodes = '';
			var serial_number = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.small_cell_code + ","
					serial_number += item.serial_number + ","
				})
			}
			this.ruleForm.cellCodes = cellCodes;
			this.ruleForm.serial_number = serial_number
		},
		rowData(newVal){
			this.ruleForm.file = newVal.id
		},
		"ruleForm.execute_type":function(newVal){
			if(newVal == 'Periodically'){
				this.setTimeEnable = false;
			}else{
				this.setTimeEnable = true;
				//清空日期时间选择器
				this.ruleForm.time = [];
				this.ruleForm.reportPeriod = "900";

				if(this.ruleForm.logType == 'securityLog') {
					this.ruleForm.reportPeriod = "12";
				}
			}
			this.$refs.ruleForm.validateField('time')
		}
	},
	mounted(){
		this.init();
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('close-collect').$on('close-collect',this.cancel);
		eventBus.$off("collect-type").$on("collect-type",this.initExecuteMode);

	}
})
</script>