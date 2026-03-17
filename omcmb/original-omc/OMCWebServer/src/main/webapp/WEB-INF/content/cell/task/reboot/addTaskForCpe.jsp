<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#addCpeRebootTask .el-icon-time{
		line-height:1;
	}
	#addCpeRebootTask .timeItem .el-input__inner{
		width:220px;
	}
	#addCpeRebootTask .ml45{
		margin-left: 45px
	}
	#addCpeRebootTask .mt20{
		margin-top: 20px
	}
	
	#addCpeRebootTask .taskInput{
		width:680px;
		height:28px;
		line-height:28px;	
	}

	#addCpeRebootTask .el-form-item__label{
		line-height:26px;
		width:140px;
		text-align:left;
	}
	#addCpeRebootTask .modeItem .el-radio{
		display:inline-block;
		margin-left:0px;
	} 
	#addCpeRebootTask .modeItem{
		margin-top:30px;
	}
	#addCpeRebootTask .tableInfo .el-form-item__error{
		margin-left: 35px
	}
	#addCpeRebootTask .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#addCpeRebootTask .titleStyML{
		margin-left: 20px;
	}
	#addCpeRebootTask  .editButton{
		position: absolute;
		top: 15px;
		right: 140px;
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
	}
	#addCpeRebootTask .editButton i{
		font-size:14px !important;
	}
	#addCpeRebootTask .editButton span{
		font-size:12px;
	}
	.cpeRebootDialog .el-icon-circle-info:before{
		color:#CFCFCF;
	}
	#addCpeRebootTask .cpeDeviceTableBox .el-pairgrid-title{
		top:15px!important;
		right: 15px!important;
	}
</style>

<!-- 新建重启任务 -->
<div id="addCpeRebootTask" style="margin-top:20px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' class="ml45 mt20">
			<el-input maxlength=50 :disabled='showName || isReadonly' v-model="ruleForm.taskname" size="mini" class="taskInput"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<div class="cpeDeviceTableBox" style='position:relative'>
			<el-pairgrid id="enb_select_list" v-if="!isReadonly"
				ref="cpairgrid" 
				@selection-change='selectChange'  
				:right-url="rightUrl" 
				:left-url="leftUrl" 
				:height="height" 
				row-key="cpe_code" 
				:query-params="queryParams" 
				:title="deviceTitle" 
				:messages="{placeholder:'<%=rb.getString("CPEBianMa")%>'}" 
				style='margin:10px 45px 0px 45px;'
				class='tableInfo'>
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
					<el-table-column prop='host_name' label='<%=rb.getString("CpeName")%>'></el-table-column>
					<el-table-column prop='mac_address' label='<%=rb.getString("CPEMacAddress")%>'></el-table-column>
					<el-table-column prop='pci' label='PCI'></el-table-column>
				</template>
				<template slot='toolbar'>
					<!-- 新建任务 基站列表的高级选择 -->
					<el-form :model='queryForm' ref="queryForm" label-position="top">
						<div style="display: flex;align-items: center;">
							<el-query type="normal" @query="query"  :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"></el-query>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("SheBeiZu")%>'
								v-model="queryForm.group_id"
								:list="groupOptions.map(item=>{return {label:item.group_name,value:item.id}})"
								@check-change="advanceQuery">
							</el-popfilter>
							<div class="pop-filter-clear" style="margin: 0 5px;" 
								@click="resetQuery">
								<%=rb.getString("QingKongShaiXuan")%>
							</div>
						</div>
					</el-form>
					<div class="editButton" size="mini" @click="addBatchSn">
						<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
						<span><%=rb.getString("PiLiangShuRu")%></span>
					</div>
				</template>
				<template slot='right'>
					<el-table-column prop='serial_number' label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
					<el-table-column prop='host_name' label='<%=rb.getString("CpeName")%>'></el-table-column>
				</template>
			</el-pairgrid>
	
			<el-ctable v-if="isReadonly" height="300"
				:url="rightUrl">
				<el-table-column prop='serial_number' label='<%=rb.getString("CPEBianMa")%>'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("CpeName")%>'></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='cellCodes' style="margin-left:45px;">
			<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;margin-left:45px;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status" :disabled='showType || isReadonly' @change="statusChange">
				<el-radio label="active" style='margin-right:80px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio label="suspend" style='margin-right:80px;'><%=rb.getString("GuaQi")%></el-radio>
				<el-radio label="timing" style='margin-bottom:0px;margin-right:10px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='time' style='display:inline-block;margin-bottom:0px;margin-left:15px' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.time' :disabled="setTimeEnable || isReadonly" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
	</el-form>
	<el-dialog class='cpeRebootDialog' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label>MAC</label>
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
new Vue({
	el:'#addCpeRebootTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else {
				axios.post('${ctx}/task/reboot/taskNameExistForCpe.action',stringify({
					taskName: vm.ruleForm.taskname.trim(),
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
		 var validatorNum = (rule,value,callback) => {
			 var serialNumber = value, 
				 list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
					return item.length > 0;
				}),
				
				temp = /^[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}:[A-Fa-f0-9]{2}$/,
				noColTemp = /^([A-Fa-f0-9]{2}){6}$/;
				
			    if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
				}else {
					var nameFlag = list.every(function(item,index){
						return (temp.test(item) ||  noColTemp.test(item)) 
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueMac")%>'));
					}
				}
			};
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.status !== 'timing'){
				callback()
			}else{
				if(value == '' || value==null){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		}
		return {
			isReadonly: false,
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			groupOptions:[],
			pageSize:10,
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskId: '',
				taskname: '${addTaskName}',
				cellCodes: '',
				status: 'active',
				time: '',
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
				],
				time:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			queryParams:{
				timeZone:timeZone,
				isShowSlave:false,
				serial_nubmer:'',
				host_name:'',
				search_text:'',
				group_id:'',
				productValue:'',
				pci:''
			},
			queryForm:{
				group_id:'',
			},
			taskId:'',
			defaultTaskName:'',
			task_type:'add',
			showPairGrid:true,
			showName:false,
			showMode:false,
			showType:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			rebootTypeArr:[
		    	{"text":"<%=rb.getString("XiTong")%>",value:"reboot_system"},
				{"text":"<%=rb.getString("JinCheng")%>",value:"reboot_stk"},
				{"text":"<%=rb.getString("ZiJi")%>",value:"reboot_ru"}
				],
			productType:[],
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:"<%=rb.getString("TianJia")%>"
		}
	},
	methods:{
		getInfo(row) {
			var vm = this,
				params = {
					taskId: row.TASK_ID,
					timeZone: timeZone
				};

			axios.post('${ctx}/task/reboot/getTaskInfo.action', stringify(params)).then(function(res){
				var data = res.data;
				
				if(data) {
					vm.defaultTaskName = data.TASK_NAME;
					Object.assign(vm.ruleForm, {
						taskId: row.TASK_ID,
						taskname: data.TASK_NAME,
						status: data.EXECUTE_TYPE,
						time: data.EXECUTE_TYPE == 'timing'?data.TIME:''
					});
				}
			}).catch(function(error){});
		},
		getGroups() {
			var vm = this;

			axios.post('${ctx}/system/deviceGroup/queryDeviceGroupNameAndId.action').then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){});
		},
		init(rows){
			var vm = this;
			vm.$refs.cpairgrid.clear();
			vm.leftUrl = '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3';
			vm.$refs.cpairgrid.reload();
			if(rows && rows.length) {
				vm.$refs.cpairgrid.appendCheckedRows(rows);
			}
			
			vm.getGroups();
		},
		modifyTask(row, type) {
			var vm = this,
				taskId = row.TASK_ID;

			vm.$refs.cpairgrid.clear();
			// 修改任务
			if(type == 'edit') {
				vm.leftUrl = '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3';
				vm.rightUrl = '${ctx}/task/reboot/getSelectedCpeInfo.action?taskId='+ taskId +'&type=modify';
				vm.getInfo(row);
			}
			// 查看任务
			if(type == 'view') {
				vm.rightUrl = '${ctx}/task/reboot/getSelectedCpeInfo.action?taskId='+ taskId;
				vm.isReadonly = true;
				vm.getInfo(row);
			}

			vm.getGroups();
		},
		/**
		 * 模糊查询
		 * @param val:查询参数
		*/
		query(val){
			this.queryParams.search_text = val;
			this.$refs.cpairgrid.reload();
		},
		advanceQuery(){ // 高级搜索查询
			var vm = this;
			Object.assign(vm.queryParams,vm.queryForm);
		},
		resetQuery(){ // 普通搜索查询按钮
			var vm = this,
				params = {
					group_id:'',
				};
			Object.assign(vm.queryForm,params);
			Object.assign(vm.queryParams,params);
		},
		/**
		 * 全选操作
		 * @param selection:选择的数据
		*/
		selectChange(selection){
			this.selection = selection
		},
		rowClickUpgrade(row){
	        this.rowData = row;
	    },
		// 执行方式改变事件
		statusChange(val){
			var vm = this;
			if(val !== 'timing'){
				vm.ruleForm.time = '';
				vm.$refs.ruleForm.clearValidate('time')
			}
		},
		submit(){ // 确定按钮 
	    	var vm = this,
                message = '';
            // 防止多次提交
            if(vmCPEReboot.slideSubmitLoading)return

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			params.timeZone = timeZone;
	    			params.cellCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskname;
	    			params.status = vm.ruleForm.status;
	    			params.taskId = vm.ruleForm.taskId;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.time;
	    			}
	    			
	    	    	url = '${ctx}/task/reboot/addTaskForCpe.action'
	    	    	message = '<%=rb.getString("ChengGong")%>'
	    	    	vmCPEReboot.slideSubmitLoading = true;
	    			axios.post(url,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            eventBus.$emit('cancel-reboot')
	    				}else{
	    					vm.$message.error(data["message"]);
                            vmCPEReboot.slideSubmitLoading = false;
	    				}
	    			})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		cancel(){ // 关闭弹窗
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
					eventBus.$emit('hide-reboot')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-reboot')
			}
		},
		setTime(){
			this.ruleForm.time = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('time');
		},
		addBatchSn(){
			this.listVisible = true;
		},
		closeBatchSn(){
			this.listVisible = false;
			this.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '',
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			var params = {
				type: 'mac',
				macs : list.join(";")
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/CPE/queryCpeInfoByMacs.action",stringify(params)).then((res)=>{
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
		}
	},
	watch:{
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var cellCodes = '';
			if(data.length != 0){
				data.map(function(item){
					cellCodes += item.cpe_code + ","
				})
			}
			this.ruleForm.cellCodes = cellCodes
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('time')
		},
	},
	mounted(){
		//this.init();
		eventBus.$off('hander-ok').$on('hander-ok',this.submit);
		eventBus.$off('hander-cancel').$on('hander-cancel',this.cancel);
		eventBus.$off('modify-task').$on('modify-task',this.modifyTask);

		eventBus.$off('addRows').$on('addRows',this.init);
	}
})
</script>
