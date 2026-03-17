<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#egwUpgradeAddTask .modeItem .el-radio{
	display:block;
	margin-left:0px;
	margin-bottom:20px;
} 
#egwUpgradeAddTask .modeItem{
	margin-top:20px;
}
#egwUpgradeAddTask .tableDiv{
	height:15px;
	width:auto;
}
#egwUpgradeAddTask .tableSelectCls{
	cursor: pointer;
	text-align: center;
}
#egwUpgradeAddTask .tableSelectCls .el-icon::before{
	color: #1EBC1E;
}
#egwUpgradeAddTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#egwUpgradeAddTask .deviceItem{
	display:inline-block;
	margin-right:60px;
}
#egwUpgradeAddTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#egwUpgradeAddTask .titleStyML{
	margin-left: 20px;
}
#egwUpgradeAddTask .filterFile{
	margin-left: 15px;
}
#egwUpgradeAddTask .filterFile .el-checkbox__label {
	font-size:12px;
	font-weight:normal;
	margin-left:0;
	color: #4D84FF;
}
#egwUpgradeAddTask .deviceTableBox{
	margin:10px 0px 0px 45px;
	display: flex;
}
#egwUpgradeAddTask .deviceSpecifiedBox{
	width: 200px;
	height: 370px;
	border:1px solid #E9E9E9;
	border-right: none;
}
#egwUpgradeAddTask .deviceSpecifiedTitle{
	height: 36px;
	width: 200px;
	font-size: 12px;
	line-height: 36px;
	text-align: center;
	background: #F6F7FB;
	box-sizing: border-box;
	border-bottom:1px solid #E9E9E9;
}
#egwUpgradeAddTask .specifiedTypeBox{
	flex: 1;
	padding-top: 30px;
	padding-left: 40px;
}
#egwUpgradeAddTask .specifiedTypeBox .el-radio__label{
	font-size: 12px !important;
}
#egwUpgradeAddTask .specifiedTypeBox .el-radio+.el-radio{
	margin-left: 0px;
	display: block;
}
#egwUpgradeAddTask .egwDeviceTableBox{
	width: calc(100% - 0px)!important;
	font-size: 12px !important;
}
#egwUpgradeAddTask .egwDeviceTableBox .pairgrid-right{
	top:40px!important;
	height: calc(100% - 40px)!important;
}
#egwUpgradeAddTask .egwDeviceTableBox .el-pairgrid-title{
	top:15px!important;
	right: 15px!important;
}
#egwUpgradeAddTask .egwDeviceTableBox .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
#egwUpgradeAddTask .tableTitles{
	margin-left: 45px;
	margin-top: 20px;
	font-size: 14px;
}
#egwUpgradeAddTask .el-form-item{
	display: flex;
	align-items: center;
}
#egwUpgradeAddTask .el-form-item__content{
	margin-left: unset!important;
	width: 100%;
}
</style>

<%-- 新建egw升级任务 --%>
<div id="egwUpgradeAddTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:20px;' label-width="165px" label-position="left"  :hide-required-asterisk=true>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskName' style='margin-left:45px;margin-top:20px;'>
			<el-input maxlength=100 :disabled='viewFlag' v-model="ruleForm.taskName" size="mini" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div>
			<div class="group-title not-extend titleStyML" >
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			</div>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' style="margin:20px 0px 0px 45px;">
				<el-radio-group  v-model="ruleForm.selectAll" :disabled="viewFlag" @change="selectAllChange" style="padding-top: 5px;">
					<el-radio border  label="true"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio border  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<div class="deviceTableBox">
				<!--<div class="deviceSpecifiedBox">
					<div class="deviceSpecifiedTitle"><%=rb.getString("SheBeiZhiDing")%></div>
					<div class="specifiedTypeBox">
						<el-radio-group  v-model="ruleForm.selectAll" :disabled="viewFlag" @change="selectAllChange">
							<el-radio  label="true" style="margin-bottom:26px;"><%=rb.getString("QuanBu")%></el-radio>
							<el-radio  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
						</el-radio-group>
					</div>
				</div>-->
				<div class="egwDeviceTableBox">
					<el-pairgrid 
						:id="'select_device_list'" 
						v-if="showPairGrid && ruleForm.selectAll == 'false'" 
						style="margin-right:45px;" query-name="egwSn" 
						:rownumber="true" 
						ref="egwDevicePairgrid"  
						@selection-change='selectChange' 
						:right-url="rightUrl" :left-url="leftUrl" 
						:height="height" row-key="egwCode" 
						:query-params="queryParams" :title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("eGWBianMa")%>'}"
					 >
						<template slot="left">
							<el-table-column type="selection" width="45"></el-table-column>
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
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-query @query="query" type="normal"  placeholder="<%=rb.getString("eGWBianMa")%>"></el-query>
						</template>
						<template slot='right'>
								<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						:id="'all_device_list'" 
						v-if="ruleForm.selectAll == 'true'" 
						:query-params="queryParams" 
						style="border:1px solid #E9E9E9;margin-right:45px;" 
						ref="egwDevicePairgrid"
						:url="leftUrl" 
						:height="height" 
						pagination="true" 
						>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-query @query="query" type="normal"  placeholder="<%=rb.getString("eGWBianMa")%>"></el-query>
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
						<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="fileParams">
							<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
							<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
							<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
							<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
							<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
						</el-ctable>
					</div>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-left:45px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div class="tableTitles">
			<span ><%=rb.getString("WenJianLieBiao")%></span>
		</div>
		<div class='fileContent' style='margin:10px 45px 0px 45px;border:1px solid #E9E9E9;'>
			<el-ctable :id="'select_file_list'" row-key="id" :readonly='viewFlag' ref="fileTables" :url='fileDateUrl' :default-checked="defaultChecked" @row-click="rowClickUpgrade" @load-success='loadSuccess' :height="height" pagination="true" :query-params="fileParams">
				<el-table-column label='<%=rb.getString("XuanZe")%>' width="60" >
					<template slot-scope="scope" >
	              		<div class="tableSelectCls">
							<span class='tableDiv el-icon el-icon-status-yes selected-status'></span>
						</div>
	            	</template>
				</el-table-column>
				<el-table-column prop="id" v-if="false"></el-table-column>
				<el-table-column prop="file_name" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>"  width="300"></el-table-column>
				<el-table-column prop="version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" width="250"></el-table-column>
				<el-table-column prop="product_type" show-overflow-tooltip label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" width="120"></el-table-column>
				<el-table-column prop="file_size" label="<%=rb.getString("WenJianDaXiao")%>" width="150"></el-table-column>
				<el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" width="180"></el-table-column>
				<el-table-column prop="description" show-overflow-tooltip label="<%=rb.getString("MiaoShu")%>"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='fileId' style='margin-left:45px;'>
			<el-input v-model='ruleForm.fileId' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item  prop='status'>
				<el-radio-group v-model="ruleForm.status" style='margin-top:17px;' :disabled='viewFlag'>
					<el-radio border label="active" style='margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio border label="suspend"><%=rb.getString("GuaQi")%></el-radio>
					<el-radio border label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;' >
				<el-date-picker style='margin-top:10px;vertical-align:middle;' value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="viewFlag||setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
</div>

<script type="text/javascript">
new Vue({
	el:'#egwUpgradeAddTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/egw/softwareUpgrade/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskName.trim()
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("RenWuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.status !== 'timing'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		};
		var validateCellCodes = (rule,value,callback) => {
			if(this.ruleForm.selectAll == 'true'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		};
		return {
			deviceData:{}, // 新建任务接收的设备信息
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			fileParams:{
				timeZone:timeZone,
			},
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			deviceSelect:[],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskName:'${addTaskName}',
				cellCodes:'',
				fileId:'',
				fileName:'',
				status:'active',
				exetime:'',
				selectAll:'false',
				version:''
			},
			rules:{
				taskName:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{validator:validateCellCodes,trigger:'change'}
				],
				fileId:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			queryParams:{
				search_text:'',
				timeZone:timeZone
			},
			taskId:'',
			operationType:'',
			defaultTaskName:'',
			showPairGrid:true,
			fileDateUrl:'',
			viewFlag:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.getNow()-8.64e7;
				}
			},
		}
	},
	methods:{ 
		/**
		 * 初始化
		 * @param id:number     egw升级任务id
		 * @param type:string   任务类型  'addTask' -- 新增  'taskView'-- 查看  'taskEdit' -- 修改
		*/
		init(id,type){ 
			var vm = this;
			vm.taskId = id ;
			vm.operationType = type;
			if(type == 'taskView'){
				vm.viewFlag = true;
				vm.showPairGrid = false;
			}else if(type == 'addTask'){
				if(egwUpgrade.selection.length > 0){
					vm.$refs.egwDevicePairgrid.appendCheckedRows(egwUpgrade.selection);
				}
			}
			vm.leftUrl="${ctx}/egw/monitor/getEgwMonitorPageList.action";
			if(type !== "addTask"){
				vm.$nextTick(function(){
					vm.rightUrl="${ctx}/egw/softwareUpgrade/getSelectedEGWInfo.action?taskId="+vm.taskId;
				})
				vm.getTaskDateInfo();
			}else{
				vm.fileDateUrl = '${ctx}/egw/softwareFile/querySoftwareFilePageList.action';
			}
			initForm(vm.$refs.ruleForm);
		},
		// 设备执行类别  1 全部执行 2 指定执行
		selectAllChange(val){
			var vm = this;

			if(val == 'true'){
				vm.rightUrl = '';
			}else{
				vm.rightUrl="${ctx}/egw/softwareUpgrade/getSelectedEGWInfo.action?taskId="+vm.taskId;
			}
		},
		// 模糊查询
		query(val){
			var vm = this;
			vm.queryParams.search_text = val;
		},
		// egw设备选择事件
		selectChange(selection){
			var vm = this;
			vm.selection = selection;
		},
		// 升级文件选择事件
		rowClickUpgrade(row){
	        this.rowData = row;
	    },
		// 获取egw升级任务详情
		getTaskDateInfo(){
			var vm = this,
				params={
					taskId:vm.taskId,
					timeZone:timeZone
				};
			
			axios.post('${ctx}/egw/softwareUpgrade/getTaskInfo.action',stringify(params)).then(function(response){
				let data = response.data;
				Object.assign(vm.ruleForm, data);
				if(data.selectAll == ''){
					vm.ruleForm.selectAll = 'false';
				}
				if(data.status == 'timing'){
					vm.ruleForm.exetime = data.time;
				}else{
					vm.ruleForm.exetime = '';
				}
				vm.fileDateUrl = "${ctx}/egw/softwareFile/querySoftwareFilePageList.action"+"&fileId="+data.fileId+"&timeZone="+timeZone;
				vm.defaultChecked = [data.fileId+''];
				vm.defaultTaskName = data.taskName;
				
			}).catch(function(error){})
		},
		// 提交
		submit(){
	    	var vm = this;
             // 防止多次提交
             if(egwUpgrade.slideSubmitLoading)return

			if(vm.operationType == 'taskEdit'){
				if(!isFormChanged(vm.$refs.ruleForm)){
					vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						type:'warning'
					}).then().catch();
					return;
				}
			}
			
	    	var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},urls='';
	    			params.timeZone = timeZone;
	    			params.egwCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskName;
					params.selectAll = vm.ruleForm.selectAll;
	    			params.status = vm.ruleForm.status;

	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.fileId = vm.ruleForm.fileId;
					if(vm.operationType == 'addTask'){
						urls = '${ctx}/egw/softwareUpgrade/addTask.action';
					}else{
						params.taskId = vm.taskId
						urls = '${ctx}/egw/softwareUpgrade/updateSoftwareInfos.action';
					}
					vm.saveTask(urls,params,message);
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(urls,params,message){
			var vm = this;
           
            egwUpgrade.slideSubmitLoading = true;
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    egwUpgrade.$refs.upgradedeviceTable.clearSelection();
                    eventBus.$emit('hide-egwUpgrade-slide');
                    isJumpToPage = ''
				}else{
					vm.$message.error(data["message"]);
                    egwUpgrade.slideSubmitLoading = false;
				}
			}).catch(function(error){})
			
		},
		// 取消
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
			if(vm.operationType !== 'taskView'){
				if(isFormChanged(this.$refs.ruleForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('hide-egwUpgrade-slide');
						isJumpToPage = ''
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('hide-egwUpgrade-slide')
					isJumpToPage = ''
				}
			}else{
				eventBus.$emit('hide-egwUpgrade-slide');
				isJumpToPage = ''
			}
			
		},
		// egw 升级文件表格加载成功回调
		loadSuccess(){
			var vm = this;
			var tb = vm.$refs.fileTables;
			var rows = tb.tbData;
			if(vm.operationType !== 'addTask'){
				vm.updateRow(tb.tbData,tb);
			}else{
				if(isJumpToPage){
					rows.map(function(item){
						if(item.id == isJumpToPage.vid){
							tb.setCurrentRow(item)
						}
					})
				}
			}
			
		},
		updateRow(rows,tb){
			var vm = this;
			axios.post('${ctx}/egw/softwareUpgrade/getTaskInfo.action',stringify({
				taskId : vm.taskId,
				timeZone : timeZone
			})).then(function(response){
				var data = response.data;
				rows.map(function(item){
					if(item.id == data.fileId){
						tb.setCurrentRow(item)
					}
				})
			})
		},
		// 定时时间失焦事件
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		
	},
	watch:{
		rowData(newVal){
			var vm = this;

			this.ruleForm.fileId = newVal.id;
			this.ruleForm.fileName = newVal.file_name;
			this.ruleForm.version = newVal.version;
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		selection(){
			var vm = this,
				data = this.$refs.egwDevicePairgrid.getData();
				cellCodes = '',cellCodeList = [];
			if(data.length != 0){
				data.map(function(item){
					cellCodeList.push(item.egwCode);
				})
			}
			cellCodes = cellCodeList.join(',');
			vm.ruleForm.cellCodes = cellCodes;
		}
	},
	mounted(){
		eventBus.$off('egw-upgrade-taskInit').$on('egw-upgrade-taskInit',this.init);
		eventBus.$off('egw-upgrade-addSubmit').$on('egw-upgrade-addSubmit',this.submit);
		eventBus.$off('egw-upgrade-addCancel').$on('egw-upgrade-addCancel',this.cancel);
	}
})
</script>