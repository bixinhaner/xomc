<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<style>
	
	#addGnbCollectLog{
		box-shadow:none;
	}
	
	#addGnbCollectLog .pairgridBox{
		margin-left:35px;
		margin-right:35px;
		margin-top:20px;
	}
	#addGnbCollectLog .pt20{
		padding-top: 20px
	}
	
	#addGnbCollectLog .excuteMode-radio .el-radio__input{
		margin-top:-2px;
	}
	#addGnbCollectLog .deviceTip {
		position:relative;
	}
	#addGnbCollectLog .deviceTip .deviceNum-tip {
		font-size:12px;
		color:#4D84FF;
		position:absolute;
		top:44px;
		left:100px;
	}
	#addGnbCollectLog .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#addGnbCollectLog .titleStyML{
		margin-left: 20px;
	}
</style>

<!--新建日志收集任务页面 -->
<div class="panelDefault" id='addGnbCollectLog'>
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" class="pt20">
	 	<div class="group-title not-extend deviceTip titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			<span class="deviceNum-tip" v-show="showDeviceNum"><%=rb.getString("ZuiDuoXuanZeSheBei5")%></span>
		</div>
		<el-pairgrid :limit="limitNum" class="pairgridBox" :id="'select_device_list'" :rownumber="true" ref="cpairgrid" @selection-change='selectChange' 
					:left-url="leftUrl" :height="height" row-key="small_cell_code" :query-params="queryForm" style="margin-left:45px"
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
				<div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
                    <el-query type="normal" @query="query" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>  
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
		<div v-show="false" class="alarmBottomLine"></div>
		<!-- 执行方式 -->
		<div v-show="false">
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("ZiKaiZhanZhiXingFangShi")%></span>
			</div>
			<div style="padding: 10px 30px;">
				<div class="el-textarea__inner" style='border:none;'>
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

					<el-form-item style='display:inline-block;margin:-15px 0 -15px 30px;' label='<%=rb.getString("FenZhongZhouQi") %>'>
						
					</el-form-item>
					<el-form-item style="display:inline-block;margin:10px 0px;" prop="reportPeriod">
						<el-select v-model="ruleForm.reportPeriod" size="mini" :disabled="setTimeEnable">
							<el-option label='15' value='900'></el-option>
							<el-option label='30' value='1800'></el-option>
							<el-option label='60' value='3600'></el-option>
						</el-select>
					</el-form-item>

				</div>
			</div> 
		</div>	
	</el-form>
</div>

<script type="text/javascript">

var colectLogs = new Vue({
	el:'#addGnbCollectLog',
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
						
						if(end_time > start){
							callback(new Error('<%=rb.getString("ShiJianFanWei")%>'));
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
		}
		return{
			queryForm:{
				isGnb: 1,
				timeZone: timeZone,
				search_text: '',
				like_fields: 'serial_number,host_name'
			},
			params:'',
		    rowData:[],
		    leftUrl:'',
		    selection:'',
		    setTimeEnable:true,
		    //新建收集任务-表单默认项
		    ruleForm:{
				cellCodes:'', //设备方式
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
		    deviceTitle:['<%=rb.getString("SheBeiLieBiao")%>','<%=rb.getString("YiXuan")%>'],
		    limitNum:'',
		    pageSize:'100',
		    height:'70%',
		    width:'100%',
		    showTip: true,
		    modal: false,
		    queryButton:'<%=rb.getString("ChaXun")%>',
		    resetButton:'<%=rb.getString("ChaXunChongZhi")%>',
		    showDeviceNum: false,
		    pickerOptions:{
				disabledDate(time){
					//周期上报选择时间范围：当前时间之前的日期为置灰状态
					return time.getTime()< Date.getNow()-8.64e7;
				}
			},
		}		
	},
	methods:{ 
		init:function(){
			var vm = this;

			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				selectType: 'deviceGroup',
				isGnb: 1
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			
			vm.leftUrl = '${ctx}/system/device/enodeb/queryENBInfoPageListForSelect.action'
			vm.$refs.cpairgrid.reload();

			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				selectType : 'deviceGroup',
				isGnb: 1
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
		},
		/**
		 * 查询
		 * @param val:输入的参数
		*/
		query(val){ 
            var vm = this;
            vm.queryForm.search_text = val;
		},
		/**
		 * 表格全选
		 * @param selection:选择的数据
		*/
		selectChange(selection){ 
			this.selection = selection
		},
		/**
		 * 点击确定函数
		 * @param pane:保存时候判断的参数 是first或者 second
		*/
		submit(pane){ 
	    	var vm = this;
	    	var message = '';
            // 防止多次提交
            if(gnbLogsVue.slideSubmitLoading)return

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {isGnb: 1};

					params.serial_number = vm.ruleForm.serial_number;//设备编码 				    				
					params.timeZone = timeZone;
					params.isReboot = "false";//是否重新启动
					params.device_code = vm.ruleForm.cellCodes;
					params.device_type = 'eNB';
					var dateTime = vm.ruleForm.time;  	        			
					//执行方式： Periodically 定时执行--开始时间，结束时间
					params.execute_type = vm.ruleForm.execute_type;
					
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
					
					url = '${ctx}/cell/collect/goImmediateCollectLogFile.action';
	    	    	message = '<%=rb.getString("ChengGong")%>'
                    gnbLogsVue.slideSubmitLoading = true;
	    			axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message: data.message,
	    						type:'success',
	    					})
                            eventBus.$emit('close-collect')
	    				}else if(data.responseCode == "901"){
		    				vm.$message.error(data.message);
                            gnbLogsVue.slideSubmitLoading = false;
		    			}else if(data.responseCode == "401"){//存在未完成的任务时，再次创建任务失败，给出提示
		    				vm.$message.error(data["message"]);
                            gnbLogsVue.slideSubmitLoading = false;
		    			}else{
		    	 			vm.$message.error('<%=rb.getString("ShouJiShiBai")%>');
                            gnbLogsVue.slideSubmitLoading = false;
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
			this.ruleForm.time = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('time');
		},
	},
	watch:{
		selection(){ // 监听选择
			var data = this.$refs.cpairgrid.getData();
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
			}
			this.$refs.ruleForm.validateField('time')
		}
	},
	mounted(){
		this.init();
		eventBus.$off('collect-submit').$on('collect-submit',this.submit);
	}
})
</script>